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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

// newInstrumentCmd is "kdeps instrument": the sessions recorded with
// kdeps --instrument=<name>.
func newInstrumentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "instrument",
		Short: "List, show and remove instrumented sessions (kdeps --instrument=<name>)",
		Long: `An instrumented session records a run for a model to debug: start one
with kdeps --instrument=<name> (any command; bare --instrument names it by
time). It is kept in ~/.kdeps/instrument/<name>/:

  session.md     what ran, when, and how to read the other two files
  events.ndjson  one JSON object per prompt, model request and response,
                 tool call and workflow event, with durations and errors
  trace.log      every log line and call-chain trace, at debug level

  kdeps instrument list           saved sessions, newest first
  kdeps instrument show <name>    session.md and events.ndjson (--trace adds trace.log)
  kdeps instrument path <name>    the session's folder
  kdeps instrument rm <name>      delete it`,
	}
	c.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Saved instrumented sessions, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := instrument.List()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(list) == 0 {
				fmt.Fprintln(out, "No instrumented sessions: start one with kdeps --instrument=<name>")
				return nil
			}
			for _, s := range list {
				fmt.Fprintf(out, "%-32s %s  %6.1f KB  %s\n", s.Name, s.Started.Format("2006-01-02 15:04"),
					float64(s.Bytes)/1024, s.Dir) //nolint:mnd // bytes to KB
			}
			return nil
		},
	})
	show := &cobra.Command{
		Use:   "show <name>",
		Short: "Print a session for a model to read: session.md and events.ndjson",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := instrument.Dir(args[0])
			if err != nil {
				return err
			}
			files := []string{instrument.SessionFile, instrument.EventsFile}
			if trace, _ := cmd.Flags().GetBool("trace"); trace {
				files = append(files, instrument.TraceFile)
			}
			return printSessionFiles(cmd.OutOrStdout(), dir, files)
		},
	}
	show.Flags().Bool("trace", false, "also print trace.log (every log line and call-chain trace)")
	c.AddCommand(show, &cobra.Command{
		Use:   "path <name>",
		Short: "Print a session's folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := instrument.Dir(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), dir)
			return nil
		},
	}, &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a saved session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := instrument.Remove(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed instrumented session %s\n", args[0])
			return nil
		},
	})
	return c
}

// printSessionFiles prints each file under a "==> name <==" header.
func printSessionFiles(w io.Writer, dir string, files []string) error {
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "==> %s <==\n%s\n", filepath.Join(dir, f), data)
	}
	return nil
}
