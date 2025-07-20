package utils

import (
	"fmt"
	"strings"
)

func EncodePklMap(m *map[string]string) string {
	if m == nil {
		return "{}\n"
	}
	var builder strings.Builder
	builder.WriteString("{\n")
	for k, v := range *m {
		builder.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", k, v))
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
		builder.WriteString(fmt.Sprintf("      \"%s\"\n", v))
	}
	builder.WriteString("    }\n")
	return builder.String()
}
