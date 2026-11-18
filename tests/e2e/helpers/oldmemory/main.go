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

// Command oldmemory writes a memory database in the layout kdeps used before
// memory moved into the knowledge vault (~/.kdeps/memory/<encoded dir>/
// memory.bolt), so the E2E tests can check the one-time move.
//
//	go run ./tests/e2e/helpers/oldmemory <home> <project dir> <key> <value>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	argCount  = 5
	usageExit = 2
	dirPerm   = 0o750
	filePerm  = 0o600
)

func main() {
	if len(os.Args) != argCount {
		fmt.Fprintln(os.Stderr, "usage: oldmemory <home> <project dir> <key> <value>")
		os.Exit(usageExit)
	}
	home, dir, key, value := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	if err := os.Chdir(dir); err != nil {
		fail(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	clean := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(strings.TrimLeft(wd, "/\\"))
	path := filepath.Join(home, ".kdeps", "memory", "--"+clean+"--", "memory.bolt")
	if err = os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		fail(err)
	}
	db, err := bolt.Open(path, filePerm, nil)
	if err != nil {
		fail(err)
	}
	defer db.Close()
	now := time.Now().UnixMilli()
	entry, _ := json.Marshal(map[string]any{
		"key": key, "value": value, "type": "fact", "createdAt": now, "updatedAt": now,
	})
	err = db.Update(func(tx *bolt.Tx) error {
		b, bErr := tx.CreateBucketIfNotExists([]byte("agent_memory"))
		if bErr != nil {
			return bErr
		}
		return b.Put([]byte(key), entry)
	})
	if err != nil {
		fail(err)
	}
	fmt.Fprintln(os.Stdout, path)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
