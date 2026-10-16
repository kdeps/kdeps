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

package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// assetModes names where a set is used, for kdeps info.
//
//nolint:gochecknoglobals // fixed table
var assetModes = map[string]string{
	"agent":    "agent mode",
	"workflow": "workflow mode",
	"both":     "workflow and agent mode",
	"cli":      "the kdeps CLI",
}

// assetInfo is kdeps info for an asset set ("tools") or item
// ("tools/read_file"): its purpose, where kdeps runs it and its state, as
// markdown. ok is false when ref names no asset set.
func assetInfo(ref string) (string, bool) {
	set, name, err := assets.SplitID(ref)
	if err != nil || !slices.Contains(assets.Registered(), set) {
		return "", false
	}
	info, known := assets.SetInfos[set]
	if !known {
		return "", false
	}
	var b strings.Builder
	title := set
	if name != "" {
		title = assets.ID(set, name)
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	if name == "" {
		fmt.Fprintf(&b, "%s\n\n", info.Purpose)
		writeAssetDetails(&b, set, info)
		writeAssetFlow(&b, info, set)
		writeSetItems(&b, set)
		return b.String(), true
	}
	items, _ := assets.Items(set, assets.Root())
	var item *assets.Item
	for i := range items {
		if items[i].Name == name {
			item = &items[i]
		}
	}
	data, readErr := assets.Open(set).ReadFile(set + "/" + name + ".yaml")
	purpose := ""
	if readErr == nil {
		purpose = assets.ItemDescription(data)
	}
	if purpose == "" {
		purpose = info.Purpose
	}
	fmt.Fprintf(&b, "%s\n\n", purpose)
	writeAssetDetails(&b, set, info)
	switch {
	case item == nil || item.Version == "":
		fmt.Fprintf(&b, "- Installed: no - `kdeps install %s`\n", title)
	case item.Removed:
		fmt.Fprintf(&b, "- Installed: removed - `kdeps install %s` brings it back\n", title)
	default:
		fmt.Fprintf(&b, "- Installed: %s\n", item.Version)
	}
	if readErr == nil {
		if t := assets.ContextTokens(set, name, data); t > 0 {
			fmt.Fprintf(&b, "- LLM context: ~%d tokens when loaded\n", t)
		} else {
			b.WriteString("- LLM context: none - kdeps uses it without sending it to the model\n")
		}
	}
	writeAssetFlow(&b, info, title)
	return b.String(), true
}

func writeAssetDetails(b *strings.Builder, set string, info assets.SetInfo) {
	fmt.Fprintf(b, "- Set: %s, used in %s\n", set, assetModes[info.Mode])
	fmt.Fprintf(b, "- Loaded by: %s\n", strings.Join(info.UsedBy, ", "))
}

// writeAssetFlow draws the set's flow, marking the step it drives.
func writeAssetFlow(b *strings.Builder, info assets.SetInfo, label string) {
	b.WriteString("\n## Where it runs\n\n```text\n")
	for i, s := range info.Flow {
		arrow := "   "
		if i > 0 {
			arrow = "-> "
		}
		if s.Here {
			fmt.Fprintf(b, "%s[ %s ]   <- %s\n", arrow, s.Label, label)
			continue
		}
		fmt.Fprintf(b, "%s%s\n", arrow, s.Label)
	}
	b.WriteString("```\n")
}

// writeSetItems lists a set's items and their state.
func writeSetItems(b *strings.Builder, set string) {
	items, err := assets.Items(set, assets.Root())
	if err != nil || len(items) == 0 {
		fmt.Fprintf(b, "\nNo items installed - `kdeps install %s`\n", set)
		return
	}
	b.WriteString("\n## Items\n\n")
	for _, it := range items {
		state := it.Version
		switch {
		case it.Removed:
			state = "removed"
		case state == "":
			state = "not installed"
		}
		fmt.Fprintf(b, "- `%s` %s\n", it.ID, state)
	}
}
