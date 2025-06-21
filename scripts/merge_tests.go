//go:build tools
// +build tools

// This file provides a utility to merge Go test files. It is excluded from
// normal builds and test runs via the build tag above.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// helper to read file content as lines
func ReadLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// write formatted Go source to file
func WriteFormatted(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		// if formatting fails, write unformatted for debugging
		formatted = src
	}
	return os.WriteFile(path, formatted, 0644)
}

// mergeTestsInDir merges test files with the same prefix (before the first "_")
// into a single *_test.go file.
func MergeTestsInDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Build symbol -> production file map for this directory to help
	// map "*_extra_test.go" files to the correct base file even when the
	// filename prefixes differ (e.g., current_architecture_extra_test.go
	// targets cache.go).
	symToFile := map[string]string{}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fset := token.NewFileSet()
		af, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		if err != nil {
			return nil
		}
		for name := range af.Scope.Objects {
			symToFile[name] = path
		}
		return nil
	})

	// map[prefix] -> list of files
	groups := make(map[string][]string)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		pkgName, err := detectPackage(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		// Attempt to map via symbol matching for *_extra_test.go variants
		basePrefix := strings.TrimSuffix(name, "_test.go")
		basePrefix = strings.Split(basePrefix, "_extra")[0]

		// Default mapping key
		key := pkgName + "::" + basePrefix

		if strings.Contains(name, "_extra_test.go") || strings.Contains(name, "_additional_test.go") || strings.Contains(name, "_more_test.go") || strings.Contains(name, "_simple_test.go") {
			// Inspect test file for referenced symbols
			src, _ := os.ReadFile(filepath.Join(dir, name))
			for sym, prod := range symToFile {
				if bytes.Contains(src, []byte(sym+"(")) {
					prodBase := strings.TrimSuffix(filepath.Base(prod), ".go")
					key = pkgName + "::" + prodBase
					break
				}
			}
		}

		groups[key] = append(groups[key], filepath.Join(dir, name))
	}

	for key, files := range groups {
		parts := strings.SplitN(key, "::", 2)
		if len(parts) != 2 {
			continue
		}
		// parts[0] is package name, parts[1] is prefix
		prefix := parts[1]
		if len(files) <= 1 {
			continue // nothing to merge
		}
		sort.Strings(files) // deterministic
		baseFile := filepath.Join(dir, prefix+"_test.go")
		if !contains(files, baseFile) {
			// choose first as base if canonical not present
			baseFile = files[0]
		}

		var baseLines []string
		baseLines, err = readLines(baseFile)
		if err != nil {
			return err
		}

		// extract import block indices in base file
		importStart, importEnd := -1, -1
		for i, line := range baseLines {
			lineTrim := strings.TrimSpace(line)
			if importStart == -1 {
				if lineTrim == "import(" || lineTrim == "import (" || strings.HasPrefix(lineTrim, "import (") {
					importStart = i
				} else if strings.HasPrefix(lineTrim, "import ") {
					// convert single line import to block for easier merging
					orig := baseLines[i]
					importStart = i
					importEnd = i
					parts := strings.SplitN(orig, " ", 2)
					if len(parts) == 2 {
						imp := strings.TrimSpace(parts[1])
						baseLines[i] = "import ("
						baseLines = append(baseLines, "") // extend slice
						copy(baseLines[i+2:], baseLines[i+1:])
						baseLines[i+1] = "\t" + imp
						importEnd = i + 2
					}
					break
				}
			} else if importEnd == -1 && lineTrim == ")" {
				importEnd = i
			}
		}
		if importStart == -1 {
			// if no import block, create one after package line
			for i, line := range baseLines {
				if strings.HasPrefix(strings.TrimSpace(line), "package ") {
					importStart = i + 1
					importEnd = importStart + 1
					// build new slice: lines before, then import block, then remaining lines
					newLines := append([]string{}, baseLines[:importStart]...)
					newLines = append(newLines, "import (", ")")
					newLines = append(newLines, baseLines[importStart:]...)
					baseLines = newLines
					break
				}
			}
		}

		// collect existing imports in base and compute self import path
		modulePath := modulePath()
		rel, _ := filepath.Rel(workRoot(), dir)
		selfImportPath := filepath.ToSlash(filepath.Join(modulePath, rel))

		existingImports := map[string]struct{}{}
		selfImportLiteral := fmt.Sprintf("\"%s\"", selfImportPath)

		// rebuild import block without self-imports
		newBlock := []string{}
		for i := importStart + 1; i < importEnd; i++ {
			imp := strings.TrimSpace(baseLines[i])
			if imp == selfImportLiteral {
				continue // drop self import
			}
			if imp != "" {
				existingImports[imp] = struct{}{}
				newBlock = append(newBlock, baseLines[i])
			}
		}
		// replace block
		baseLines = append(baseLines[:importStart+1], append(newBlock, baseLines[importEnd:]...)...)
		importEnd = importStart + 1 + len(newBlock)

		// content to append after end of file
		var additionalContent bytes.Buffer

		// iterate over other files
		for _, f := range files {
			if f == baseFile {
				continue
			}
			lines, err := readLines(f)
			if err != nil {
				return err
			}
			inImport := false
			for _, line := range lines {
				trim := strings.TrimSpace(line)
				// skip package line from additional file
				if strings.HasPrefix(trim, "package ") {
					continue
				}
				// handle import block in additional file
				if !inImport {
					if trim == "import(" || trim == "import (" || strings.HasPrefix(trim, "import (") {
						inImport = true
						continue
					} else if strings.HasPrefix(trim, "import ") {
						imp := strings.TrimPrefix(trim, "import ")
						if imp == selfImportLiteral {
							continue
						}
						if _, ok := existingImports[imp]; !ok {
							existingImports[imp] = struct{}{}
							baseLines = insertImport(baseLines, importEnd, imp)
							importEnd++
						}
						continue
					}
				} else {
					if trim == ")" {
						inImport = false
						continue
					}
					imp := trim
					if imp == selfImportLiteral {
						continue
					}
					if _, ok := existingImports[imp]; !ok {
						existingImports[imp] = struct{}{}
						baseLines = insertImport(baseLines, importEnd, imp)
						importEnd++
					}
					continue
				}
				// regular code lines
				additionalContent.WriteString(line)
				additionalContent.WriteByte('\n')
			}
			// delete the processed file
			if err := os.Remove(f); err != nil {
				return err
			}
		}

		// append additional content
		if additionalContent.Len() > 0 {
			baseLines = append(baseLines, "", strings.TrimRight(additionalContent.String(), "\n"))
		}

		// write back
		raw := []byte(strings.Join(baseLines, "\n"))
		if err := writeFormatted(baseFile, raw); err != nil {
			return fmt.Errorf("formatting %s: %w", baseFile, err)
		}
	}

	return nil
}

func Contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func InsertImport(lines []string, importEnd int, imp string) []string {
	// insert imp before importEnd line index
	lines = append(lines, "") // extend slice
	copy(lines[importEnd+1:], lines[importEnd:])
	lines[importEnd] = "\t" + imp
	return lines
}

func Main() {
	dirFlag := flag.String("dir", ".", "root directory to process")
	flag.Parse()

	if err := filepath.WalkDir(*dirFlag, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return mergeTestsInDir(path)
		}
		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// detectPackage returns the package clause of a Go file.
func DetectPackage(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "package ")), nil
		}
	}
	return "", fmt.Errorf("package not found")
}

func ModulePath() string {
	data, err := os.ReadFile(filepath.Join(workRoot(), "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func WorkRoot() string {
	wd, _ := os.Getwd()
	return wd
}
