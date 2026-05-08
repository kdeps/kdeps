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

func newRegistrySubmitCmd() *cobra.Command {
	kdeps_debug.Log("enter: newRegistrySubmitCmd")
	cmd := &cobra.Command{
		Use:   "submit [path]",
		Short: "Generate a registry formula for submitting a package via GitHub PR.",
		Long: `Generate a formula YAML for your package and print it to stdout.

To publish a package:
  1. Tag a release in your GitHub repo (e.g. git tag v1.2.0 && git push --tags)
  2. Run kdeps registry submit --tag v1.2.0
  3. Open a PR to https://github.com/kdeps-io/registry adding the printed formula
     under formulas/<your-package-name>.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kdeps_debug.Log("enter: registrySubmitCmd.RunE")
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
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

func doRegistrySubmit(cmd *cobra.Command, dir, tag string) error {
	kdeps_debug.Log("enter: doRegistrySubmit")

	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	if validateErr := manifest.Validate(m); validateErr != nil {
		return validateErr
	}

	githubRepo, err := detectGitHubRepo(dir)
	if err != nil {
		return fmt.Errorf(
			"detect GitHub repo: %w\n\nSet the GitHub remote with: git remote add origin https://github.com/owner/repo",
			err,
		)
	}

	version := strings.TrimPrefix(tag, "v")
	tarbullURL := fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.tar.gz", githubRepo, tag)

	hash, err := computeRemoteSHA256(tarbullURL)
	if err != nil {
		return fmt.Errorf("compute sha256 for %s: %w", tarbullURL, err)
	}

	type formula struct {
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

	f := formula{
		Name:        m.Name,
		Version:     version,
		Type:        m.Type,
		GitHub:      githubRepo,
		Tarball:     tarbullURL,
		SHA256:      hash,
		Description: m.Description,
		Tags:        m.Tags,
		License:     m.License,
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(githubURLSplitParts)
	if encErr := enc.Encode(f); encErr != nil {
		return fmt.Errorf("encode formula: %w", encErr)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "# Save this as formulas/"+m.Name+".yaml in a PR to https://github.com/kdeps-io/registry")
	fmt.Fprintln(w)
	fmt.Fprint(w, buf.String())
	return nil
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
	// SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(remote, "git@github.com:") {
		repo := strings.TrimPrefix(remote, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo, nil
	}
	// HTTPS: https://github.com/owner/repo.git or https://github.com/owner/repo
	if strings.Contains(remote, "github.com/") {
		parts := strings.SplitN(remote, "github.com/", githubURLSplitParts)
		if len(parts) == githubSSHRepoParts {
			repo := strings.TrimSuffix(parts[1], ".git")
			repo = strings.TrimRight(repo, "/")
			return repo, nil
		}
	}
	return "", fmt.Errorf("unrecognized GitHub remote URL: %s", remote)
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
