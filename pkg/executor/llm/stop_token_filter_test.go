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

package llm

import (
	"strings"
	"testing"
)

func TestStopTokenFilterWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		chunks []string
		want   string
	}{
		{
			name:   "no stop token passes through",
			chunks: []string{"Hello", ", ", "world"},
			want:   "Hello, world",
		},
		{
			name:   "whole token in one chunk",
			chunks: []string{"Hello", "<|eot_id|>"},
			want:   "Hello",
		},
		{
			name:   "token split across chunks",
			chunks: []string{"Hello", "<|eot", "_id|>"},
			want:   "Hello",
		},
		{
			name:   "token split one byte at a time",
			chunks: []string{"Hi", "<", "|", "e", "o", "t", "_", "i", "d", "|", ">"},
			want:   "Hi",
		},
		{
			name:   "short token",
			chunks: []string{"Bye", "</s>"},
			want:   "Bye",
		},
		{
			name:   "text resembling a token prefix is still emitted",
			chunks: []string{"a < b", " and c"},
			want:   "a < b and c",
		},
		{
			name:   "token in the middle is removed",
			chunks: []string{"one<|im_end|>two"},
			want:   "onetwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			f := &stopTokenFilterWriter{w: &sb}
			for _, c := range tt.chunks {
				n, err := f.Write([]byte(c))
				if err != nil {
					t.Fatalf("Write(%q) error: %v", c, err)
				}
				if n != len(c) {
					t.Fatalf("Write(%q) = %d, want %d", c, n, len(c))
				}
			}
			if got := sb.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLongestStopTokenPrefixSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want int
	}{
		{"hello", 0},
		{"hello<", 1},
		{"hello<|eot", 5},
		{"hello<|eot_id|>", 0}, // complete token is not a proper prefix
		{"hello<", 1},
		{"bye</", 2},
	}
	for _, tt := range tests {
		if got := longestStopTokenPrefixSuffix(tt.in); got != tt.want {
			t.Errorf("longestStopTokenPrefixSuffix(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
