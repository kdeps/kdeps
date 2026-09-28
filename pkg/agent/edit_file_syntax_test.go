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

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyntaxKindForPath(t *testing.T) {
	assert.Equal(t, syntaxGo, syntaxKindForPath("/app/main.go"))
	assert.Equal(t, syntaxJSON, syntaxKindForPath("/app/config.json"))
	assert.Equal(t, syntaxYAML, syntaxKindForPath("/app/config.yaml"))
	assert.Equal(t, syntaxYAML, syntaxKindForPath("/app/config.yml"))
	assert.Equal(t, syntaxBalanced, syntaxKindForPath("/app/main.js"))
	assert.Equal(t, syntaxBalancedHash, syntaxKindForPath("/app/main.py"))
	assert.Equal(t, syntaxNone, syntaxKindForPath("/app/README.md"))
	assert.Equal(t, syntaxGo, syntaxKindForPath("/app/MAIN.GO"), "extension match is case-insensitive")
}

func TestValidateGoSyntax_ValidPasses(t *testing.T) {
	require.NoError(t, validateGoSyntax("package x\n\nfunc Foo() {\n\treturn\n}\n"))
}

func TestValidateGoSyntax_MissingBraceRejected(t *testing.T) {
	err := validateGoSyntax("package x\n\nfunc Foo() {\n\treturn\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")
}

func TestValidateJSONSyntax_ValidPasses(t *testing.T) {
	require.NoError(t, validateJSONSyntax(`{"a": 1, "b": [1, 2, 3]}`))
}

func TestValidateJSONSyntax_InvalidRejected(t *testing.T) {
	err := validateJSONSyntax(`{"a": 1,}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestValidateYAMLSyntax_ValidPasses(t *testing.T) {
	require.NoError(t, validateYAMLSyntax("a: 1\nb:\n  - x\n  - y\n"))
}

func TestValidateYAMLSyntax_InvalidRejected(t *testing.T) {
	err := validateYAMLSyntax("a: [1, 2\nb: 3\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")
}

func TestValidateBalancedDelimiters_MatchedPasses(t *testing.T) {
	require.NoError(t, validateBalancedDelimiters("function f(a, b) {\n  return [a, b];\n}\n", false))
}

func TestValidateBalancedDelimiters_UnclosedRejected(t *testing.T) {
	err := validateBalancedDelimiters("function f() {\n  return 1;\n", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed")
	assert.Contains(t, err.Error(), "line 1")
}

func TestValidateBalancedDelimiters_UnexpectedCloserRejected(t *testing.T) {
	err := validateBalancedDelimiters("function f() {\n  return 1;\n}}\n", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected")
}

func TestValidateBalancedDelimiters_MismatchedNestingRejected(t *testing.T) {
	err := validateBalancedDelimiters("let x = (1, 2];\n", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected")
}

func TestValidateBalancedDelimiters_StringContentIgnored(t *testing.T) {
	require.NoError(t, validateBalancedDelimiters(`let s = "unbalanced ( and [ inside a string";`, false))
}

func TestValidateBalancedDelimiters_LineCommentContentIgnored(t *testing.T) {
	require.NoError(t, validateBalancedDelimiters("let x = 1; // unbalanced ( here\n", false))
}

func TestValidateBalancedDelimiters_BlockCommentContentIgnored(t *testing.T) {
	require.NoError(t, validateBalancedDelimiters("let x = 1; /* unbalanced ( here */\n", false))
}

func TestValidateBalancedDelimiters_HashCommentContentIgnored(t *testing.T) {
	require.NoError(t, validateBalancedDelimiters("x = 1  # unbalanced ( here\n", true))
	err := validateBalancedDelimiters("x = 1  # unbalanced ( here\n", false)
	require.Error(t, err, "without hashComments, # is not a comment marker")
}

func TestValidateSyntax_UnsupportedExtensionIsNoOp(t *testing.T) {
	require.NoError(t, validateSyntax("/app/README.md", "this is [ not ) balanced { at all"))
}
