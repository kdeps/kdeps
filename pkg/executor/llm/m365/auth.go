package m365

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// This file authenticates against Azure AD as a public client using the
// authorization-code + PKCE flow, caching the refresh token on disk so later
// calls refresh silently. The interactive leg is driven headlessly by a browser
// (see auth_browser.go) that fills stored credentials and a TOTP code. Tokens
// for different resources (chat vs Power Platform) are obtained by redeeming the
// same refresh token against different scopes.

const (
	clientID    = "c0ab8ce9-e9a0-42e7-b064-33d422df41f1"
	redirectURI = "https://login.microsoftonline.com/common/oauth2/nativeclient"

	// oidcScopes are appended to every request so the flow returns a refresh token.
	oidcScopes = "offline_access openid profile"

	// pkceVerifierBytes is the entropy size of the PKCE code verifier.
	pkceVerifierBytes = 32
	// tokenExpirySkewSec renews an access token this many seconds before expiry.
	tokenExpirySkewSec = 60
	// totpWindowWait lets a fresh TOTP window open before a login retry.
	totpWindowWait = 31 * time.Second
)

// chatScopes are the substrate scopes the chat WebSocket requires.
//
//nolint:gochecknoglobals // static list, read-only
var chatScopes = []string{
	"https://substrate.office.com/sydney/M365Chat.Read",
	"https://substrate.office.com/sydney/sydney.readwrite",
}

// authority is the AAD authority base URL. It is a var (not a const) only so
// tests can point the OAuth flow at a local server.
//
//nolint:gochecknoglobals // effectively constant; overridden only in tests
var authority = "https://login.microsoftonline.com/common"

// configDir is the on-disk directory holding the token cache and secrets.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "kdeps", "m365")
}

// resolveFile returns an env override or the default file under configDir.
func resolveFile(env, defaultName string) string {
	if p := os.Getenv(env); p != "" {
		return p
	}
	dir := configDir()
	_ = os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, defaultName)
}

func cacheFile() string   { return resolveFile("M365_CACHE_FILE", "token-cache.json") }
func secretsFile() string { return resolveFile("M365_SECRETS_FILE", "secrets.json") }

// cachedToken is one scoped access token with its absolute expiry.
type cachedToken struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"`
}

