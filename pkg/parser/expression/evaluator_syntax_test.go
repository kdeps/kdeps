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

package expression

import (
	"testing"
)

func TestIsExprLangSyntax_DoubleQuoted(t *testing.T) {
	if !isExprLangSyntax(`"hello"`) {
		t.Error(`expected true for "hello"`)
	}
	if !isExprLangSyntax(`"foo bar"`) {
		t.Error(`expected true for "foo bar"`)
	}
}

func TestIsSimpleIdentifier_InvalidChars(t *testing.T) {
	if isSimpleIdentifier("foo@bar") {
		t.Error("expected false for foo@bar")
	}
	if isSimpleIdentifier("foo bar") {
		t.Error("expected false for 'foo bar'")
	}
	if isSimpleIdentifier("foo$bar") {
		t.Error("expected false for foo$bar")
	}
	if isSimpleIdentifier("foo!bar") {
		t.Error("expected false for foo!bar")
	}
}
