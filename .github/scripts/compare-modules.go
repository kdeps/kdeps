package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if data, err := os.ReadFile("/tmp/modules_before.json"); err == nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		for {
			var m Module
			if err := dec.Decode(&m); err != nil {
				break
			}
			if m.Version != "" {
				before[m.Path] = m.Version
			}
		}
	}

	// Read after modules
	if data, err := os.ReadFile("/tmp/modules_after.json"); err == nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		for {
			var m Module
			if err := dec.Decode(&m); err != nil {
				break
			}
			if m.Version != "" {
				after[m.Path] = m.Version
			}
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
