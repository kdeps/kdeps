package utils

import (
	"fmt"
	"sort"
	"strings"
)

func EncodePklMap(m *map[string]string) string {
	if m == nil {
		return "{}\n"
	}
	var builder strings.Builder
	builder.WriteString("{\n")
	// Sort keys for deterministic output
	keys := make([]string, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := (*m)[k]
		builder.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", k, EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}

func EncodePklSlice(s *[]string) string {
	if s == nil {
		return "{}\n"
	}
	var builder strings.Builder
	builder.WriteString("{\n")
	for _, v := range *s {
		builder.WriteString(fmt.Sprintf("      \"%s\"\n", EncodeValue(v)))
	}
	builder.WriteString("    }\n")
	return builder.String()
}
