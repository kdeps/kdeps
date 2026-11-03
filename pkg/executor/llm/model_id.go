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

package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// modelIDLen is the short model id length. A shared prefix grows by one
// character until every catalog model has its own id.
const modelIDLen = 8

// modelIDHash is the full hexadecimal id of one model. The short id is a
// prefix that no other model in the loaded catalog shares.
func modelIDHash(backend, id string) string {
	sum := sha256.Sum256([]byte(backend + "\x00" + id))
	return hex.EncodeToString(sum[:])
}

// assignModelIDs sets ModelID on each model. The same backend and API id
// always hash the same way. The prefix length grows only when two models
// would otherwise share it, so one model's id does not depend on catalog order.
func assignModelIDs(models []CloudModel) {
	full := make([]string, len(models))
	for i, m := range models {
		full[i] = modelIDHash(m.Backend, m.ID)
	}
	for i := range models {
		models[i].ModelID = uniqueModelID(full, i)
	}
}

func uniqueModelID(full []string, i int) string {
	n := modelIDLen
	if n > len(full[i]) {
		n = len(full[i])
	}
	for n < len(full[i]) {
		key := full[i][:n]
		if !modelIDPrefixTaken(full, i, key) {
			return key
		}
		n++
	}
	return full[i]
}

func modelIDPrefixTaken(full []string, self int, key string) bool {
	for j := range full {
		if j != self && strings.HasPrefix(full[j], key) {
			return true
		}
	}
	return false
}

// ModelByKey returns the catalog model whose short id is key.
func ModelByKey(key string) (CloudModel, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return CloudModel{}, false
	}
	for _, m := range KnownCloudModels {
		if m.ModelID != "" && strings.EqualFold(m.ModelID, key) {
			return m, true
		}
	}
	return CloudModel{}, false
}

// ModelID returns the short id of a catalog model, or "" when id is not one.
func ModelID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	for _, m := range KnownCloudModels {
		if strings.EqualFold(m.ID, id) {
			return m.ModelID
		}
	}
	return ""
}
