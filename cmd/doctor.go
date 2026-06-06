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

//go:build !js

package cmd

import (
	"errors"
	"fmt"
	"os"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func newDoctorCmd() *cobra.Command {
	kdeps_debug.Log("enter: newDoctorCmd")
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run system health checks (config, Ollama, Python, agents)",
		Long: `Run diagnostic health checks for kdeps:

  - Config file existence and validity
  - Config validation warnings (typos, missing keys)
  - Ollama server reachability
  - Python interpreter availability
  - Backend / API key alignment
  - Installed agents
  - Critical environment variables

Exits with code 1 if any check fails.`,
		RunE: runDoctor,
	}
}

func runDoctor(_ *cobra.Command, _ []string) error {
	kdeps_debug.Log("enter: runDoctor")

	cfg := loadDoctorConfig()
	report := config.RunDoctor(cfg)
	fmt.Fprint(os.Stdout, report.FormatReport())

	if report.Healthy {
		return nil
	}
	return errors.New("health check failed — review warnings above")
}

// loadDoctorConfig loads config for doctor, falling back to an empty config on error.
func loadDoctorConfig() *config.Config {
	cfg, err := config.LoadStruct()
	if err != nil {
		return &config.Config{}
	}
	return cfg
}