// tokenCache is the persisted auth state: one refresh token plus per-scope
// access tokens.
type tokenCache struct {
	RefreshToken string                 `json:"refreshToken,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Access       map[string]cachedToken `json:"access,omitempty"`
}

//nolint:gochecknoglobals // package-level cache guarded by cacheMu
var (
	cacheMu    sync.Mutex
	cacheState *tokenCache
	tokenMu    sync.Mutex // single-flights token acquisition
)

func loadCache() *tokenCache {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cacheState != nil {
		return cacheState
	}
	cacheState = &tokenCache{Access: map[string]cachedToken{}}
	data, err := os.ReadFile(cacheFile())
	if err == nil {
		var c tokenCache
		if json.Unmarshal(data, &c) == nil {
			if c.Access == nil {
				c.Access = map[string]cachedToken{}
			}
			cacheState = &c
		}
	}
	return cacheState
}

func saveCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cacheState == nil {
		return
	}
	data, err := json.MarshalIndent(cacheState, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cacheFile(), data, 0o600)
}

// scopeKey is a stable cache key for a scope set.
func scopeKey(scopes []string) string {
	s := append([]string(nil), scopes...)
	sort.Strings(s)
	return strings.Join(s, " ")
}

// tokenEndpoint returns the AAD v2 token URL.
func tokenEndpoint() string { return authority + "/oauth2/v2.0/token" }

// authorizeEndpoint returns the AAD v2 authorize URL.
func authorizeEndpoint() string { return authority + "/oauth2/v2.0/authorize" }

// pkce returns a fresh PKCE verifier and its S256 challenge.
func pkce() (string, string) {
	b := make([]byte, pkceVerifierBytes)
	_, _ = rand.Read(b)
	verifier := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

// buildAuthURL builds the authorize URL for scopes and returns it with the PKCE
// verifier that must be presented at code redemption.
func buildAuthURL(scopes []string) (string, string) {
	verifier, challenge := pkce()
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("response_mode", "query")
	q.Set("scope", strings.Join(scopes, " ")+" "+oidcScopes)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return authorizeEndpoint() + "?" + q.Encode(), verifier
}

// tokenResponse is the AAD token endpoint reply.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// postToken performs a token endpoint request and stores the resulting tokens.
func postToken(ctx context.Context, form url.Values, scopes []string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var tr tokenResponse
	if err = json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return "", err
	}
	if tr.Error != "" {
		return "", fmt.Errorf("m365: token endpoint: %s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", errors.New("m365: token endpoint returned no access token")
	}

	c := loadCache()
	cacheMu.Lock()
	if tr.RefreshToken != "" {
		c.RefreshToken = tr.RefreshToken
	}
	c.Access[scopeKey(scopes)] = cachedToken{
		AccessToken: tr.AccessToken,
		// Renew a minute early to avoid edge-of-expiry failures.
		ExpiresAt: time.Now().Add(time.Duration(tr.ExpiresIn-tokenExpirySkewSec) * time.Second).Unix(),
	}
	cacheMu.Unlock()
	saveCache()
	return tr.AccessToken, nil
}

// acquireSilent returns a cached or refresh-token-derived access token for
// scopes, or an error if neither is possible.
func acquireSilent(ctx context.Context, scopes []string) (string, error) {
	c := loadCache()
	cacheMu.Lock()
	if t, ok := c.Access[scopeKey(scopes)]; ok && t.ExpiresAt > time.Now().Unix() {
		cacheMu.Unlock()
		return t.AccessToken, nil
	}
	refresh := c.RefreshToken
	cacheMu.Unlock()

	if refresh == "" {
		return "", errors.New("m365: no cached token")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	form.Set("scope", strings.Join(scopes, " ")+" "+oidcScopes)
	return postToken(ctx, form, scopes)
}

// exchangeCode redeems an authorization code for tokens.
func exchangeCode(ctx context.Context, code, verifier string, scopes []string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	form.Set("scope", strings.Join(scopes, " ")+" "+oidcScopes)
	return postToken(ctx, form, scopes)
}

// Credentials holds the stored login secrets used for headless authentication.
type Credentials struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	MFASecret string `json:"mfaSecret"`
}

// loadSecrets reads the on-disk credentials, or returns nil if absent/invalid.
func loadSecrets() *Credentials {
	data, err := os.ReadFile(secretsFile())
	if err != nil {
		return nil
	}
	var c Credentials
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	if c.Email == "" || c.Password == "" || c.MFASecret == "" {
		return nil
	}
	return &c
}

// SecretsPath returns the on-disk path credentials are read from/written to
// (M365_SECRETS_FILE, or the default under configDir).
func SecretsPath() string { return secretsFile() }

// CredentialsReady reports whether kdeps can authenticate without prompting:
// either a cached refresh token already exists, or a complete secrets file
// (email/password/mfaSecret) is present on disk.
func CredentialsReady() bool {
	c := loadCache()
	cacheMu.Lock()
	hasRefresh := c.RefreshToken != ""
	cacheMu.Unlock()
	return hasRefresh || loadSecrets() != nil
}

// SaveCredentials validates and writes email/password/mfaSecret to the
// secrets file (0600), creating configDir if needed. Used by interactive
// callers (the REPL, --model m365 startup) to collect credentials on first
// use instead of requiring the user to hand-write the JSON file themselves.
func SaveCredentials(email, password, mfaSecret string) error {
	email, password, mfaSecret = strings.TrimSpace(email), strings.TrimSpace(password), strings.TrimSpace(mfaSecret)
	if email == "" || password == "" || mfaSecret == "" {
		return errors.New("m365: email, password, and mfaSecret are all required")
	}
	data, err := json.MarshalIndent(Credentials{Email: email, Password: password, MFASecret: mfaSecret}, "", "  ")
	if err != nil {
		return err
	}
	path := secretsFile()
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o700); mkErr != nil {
		return fmt.Errorf("m365: create config dir: %w", mkErr)
	}
	if wErr := os.WriteFile(path, data, 0o600); wErr != nil {
		return fmt.Errorf("m365: write secrets file: %w", wErr)
	}
	return nil
}

// browserLoginFunc is a var (not a direct call) only so tests can stub out the
// real Playwright-driven login without a browser.
//
//nolint:gochecknoglobals // effectively constant; overridden only in tests
var browserLoginFunc = browserLogin

// headedBrowserLoginFunc is browserLoginHeaded, overridable in tests.
//
//nolint:gochecknoglobals // effectively constant; overridden only in tests
var headedBrowserLoginFunc = browserLoginHeaded

// InteractiveLogin performs a one-time, fully manual sign-in: opens a
// visible browser window and waits for the user to complete whatever
// challenge Azure AD presents (password, MFA app, passkey, SSO tile --
// whatever the tenant requires). No Credentials are read or stored. On
// success the resulting refresh token is cached exactly like the scripted
// flow (exchangeCode persists it the same way), so every subsequent
// getToken() call is silent -- the browser only reappears if the cache is
// cleared, the refresh token is revoked, or the caller runs this again
// explicitly (e.g. the REPL's /login command).
func InteractiveLogin(ctx context.Context) (string, error) {
	kdeps_debug.Log("enter: InteractiveLogin")
	tokenMu.Lock()
	defer tokenMu.Unlock()
	authURL, verifier := buildAuthURL(chatScopes)
	code, err := headedBrowserLoginFunc(ctx, authURL)
	if err != nil {
		return "", err
	}
	return exchangeCode(ctx, code, verifier, chatScopes)
}

// runBrowserLogin performs the interactive PKCE leg headlessly, retrying up to
// attempts times (TOTP codes are single-use per 30s window, so retries wait for
// a fresh window).
func runBrowserLogin(ctx context.Context, scopes []string, creds *Credentials, attempts int) (string, error) {
	return runBrowserLoginWithWait(ctx, scopes, creds, attempts, totpWindowWait)
}

// runBrowserLoginWithWait is runBrowserLogin with an injectable inter-attempt
// wait, so tests can exercise the retry loop without sleeping a real TOTP window.
func runBrowserLoginWithWait(
	ctx context.Context, scopes []string, creds *Credentials, attempts int, wait time.Duration,
) (string, error) {
	kdeps_debug.Log("enter: runBrowserLogin")
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		authURL, verifier := buildAuthURL(scopes)
		code, err := browserLoginFunc(ctx, authURL, creds)
		if err == nil {
			var token string
			token, err = exchangeCode(ctx, code, verifier, scopes)
			if err == nil {
				return token, nil
			}
		}
		lastErr = err
		if attempt < attempts {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("m365: browser login failed")
	}
	return "", lastErr
}

// getTokenSilent returns a chat token from cache/refresh without any interactive
// login, or an error if none is available.
func getTokenSilent(ctx context.Context) (string, error) {
	return acquireSilent(ctx, chatScopes)
}

// loginAutomated performs a headless credential login for the chat scopes.
func loginAutomated(ctx context.Context, creds *Credentials) (string, error) {
	kdeps_debug.Log("enter: loginAutomated")
	return runBrowserLogin(ctx, chatScopes, creds, 1)
}

// getToken returns a usable chat access token, refreshing silently when possible
// and falling back to an automated login. Concurrent callers are serialized.
func getToken(ctx context.Context) (string, error) {
	kdeps_debug.Log("enter: getToken")
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if tok, err := getTokenSilent(ctx); err == nil {
		return tok, nil
	}
	secrets := loadSecrets()
	if secrets == nil {
		return "", fmt.Errorf(
			"m365: not signed in -- run /login in the REPL (or kdeps --model m365-copilot"+
				" --backend m365 interactively) to open a browser and sign in,"+
				" or provide email/password/mfaSecret at %s for a headless host",
			secretsFile(),
		)
	}
	return loginAutomated(ctx, secrets)
}

// getTokenForScope returns an access token for arbitrary scopes (e.g. Power
// Platform), refreshing silently when possible and otherwise logging in with
// stored credentials. It returns ("", nil) when no credentials are available.
func getTokenForScope(ctx context.Context, scopes []string) (string, error) {
	kdeps_debug.Log("enter: getTokenForScope")
	if tok, err := acquireSilent(ctx, scopes); err == nil {
		return tok, nil
	}
	secrets := loadSecrets()
	if secrets == nil {
		return "", nil
	}
	return runBrowserLogin(ctx, scopes, secrets, 1)
}

// forceReauth refreshes silently if possible, else performs an automated login.
func forceReauth(ctx context.Context) bool {
	kdeps_debug.Log("enter: forceReauth")
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if _, err := getTokenSilent(ctx); err == nil {
		return true
	}
	secrets := loadSecrets()
	if secrets == nil {
		return false
	}
	_, err := loginAutomated(ctx, secrets)
	return err == nil
}
