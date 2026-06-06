package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/manifest"
)

const (
	registrySubmitTimeout = 5 * time.Minute
	githubURLSplitParts   = 2
	githubSSHRepoParts    = 2
)

type registryFormula struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Type        string   `yaml:"type"`
	GitHub      string   `yaml:"github"`
	Tarball     string   `yaml:"tarball"`
	SHA256      string   `yaml:"sha256"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	License     string   `yaml:"license,omitempty"`
}

func newRegistrySubmitCmd() *cobra.Command {
	kdeps_debug.Log("enter: newRegistrySubmitCmd")
	cmd := &cobra.Command{
		Use:   "submit [path]",
		Short: "Generate a registry formula for submitting a package via GitHub PR.",
		Long: `Generate a formula YAML for your package and print it to stdout.

To publish a package:
  1. Tag a release in your GitHub repo (e.g. git tag v1.2.0 && git push --tags)
  2. Run kdeps registry submit --tag v1.2.0
  3. Open a PR to https://github.com/kdeps/registry adding the printed formula
     under formulas/<your-package-name>.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kdeps_debug.Log("enter: registrySubmitCmd.RunE")
			dir := submitDirFromArgs(args)
			tag, _ := cmd.Flags().GetString("tag")
			if tag == "" {
				return errors.New("--tag is required (e.g. --tag v1.2.0)")
			}
			return doRegistrySubmit(cmd, dir, tag)
		},
	}
	cmd.Flags().StringP("tag", "t", "", "Git tag for the release (e.g. v1.2.0)")
	return cmd
}

func submitDirFromArgs(args []string) string {
	if len(args) == 0 {
		return "."
	}
	return args[0]
}

func doRegistrySubmit(cmd *cobra.Command, dir, tag string) error {
	kdeps_debug.Log("enter: doRegistrySubmit")

	m, err := loadValidatedManifest(dir)
	if err != nil {
		return err
	}

	githubRepo, err := detectGitHubRepo(dir)
	if err != nil {
		return fmt.Errorf(
			"detect GitHub repo: %w\n\nSet the GitHub remote with: git remote add origin https://github.com/owner/repo",
			err,
		)
	}

	tarbullURL := githubTarballURL(githubRepo, tag)
	hash, err := computeRemoteSHA256(tarbullURL)
	if err != nil {
		return fmt.Errorf("compute sha256 for %s: %w", tarbullURL, err)
	}

	formulaYAML, err := encodeRegistryFormula(buildRegistryFormula(m, githubRepo, tarbullURL, hash))
	if err != nil {
		return err
	}

	printRegistryFormula(cmd, m.Name, formulaYAML)
	return nil
}

func loadValidatedManifest(dir string) (*manifest.Manifest, error) {
	m, err := manifest.Load(dir)
	if err != nil {
		return nil, err
	}
	if validateErr := manifest.Validate(m); validateErr != nil {
		return nil, validateErr
	}
	return m, nil
}

func githubTarballURL(githubRepo, tag string) string {
	return fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.tar.gz", githubRepo, tag)
}

func buildRegistryFormula(m *manifest.Manifest, githubRepo, tarballURL, hash string) registryFormula {
	return registryFormula{
		Name:        m.Name,
		Version:     m.Version,
		Type:        m.Type,
		GitHub:      githubRepo,
		Tarball:     tarballURL,
		SHA256:      hash,
		Description: m.Description,
		Tags:        m.Tags,
		License:     m.License,
	}
}

func encodeRegistryFormula(f registryFormula) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(githubURLSplitParts)
	if encErr := enc.Encode(f); encErr != nil {
		return "", fmt.Errorf("encode formula: %w", encErr)
	}
	return buf.String(), nil
}

func printRegistryFormula(cmd *cobra.Command, packageName, formulaYAML string) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "# Save this as formulas/"+packageName+".yaml in a PR to https://github.com/kdeps/registry")
	fmt.Fprintln(w)
	fmt.Fprint(w, formulaYAML)
}

func detectGitHubRepo(dir string) (string, error) {
	kdeps_debug.Log("enter: detectGitHubRepo")
	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w", err)
	}
	remote := strings.TrimSpace(string(out))
	return parseGitHubRepo(remote)
}

func parseGitHubRepo(remote string) (string, error) {
	if repo, ok := parseGitHubSSHRemote(remote); ok {
		return repo, nil
	}
	if repo, ok := parseGitHubHTTPSRemote(remote); ok {
		return repo, nil
	}
	return "", fmt.Errorf("unrecognized GitHub remote URL: %s", remote)
}

func parseGitHubSSHRemote(remote string) (string, bool) {
	if !strings.HasPrefix(remote, "git@github.com:") {
		return "", false
	}
	repo := strings.TrimPrefix(remote, "git@github.com:")
	repo = strings.TrimSuffix(repo, ".git")
	return repo, true
}

func parseGitHubHTTPSRemote(remote string) (string, bool) {
	if !strings.Contains(remote, "github.com/") {
		return "", false
	}
	parts := strings.SplitN(remote, "github.com/", githubURLSplitParts)
	if len(parts) != githubSSHRepoParts {
		return "", false
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	repo = strings.TrimRight(repo, "/")
	return repo, true
}

func computeRemoteSHA256(url string) (string, error) {
	kdeps_debug.Log("enter: computeRemoteSHA256")
	client := &http.Client{Timeout: registrySubmitTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch tarball: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, url)
	}
	h := sha256.New()
	if _, copyErr := io.Copy(h, resp.Body); copyErr != nil {
		return "", fmt.Errorf("hash tarball: %w", copyErr)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
