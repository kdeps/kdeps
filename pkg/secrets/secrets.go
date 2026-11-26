// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

// Package secrets reads and writes the kdeps vault: a YAML map of named
// secrets, encrypted with an age passphrase (scrypt). A project keeps its
// vault as secrets.age next to workflow.yaml, which is safe to commit; the
// machine keeps one at ~/.kdeps/secrets.age.
package secrets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"filippo.io/age"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
)

const (
	// FileName is the vault file name, in a project and in ~/.kdeps.
	FileName = "secrets.age"
	// EnvPassword holds the vault password.
	EnvPassword = "KDEPS_VAULT_PASSWORD"
	// EnvPasswordFile names a file whose first line is the vault password.
	EnvPasswordFile = "KDEPS_VAULT_PASSWORD_FILE"

	filePerm = 0o600
	dirPerm  = 0o700
)

// ErrNoPassword is returned when no password source is available.
var ErrNoPassword = errors.New(
	"the vault is locked: set " + EnvPassword + " or " + EnvPasswordFile + ", or run kdeps in a terminal")

// ErrNotFound is returned when no vault holds the named secret.
var ErrNotFound = errors.New("secret not found")

// Test hooks.
//
//nolint:gochecknoglobals // replaced in tests
var (
	isTerminal             = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	readPassword           = func() ([]byte, error) { return term.ReadPassword(int(os.Stdin.Fd())) }
	promptOut    io.Writer = os.Stderr
	workFactor             = 18 // age's default scrypt cost; tests lower it
)

// GlobalPath returns ~/.kdeps/secrets.age.
func GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".kdeps", FileName), nil
}

// ProjectPath returns the vault path of the project in dir.
func ProjectPath(dir string) string {
	return filepath.Join(dir, FileName)
}

// Password returns the vault password: KDEPS_VAULT_PASSWORD, then the file
// KDEPS_VAULT_PASSWORD_FILE names, then a terminal prompt.
func Password() (string, error) {
	kdeps_debug.Log("enter: secrets.Password")
	if p := os.Getenv(EnvPassword); p != "" {
		return p, nil
	}
	if f := os.Getenv(EnvPasswordFile); f != "" {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", EnvPasswordFile, err)
		}
		line, _, _ := strings.Cut(string(data), "\n")
		if line = strings.TrimRight(line, "\r"); line == "" {
			return "", fmt.Errorf("%s: %s is empty", EnvPasswordFile, f)
		}
		return line, nil
	}
	if !isTerminal() {
		return "", ErrNoPassword
	}
	fmt.Fprint(promptOut, "Vault password: ")
	b, err := readPassword()
	fmt.Fprintln(promptOut)
	if err != nil {
		return "", fmt.Errorf("read vault password: %w", err)
	}
	if len(b) == 0 {
		return "", errors.New("the vault password is empty")
	}
	return string(b), nil
}

// NewPassword returns the password for a new vault. It reads the same
// sources as Password; a terminal prompt asks twice and the answers must match.
func NewPassword() (string, error) {
	kdeps_debug.Log("enter: secrets.NewPassword")
	if os.Getenv(EnvPassword) != "" || os.Getenv(EnvPasswordFile) != "" || !isTerminal() {
		return Password()
	}
	first, err := Password()
	if err != nil {
		return "", err
	}
	fmt.Fprint(promptOut, "Repeat the vault password: ")
	again, err := readPassword()
	fmt.Fprintln(promptOut)
	if err != nil {
		return "", fmt.Errorf("read vault password: %w", err)
	}
	if string(again) != first {
		return "", errors.New("the passwords do not match")
	}
	return first, nil
}

