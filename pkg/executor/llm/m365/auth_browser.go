package m365

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// This file drives the Azure AD interactive login headlessly with Playwright,
// filling stored credentials and a TOTP code, and captures the authorization
// code from the nativeclient redirect. A persistent browser profile keeps AAD
// session/device cookies across runs so later logins are SSO-silent and present
// as a familiar (low-risk) device.

// loginUserAgent is a coherent Linux Chrome UA that avoids the default
// HeadlessChrome automation tell.
const loginUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"

const (
	loginViewportWidth  = 1280
	loginViewportHeight = 800
	authCodeTimeout     = 45 * time.Second
	fillTimeoutMs       = 20000
	pressDelayMs        = 20
	tileTimeoutMs       = 5000
	stepTimeoutMs       = 8000
)

// playwrightRunFunc and playwrightInstallFunc are playwright.Run/Install,
// overridable in tests so the auto-install retry path can be exercised
// without a real network download.
//
//nolint:gochecknoglobals // test-replaceable hooks
var (
	playwrightRunFunc     = playwright.Run
	playwrightInstallFunc = playwright.Install
)

func browserProfileDir() string { return resolveFile("M365_BROWSER_PROFILE", "browser-profile") }

// startPlaywright starts the Playwright driver, installing the Chromium
// driver/browser on first use if it's missing -- so the m365 backend works
// without a manual `playwright install chromium` step.
func startPlaywright() (*playwright.Playwright, error) {
	pw, err := playwrightRunFunc()
	if err == nil {
		return pw, nil
	}
	fmt.Fprintln(os.Stderr, "m365: installing Playwright Chromium (first use only)...")
	if instErr := playwrightInstallFunc(&playwright.RunOptions{Browsers: []string{"chromium"}}); instErr != nil {
		return nil, fmt.Errorf("m365: start playwright (auto-install failed: %w) after: %w", instErr, err)
	}
	pw, err = playwrightRunFunc()
	if err != nil {
		return nil, fmt.Errorf("m365: start playwright after auto-install: %w", err)
	}
	return pw, nil
}

// resolveChromiumPath returns an explicit or system Chromium, or "" to let
// Playwright use its bundled browser.
func resolveChromiumPath() string {
	if p := os.Getenv("CHROMIUM_PATH"); p != "" {
		return p
	}
	for _, bin := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		if p, err := exec.LookPath(bin); err == nil {
			return p
		}
	}
	return ""
}

// browserLogin opens authURL, drives the login form, and returns the captured
// authorization code.
//
//nolint:gocognit // browser setup, request capture, and timeout race in one flow
func browserLogin(ctx context.Context, authURL string, creds *Credentials) (string, error) {
	kdeps_debug.Log("enter: browserLogin")

	pw, err := startPlaywright()
	if err != nil {
		return "", err
	}
	defer func() { _ = pw.Stop() }()

	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-blink-features=AutomationControlled",
		},
		UserAgent:  playwright.String(loginUserAgent),
		Locale:     playwright.String("en-GB"),
		TimezoneId: playwright.String("Europe/Copenhagen"),
		Viewport:   &playwright.Size{Width: loginViewportWidth, Height: loginViewportHeight},
	}
	if p := resolveChromiumPath(); p != "" {
		opts.ExecutablePath = playwright.String(p)
	}

	context, err := pw.Chromium.LaunchPersistentContext(browserProfileDir(), opts)
	if err != nil {
		return "", fmt.Errorf("m365: launch browser: %w", err)
	}
	defer context.Close()

	_ = context.AddInitScript(playwright.Script{
		Content: playwright.String(`Object.defineProperty(navigator, "webdriver", { get: () => undefined });`),
	})

	// The nativeclient ?code= exists only transiently, so capture it from the
	// navigation request rather than waiting for the URL to settle.
	codeCh := make(chan string, 1)
	context.OnRequest(func(req playwright.Request) {
		u := req.URL()
		if strings.Contains(u, "/oauth2/nativeclient") && strings.Contains(u, "code=") {
			if parsed, perr := url.Parse(u); perr == nil {
				if c := parsed.Query().Get("code"); c != "" {
					select {
					case codeCh <- c:
					default:
					}
				}
			}
		}
	})

	var page playwright.Page
	if pages := context.Pages(); len(pages) > 0 {
		page = pages[0]
	} else {
		if page, err = context.NewPage(); err != nil {
			return "", fmt.Errorf("m365: new page: %w", err)
		}
	}

	if _, gerr := page.Goto(authURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); gerr != nil {
		return "", fmt.Errorf("m365: navigate to login: %w", gerr)
	}

	// Drive the form concurrently with the code race: the SSO-silent path may
	// redirect through with the code before any form step appears.
	go driveAzureLogin(page, creds)

	select {
	case code := <-codeCh:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(authCodeTimeout):
		return "", errors.New("m365: timed out waiting for auth code")
	}
}

