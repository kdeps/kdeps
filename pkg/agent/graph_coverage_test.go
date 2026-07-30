// Copyright 2026 kdeps KVK 94834768
// Licensed under the Apache License, Version 2.0.

package agent

import (
	"strings"
	"testing"
)

func TestInMemoryGraphRepositoryAndServices(t *testing.T) {
	repo := newInMemoryGraphRepository(map[string][]string{
		"a": {"b", "c"},
		"b": {"d"},
		"c": {},
		"d": {},
	})
	all := repo.GetAllDependencies()
	if len(all["a"]) != 2 {
		t.Fatalf("all deps: %+v", all)
	}
	rev := repo.GetReverseDependencies()
	if len(rev["b"]) == 0 {
		t.Fatalf("reverse: %+v", rev)
	}

	fmttr := newArrowPathFormatter()
	if got := fmttr.FormatPath(&graphPath{Nodes: nil, Direction: "forward"}); got != "" {
		t.Fatalf("empty path: %q", got)
	}
	if got := fmttr.FormatPath(&graphPath{Nodes: []string{"a", "b"}, Direction: "forward"}); got != "a -> b" {
		t.Fatalf("forward: %q", got)
	}
	if got := fmttr.FormatPath(&graphPath{Nodes: []string{"a", "b"}, Direction: "reverse"}); !strings.Contains(got, "<-") {
		t.Fatalf("reverse: %q", got)
	}

	var lines []string
	writer := graphOutputWriterFunc(func(s string) { lines = append(lines, s) })
	pathSvc := newGraphPathService(fmttr, writer)
	if got := pathSvc.ConstructPath([]string{"x", "y"}, "forward"); got != "x -> y" {
		t.Fatalf("construct: %q", got)
	}
	pathSvc.PrintPath([]string{"p", "q"}, "forward")
	if len(lines) != 1 {
		t.Fatalf("print lines: %v", lines)
	}

	depSvc := newGraphDependencyService(repo, pathSvc)
	if got := depSvc.ListDirectDependencies("a"); len(got) != 2 {
		t.Fatalf("direct: %v", got)
	}
	if got := depSvc.ListDirectDependencies("missing"); len(got) != 0 {
		t.Fatalf("missing direct: %v", got)
	}
	rec := depSvc.ListRecursiveDependencies("a")
	if len(rec) < 2 {
		t.Fatalf("recursive: %v", rec)
	}
	// cycle: d -> a
	if im, ok := repo.(*inMemoryGraphRepository); ok {
		im.dependencies["d"] = []string{"a"}
	}
	_ = depSvc.ListRecursiveDependencies("a")
}

type graphOutputWriterFunc func(string)

func (f graphOutputWriterFunc) WriteLine(s string) { f(s) }
