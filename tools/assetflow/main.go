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

// Command assetflow generates pkg/assets/setflow_gen.go: for every asset
// set, where kdeps loads it, read from the code itself. It builds the call
// graph of kdeps, finds each assets.Open / assets.ReadBundleFile call that
// names a set, and walks the graph back to the kdeps commands, slash
// commands and start-up code that reach it. kdeps info and the published
// asset index (kdeps.io) show the result.
//
//	go run ./tools/assetflow          # rewrite pkg/assets/setflow_gen.go
//	go run ./tools/assetflow -check   # fail when it is out of date
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/kdeps/kdeps/v2/cmd"
)

const outFile = "pkg/assets/setflow_gen.go"

func main() {
	check := flag.Bool("check", false, "fail when "+outFile+" is out of date instead of writing it")
	explain := flag.Bool("explain", false, "print each set's story, users and call path (file and line)")
	flag.Parse()
	flows, cg, err := analyze(".", commandEntries(cmd.NewRootCmd()))
	if *explain && cg != nil {
		explainFlows(os.Stdout, cg, flows)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "assetflow:", err)
		os.Exit(1)
	}
	if *explain {
		return
	}
	src, err := render(flows)
	if err != nil {
		fmt.Fprintln(os.Stderr, "assetflow:", err)
		os.Exit(1)
	}
	if *check {
		cur, _ := os.ReadFile(outFile)
		if !bytes.Equal(cur, src) {
			fmt.Fprintln(os.Stderr, "assetflow: "+outFile+" is out of date: run go run ./tools/assetflow")
			os.Exit(1)
		}
		return
	}
	if err = os.WriteFile(outFile, src, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "assetflow:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s: %d sets\n", outFile, len(flows))
}