// Read decrypts the vault at path. A missing file is an empty vault.
func Read(path, password string) (map[string]string, error) {
	kdeps_debug.Log("enter: secrets.Read")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	id, err := age.NewScryptIdentity(password)
	if err != nil {
		return nil, fmt.Errorf("vault password: %w", err)
	}
	r, err := age.Decrypt(bytes.NewReader(data), id)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: wrong password or not a kdeps vault: %w", path, err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", path, err)
	}
	m := map[string]string{}
	if err = yaml.Unmarshal(plain, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// Write encrypts m with password and writes it to path (mode 0600).
func Write(path, password string, m map[string]string) error {
	kdeps_debug.Log("enter: secrets.Write")
	plain, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode vault: %w", err)
	}
	rcpt, err := age.NewScryptRecipient(password)
	if err != nil {
		return fmt.Errorf("vault password: %w", err)
	}
	rcpt.SetWorkFactor(workFactor)
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, rcpt)
	if err != nil {
		return fmt.Errorf("encrypt vault: %w", err)
	}
	if _, err = w.Write(plain); err != nil {
		return fmt.Errorf("encrypt vault: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("encrypt vault: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, buf.Bytes(), filePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Names returns the secret names of m, sorted.
func Names(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Resolver looks a secret up in a list of vaults, first match wins. It
// asks for the password once and decrypts each vault once, on first use.
type Resolver struct {
	paths []string

	mu       sync.Mutex
	password string
	lockErr  error // why the password could not be read; asked once
	vaults   map[string]map[string]string
}

// NewResolver returns a resolver over paths, in priority order.
func NewResolver(paths ...string) *Resolver {
	return &Resolver{paths: paths, vaults: map[string]map[string]string{}}
}

// Lookup returns the named secret.
func (r *Resolver) Lookup(name string) (string, error) {
	kdeps_debug.Log("enter: Resolver.Lookup")
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		v, err := r.vault(p)
		if err != nil {
			return "", err
		}
		if val, found := v[name]; found {
			return val, nil
		}
	}
	return "", fmt.Errorf("%w: %s (vaults: %s)", ErrNotFound, name, strings.Join(r.paths, ", "))
}

// vault returns the decrypted vault at p, reading it on first use.
func (r *Resolver) vault(p string) (map[string]string, error) {
	if v, ok := r.vaults[p]; ok {
		return v, nil
	}
	pw, err := r.unlock()
	if err != nil {
		return nil, err
	}
	v, err := Read(p, pw)
	if err != nil {
		return nil, err
	}
	r.vaults[p] = v
	return v, nil
}

// unlock returns the password, asking for it once.
func (r *Resolver) unlock() (string, error) {
	if r.password != "" {
		return r.password, nil
	}
	if r.lockErr != nil {
		return "", r.lockErr
	}
	pw, err := Password()
	if err != nil {
		r.lockErr = err
		return "", err
	}
	r.password = pw
	return pw, nil
}

//nolint:gochecknoglobals // one resolver per vault list per process
var (
	cacheMu    sync.Mutex
	projectDir string
	resolvers  = map[string]*Resolver{}
)

// cached returns the resolver over paths, made once per process.
func cached(paths ...string) *Resolver {
	key := strings.Join(paths, "\x00")
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if r, ok := resolvers[key]; ok {
		return r
	}
	r := NewResolver(paths...)
	resolvers[key] = r
	return r
}

// Global returns the resolver over ~/.kdeps/secrets.age.
func Global() *Resolver {
	p, err := GlobalPath()
	if err != nil {
		return cached()
	}
	return cached(p)
}

// UseProject sets the folder of the workflow this process runs. Project
// then reads dir/secrets.age before ~/.kdeps/secrets.age.
func UseProject(dir string) {
	cacheMu.Lock()
	projectDir = dir
	cacheMu.Unlock()
}

// Project returns the resolver of the workflow set with UseProject.
func Project() *Resolver {
	cacheMu.Lock()
	dir := projectDir
	cacheMu.Unlock()
	return ForProject(dir)
}

// ForProject returns the resolver for a workflow in dir: dir/secrets.age,
// then ~/.kdeps/secrets.age.
func ForProject(dir string) *Resolver {
	var paths []string
	if dir != "" {
		paths = append(paths, ProjectPath(dir))
	}
	if p, err := GlobalPath(); err == nil {
		paths = append(paths, p)
	}
	return cached(paths...)
}

// SetWorkFactorForTesting lowers the scrypt cost of new vaults so tests
// run fast. It returns a function that restores the old cost.
func SetWorkFactorForTesting(n int) func() {
	old := workFactor
	workFactor = n
	return func() { workFactor = old }
}

// resetForTest clears the cached resolvers and the project folder.
func resetForTest() {
	cacheMu.Lock()
	projectDir = ""
	resolvers = map[string]*Resolver{}
	cacheMu.Unlock()
}