// visible returns a Locator scoped to the first visible match of selector.
func visible(page playwright.Page, selector string) playwright.Locator {
	return page.Locator(selector + ":visible").First()
}

// isVisibleSoon reports whether selector becomes visible within timeoutMs.
func isVisibleSoon(page playwright.Page, selector string, timeoutMs float64) bool {
	err := visible(page, selector).WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeoutMs),
	})
	return err == nil
}

// fillVerified fills a field and retypes it if the value did not land (the
// converged AAD page keeps hidden duplicate inputs that swallow a naive fill).
func fillVerified(page playwright.Page, selector, value string) error {
	loc := visible(page, selector)
	if err := loc.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(fillTimeoutMs),
	}); err != nil {
		return err
	}
	_ = loc.Click()
	_ = loc.Fill(value)
	got, _ := loc.InputValue()
	if got != value {
		_ = loc.Fill("")
		_ = loc.PressSequentially(
			value,
			playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(pressDelayMs)},
		)
		got, _ = loc.InputValue()
	}
	if got != value {
		return fmt.Errorf("m365: field %q still empty after refill", selector)
	}
	return nil
}

// clickSubmit clicks the visible primary submit button.
func clickSubmit(page playwright.Page) {
	_ = page.Locator(`input[type="submit"]:visible, button[type="submit"]:visible`).First().Click()
}

// clickAccountTileIfPresent handles the SSO "Pick an account" picker, returning
// true if a tile was clicked.
func clickAccountTileIfPresent(page playwright.Page, email string) bool {
	if err := visible(page, "#tilesHolder").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(tileTimeoutMs),
	}); err != nil {
		return false
	}
	tile := page.Locator(
		`[data-test-id="` + strings.ToLower(email) + `"]:visible, ` +
			`#tilesHolder .tile [role="button"][data-test-id]:not([data-test-id$="-menu-dots"]):visible`,
	).First()
	if err := tile.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(tileTimeoutMs),
	}); err != nil {
		return false
	}
	return tile.Click() == nil
}

// driveAzureLogin fills each login step that actually appears. With a persistent
// profile a returning session is SSO-silent, so email/password/TOTP prompts may
// not show at all; each step is therefore optional.
func driveAzureLogin(page playwright.Page, creds *Credentials) {
	kdeps_debug.Log("enter: driveAzureLogin")

	picked := clickAccountTileIfPresent(page, creds.Email)

	if !picked && isVisibleSoon(page, `input[name="loginfmt"]`, stepTimeoutMs) {
		if fillVerified(page, `input[name="loginfmt"]`, creds.Email) == nil {
			clickSubmit(page)
		}
	}

	if isVisibleSoon(page, `input[name="passwd"]`, stepTimeoutMs) {
		if fillVerified(page, `input[name="passwd"]`, creds.Password) == nil {
			clickSubmit(page)
		}
	}

	if isVisibleSoon(page, `input[name="otc"]`, stepTimeoutMs) {
		if code, err := totpNow(creds.MFASecret); err == nil {
			if fillVerified(page, `input[name="otc"]`, code) == nil {
				clickSubmit(page)
			}
		}
	}

	// "Stay signed in?" — accepting persists the cookie that makes the next login
	// SSO-silent. Best-effort.
	_ = page.Locator("#idSIButton9:visible").
		Click(playwright.LocatorClickOptions{Timeout: playwright.Float(stepTimeoutMs)})
}
