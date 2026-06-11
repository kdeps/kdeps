// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package domain

import "testing"

func TestBuildInlineResourceTypes_SkipsPrimaryOnly(t *testing.T) {
	t.Parallel()

	saved := resourceExecCatalog
	t.Cleanup(func() { resourceExecCatalog = saved })

	resourceExecCatalog = append([]ResourceExecCatalogEntry{}, resourceExecCatalog...)
	resourceExecCatalog = append(resourceExecCatalog, ResourceExecCatalogEntry{
		Name:            "primaryOnly",
		PrimaryOnly:     true,
		PresentResource: func(*Resource) bool { return false },
		PresentAction:   func(*ActionConfig) bool { return false },
	})

	for _, name := range InlineResourceTypeNames() {
		if name == "primaryOnly" {
			t.Fatal("primaryOnly must not appear in inline types")
		}
	}
}
