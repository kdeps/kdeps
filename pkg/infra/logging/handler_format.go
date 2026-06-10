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

package logging

import (
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// formatSourceLocation returns a short file:line source label when enabled.
func (h *PrettyHandler) formatSourceLocation(pc uintptr) string {
	if !h.opts.AddSource || pc == 0 {
		return ""
	}

	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	if f.File == "" {
		return ""
	}

	file := f.File
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return fmt.Sprintf("%s:%d", file, f.Line)
}

// formatColoredBool returns a colorized boolean string.
func (h *PrettyHandler) formatColoredBool(val bool) string {
	if val {
		return h.colorize(colorGreen, "true")
	}
	return h.colorize(colorRed, "false")
}

// parentIndent removes one indent level from the given indent string.
func parentIndent(indent string) string {
	if len(indent) < indentBaseLength {
		return ""
	}
	return indent[:len(indent)-indentBaseLength]
}

// formatLevel formats the log level with color and badge.
func (h *PrettyHandler) formatLevel(level slog.Level) string {
	kdeps_debug.Log("enter: formatLevel")
	var color, badge string

	switch level {
	case slog.LevelDebug:
		color = colorCyan
		badge = "DEBUG"
	case slog.LevelInfo:
		color = colorGreen
		badge = " INFO"
	case slog.LevelWarn:
		color = colorYellow
		badge = " WARN"
	case slog.LevelError:
		color = colorRed
		badge = "ERROR"
	default:
		color = colorWhite
		badge = fmt.Sprintf("%5s", level.String())
	}

	if h.opts.DisableColors {
		return fmt.Sprintf("[%s]", badge)
	}

	// Rails-style colored badge
	return fmt.Sprintf("%s[%s]%s", color, badge, colorReset)
}

// formatAttr formats a single attribute.
func (h *PrettyHandler) formatAttr(buf *strings.Builder, attr slog.Attr, indent string) {
	kdeps_debug.Log("enter: formatAttr")
	key := attr.Key
	value := attr.Value

	// Format key
	buf.WriteString(indent)
	buf.WriteString(h.colorize(colorCyan, key))
	buf.WriteString(": ")

	// Format value
	h.formatValue(buf, value, indent+"  ")
}

// formatValue formats a value with proper indentation and pretty printing.
func (h *PrettyHandler) formatValue(buf *strings.Builder, v slog.Value, indent string) {
	kdeps_debug.Log("enter: formatValue")
	switch v.Kind() {
	case slog.KindString:
		buf.WriteString(h.colorize(colorGreen, fmt.Sprintf("%q", v.String())))
	case slog.KindInt64:
		buf.WriteString(h.colorize(colorBlue, strconv.FormatInt(v.Int64(), 10)))
	case slog.KindUint64:
		buf.WriteString(h.colorize(colorBlue, strconv.FormatUint(v.Uint64(), 10)))
	case slog.KindFloat64:
		buf.WriteString(h.colorize(colorBlue, fmt.Sprintf("%.2f", v.Float64())))
	case slog.KindBool:
		buf.WriteString(h.formatColoredBool(v.Bool()))
	case slog.KindDuration:
		buf.WriteString(h.colorize(colorMagenta, v.Duration().String()))
	case slog.KindTime:
		buf.WriteString(h.colorize(colorMagenta, v.Time().Format(time.RFC3339)))
	case slog.KindAny:
		anyVal := v.Any()
		h.FormatAny(buf, anyVal, indent)
	case slog.KindGroup:
		// Group attributes
		buf.WriteString("\n")
		attrs := v.Group()
		for _, attr := range attrs {
			buf.WriteString(indent)
			h.formatAttr(buf, attr, indent)
			buf.WriteString("\n")
		}
		buf.WriteString(parentIndent(indent))
	case slog.KindLogValuer:
		// LogValuer - call LogValue() and format the result
		logVal := v.LogValuer()
		// LogValuer() always returns a non-nil Value
		h.formatValue(buf, logVal.LogValue(), indent)
	default:
		// Unknown kinds are not expected from slog; v.Any() and v.String() panic on invalid kinds.
		buf.WriteString(h.colorize(colorGray, "<unknown>"))
	}
}

// FormatAny formats any value type with pretty printing.
func (h *PrettyHandler) FormatAny(buf *strings.Builder, v interface{}, indent string) {
	kdeps_debug.Log("enter: FormatAny")
	switch val := v.(type) {
	case string:
		buf.WriteString(h.colorize(colorGreen, fmt.Sprintf("%q", val)))
	case int, int8, int16, int32, int64:
		buf.WriteString(h.colorize(colorBlue, fmt.Sprintf("%d", val)))
	case uint, uint8, uint16, uint32, uint64:
		buf.WriteString(h.colorize(colorBlue, fmt.Sprintf("%d", val)))
	case float32, float64:
		buf.WriteString(h.colorize(colorBlue, fmt.Sprintf("%.2f", val)))
	case bool:
		buf.WriteString(h.formatColoredBool(val))
	case error:
		buf.WriteString(h.colorize(colorRed, fmt.Sprintf("%q", val.Error())))
	case map[string]interface{}:
		h.formatMap(buf, val, indent)
	case []interface{}:
		h.formatSlice(buf, val, indent)
	default:
		// Try JSON marshaling for complex types
		if jsonBytes, err := jsonMarshalIndent(val, indent, "  "); err == nil {
			buf.Write(jsonBytes)
		} else {
			buf.WriteString(h.colorize(colorGray, fmt.Sprintf("%v", val)))
		}
	}
}

// formatMap formats a map with pretty printing.
func (h *PrettyHandler) formatMap(buf *strings.Builder, m map[string]interface{}, indent string) {
	kdeps_debug.Log("enter: formatMap")
	buf.WriteString("{\n")
	first := true
	for k, v := range m {
		if !first {
			buf.WriteString(",\n")
		}
		first = false
		buf.WriteString(indent)
		buf.WriteString(h.colorize(colorCyan, fmt.Sprintf("%q", k)))
		buf.WriteString(": ")
		h.FormatAny(buf, v, indent+"  ")
	}
	buf.WriteString("\n")
	buf.WriteString(parentIndent(indent))
	buf.WriteString("}")
}

// formatSlice formats a slice with pretty printing.
func (h *PrettyHandler) formatSlice(buf *strings.Builder, s []interface{}, indent string) {
	kdeps_debug.Log("enter: formatSlice")
	buf.WriteString("[\n")
	for i, v := range s {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString(indent)
		buf.WriteString("- ")
		h.FormatAny(buf, v, indent+"  ")
	}
	buf.WriteString("\n")
	buf.WriteString(parentIndent(indent))
	buf.WriteString("]")
}
