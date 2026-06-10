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

package yaml

import (
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// GlobalComponentsDir exposes the unexported globalComponentsDir helper for unit testing.
var GlobalComponentsDir = globalComponentsDir //nolint:gochecknoglobals // test-only export

// HasJ2Suffix exposes the unexported hasJ2Suffix helper for unit testing.
var HasJ2Suffix = hasJ2Suffix //nolint:gochecknoglobals // test-only export

// TrimJ2Suffix exposes the unexported trimJ2Suffix helper for unit testing.
var TrimJ2Suffix = trimJ2Suffix //nolint:gochecknoglobals // test-only export

// IsKomponentFile exposes the unexported isKomponentFile helper for unit testing.
var IsKomponentFileInternal = isKomponentFile //nolint:gochecknoglobals // test-only export

// ScanComponentsDir exposes the unexported scanComponentsDir method for unit testing.
func (p *Parser) ScanComponentsDir(
	dir string,
	existing map[string]struct{},
) ([]*domain.Resource, map[string]*domain.Component, error) {
	return p.scanComponentsDir(dir, existing)
}

// LoadComponents exposes the unexported loadComponents method for unit testing.
func (p *Parser) LoadComponents(workflow *domain.Workflow, workflowPath string) error {
	return p.loadComponents(workflow, workflowPath)
}

// errorReader returns err on every Read call.
type errorReader struct{ err error }

func (e *errorReader) Read(_ []byte) (int, error) { return 0, e.err }

type whiteboxMockSV struct{}

func (m *whiteboxMockSV) ValidateWorkflow(_ map[string]interface{}) error { return nil }

func (m *whiteboxMockSV) ValidateResource(_ map[string]interface{}) error { return nil }

func (m *whiteboxMockSV) ValidateAgency(_ map[string]interface{}) error { return nil }

func (m *whiteboxMockSV) ValidateComponent(_ map[string]interface{}) error {
	return nil
}

type whiteboxMockEP struct{}

func (m *whiteboxMockEP) Parse(_ string) (*domain.Expression, error) {
	return &domain.Expression{}, nil
}

func (m *whiteboxMockEP) ParseValue(_ interface{}) (*domain.Expression, error) {
	return &domain.Expression{}, nil
}

func (m *whiteboxMockEP) Detect(_ string) domain.ExprType { return domain.ExprTypeLiteral }

func newWhiteboxParser() *Parser {
	return NewParser(&whiteboxMockSV{}, &whiteboxMockEP{})
}
