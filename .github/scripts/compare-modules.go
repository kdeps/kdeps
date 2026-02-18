package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

type Module struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
}

type Update struct {
	Path   string
	OldVer string
	NewVer string
}

func main() {
	before := make(map[string]string)
	after := make(map[string]string)

	// Read before modules
	dataBefore, err := os.ReadFile("/tmp/modules_before.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read /tmp/modules_before.json: %v\n", err)
		os.Exit(1)
	}
	decBefore := json.NewDecoder(bytes.NewReader(dataBefore))
	for {
		var m Module
		err := decBefore.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to decode module from /tmp/modules_before.json: %v\n", err)
			os.Exit(1)
		}
		if m.Version != "" {
			before[m.Path] = m.Version
		}
	}

	// Read after modules
	dataAfter, err := os.ReadFile("/tmp/modules_after.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read /tmp/modules_after.json: %v\n", err)
		os.Exit(1)
	}
	decAfter := json.NewDecoder(bytes.NewReader(dataAfter))
	for {
		var m Module
		err := decAfter.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to decode module from /tmp/modules_after.json: %v\n", err)
			os.Exit(1)
		}
		if m.Version != "" {
			after[m.Path] = m.Version
		}
	}

	// Find updates and sort them
	var updates []Update

	for path, newVer := range after {
		if oldVer, exists := before[path]; exists && oldVer != newVer {
			updates = append(updates, Update{Path: path, OldVer: oldVer, NewVer: newVer})
		}
	}

	sort.Slice(updates, func(i, j int) bool {
		return updates[i].Path < updates[j].Path
	})

	// Print results
	fmt.Println("### Module Updates")
	fmt.Println("")

	if len(updates) == 0 {
		fmt.Println("_No direct dependency updates (only transitive dependencies)_")
	} else {
		for _, u := range updates {
			fmt.Printf("- **%s**: `%s` → `%s`\n", u.Path, u.OldVer, u.NewVer)
		}
	}

	fmt.Printf("\n**Total:** %d module(s) updated\n", len(updates))
}
