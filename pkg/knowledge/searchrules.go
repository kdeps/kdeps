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

package knowledge

import (
	"errors"
	"fmt"
	"sync"

	kgsearch "github.com/kdeps/kartographer/search"
	"gopkg.in/yaml.v3"
)

// searchItem is the local search word rules item of the knowledge asset set (search.yaml).
const searchItem = "search"

// searchRules is search.yaml: how the local search indexes split text into terms.
type searchRules struct {
	MinWordLength int      `yaml:"min_word_length"`
	Stopwords     []string `yaml:"stopwords"`
	StemSuffixes  []string `yaml:"stem_suffixes"`
	StemKeep      int      `yaml:"stem_keep"`

	SkipHidden       bool     `yaml:"skip_hidden"`
	SkipDirs         []string `yaml:"skip_dirs"`
	Extensions       []string `yaml:"extensions"`
	SecretNames      []string `yaml:"secret_names"`
	SecretPrefixes   []string `yaml:"secret_prefixes"`
	SecretExtensions []string `yaml:"secret_extensions"`
	SecretDirs       []string `yaml:"secret_dirs"`
}

// parseSearchRules reads search.yaml. The minimum word length must be at
// least 1 and stem_keep not negative.
func parseSearchRules(data []byte) (searchRules, error) {
	var r searchRules
	if err := yaml.Unmarshal(data, &r); err != nil {
		return r, err
	}
	if r.MinWordLength < 1 {
		return r, errors.New("min_word_length must be at least 1")
	}
	if r.StemKeep < 0 {
		return r, errors.New("stem_keep must not be negative")
	}
	if len(r.Extensions) == 0 || len(r.SecretNames) == 0 || len(r.SecretDirs) == 0 {
		return r, errors.New("extensions, secret_names and secret_dirs must be set")
	}
	return r, nil
}

//nolint:gochecknoglobals // loaded once from the knowledge asset set
var searchRulesItem = &yamlItem[searchRules]{name: searchItem, parse: parseSearchRules}

// SearchTokenizer is the local search tokenizer with search.yaml's word rules.
// A missing or broken file is an error: guessing other rules would split text
// differently from the indexes already built.
func SearchTokenizer() (*kgsearch.Tokenizer, error) {
	r, err := searchRulesItem.get()
	if err != nil {
		return nil, fmt.Errorf("search word rules: %w", err)
	}
	return kgsearch.NewTokenizer(kgsearch.TokenizerOptions{
		MinWordLength: r.MinWordLength, Stopwords: r.Stopwords, StemSuffixes: r.StemSuffixes, StemKeep: r.StemKeep,
	}), nil
}

// SearchFilter is the local search file filter with search.yaml's rules: which
// folders are walked, which files are indexed, and which are secrets. A
// missing or broken file is an error; callers then index nothing and treat
// every file as a secret.
func SearchFilter() (*kgsearch.Filter, error) {
	searchFilterCache.Lock()
	defer searchFilterCache.Unlock()
	if searchFilterCache.f != nil {
		return searchFilterCache.f, nil
	}
	r, err := searchRulesItem.get()
	if err != nil {
		return nil, fmt.Errorf("search file rules: %w", err)
	}
	searchFilterCache.f = kgsearch.NewFilter(kgsearch.FilterOptions{
		SkipHidden: r.SkipHidden, SkipDirs: r.SkipDirs, Extensions: r.Extensions,
		SecretNames: r.SecretNames, SecretPrefixes: r.SecretPrefixes, SecretExts: r.SecretExtensions,
		SecretDirs: r.SecretDirs,
	})
	return searchFilterCache.f, nil
}

// searchFilterCache keeps the filter once its rules load; a walk asks for it
// on every entry.
//
//nolint:gochecknoglobals // built once from the knowledge asset set
var searchFilterCache struct {
	sync.Mutex
	f *kgsearch.Filter
}
