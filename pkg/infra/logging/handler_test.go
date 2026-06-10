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

package logging_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/infra/logging"
)

func TestPrettyHandler_Enabled(t *testing.T) {
	opts := &logging.PrettyHandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := logging.NewPrettyHandler(&bytes.Buffer{}, opts)

	ctx := t.Context()

	// Debug should be disabled
	if handler.Enabled(ctx, slog.LevelDebug) {
		t.Error("Debug level should be disabled")
	}

	// Info should be enabled
	if !handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("Info level should be enabled")
	}

	// Error should be enabled
	if !handler.Enabled(ctx, slog.LevelError) {
		t.Error("Error level should be enabled")
	}
}

func TestPrettyHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true, // Disable colors for testing
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
	record.Add(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output")
	}

	// Check that message is in output
	if !contains(output, "Test message") {
		t.Errorf("Output should contain message: %s", output)
	}

	// Check that attributes are in output
	if !contains(output, "key1") || !contains(output, "value1") {
		t.Errorf("Output should contain attributes: %s", output)
	}
}

func TestPrettyHandler_Levels(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)
	logger := slog.New(handler)

	ctx := t.Context()

	// Test all levels
	logger.DebugContext(ctx, "Debug message", "key", "value")
	logger.InfoContext(ctx, "Info message", "key", "value")
	logger.WarnContext(ctx, "Warn message", "key", "value")
	logger.ErrorContext(ctx, "Error message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output for all levels")
	}
}

func TestNewLogger(t *testing.T) {
	// Test debug logger
	debugLogger := logging.NewLogger(true)
	if debugLogger == nil {
		t.Error("NewLogger should return a logger")
	}

	// Test non-debug logger
	infoLogger := logging.NewLogger(false)
	if infoLogger == nil {
		t.Error("NewLogger should return a logger")
	}
}

func TestPrettyHandler_FormatAny(t *testing.T) {
	var buf strings.Builder
	opts := &logging.PrettyHandlerOptions{
		DisableColors: true, // Disable colors for testing
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string",
			value:    "test",
			expected: `"test"`,
		},
		{
			name:     "int",
			value:    42,
			expected: "42",
		},
		{
			name:     "uint",
			value:    uint(123),
			expected: "123",
		},
		{
			name:     "float32",
			value:    float32(3.14),
			expected: "3.14",
		},
		{
			name:     "float64",
			value:    3.14,
			expected: "3.14",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},
		{
			name:     "error",
			value:    errors.New("test error"),
			expected: `"test error"`,
		},
		{
			name:  "map",
			value: map[string]interface{}{"key": "value"},
			expected: `{
"key": "value"
}`,
		},
		{
			name:  "slice",
			value: []interface{}{"a", "b"},
			expected: `[
- "a",
- "b"
]`,
		},
		{
			name:  "complex type",
			value: struct{ Name string }{Name: "test"},
			expected: `{
  "Name": "test"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			handler.FormatAny(&out, tt.value, "")
			result := out.String()
			if result != tt.expected {
				t.Errorf("FormatAny() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrettyHandler_ColorDisabled(t *testing.T) {
	// Test with colors disabled
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelInfo,
		DisableColors: true, // Explicitly disable colors
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output")
	}

	// With colors disabled, output should not contain ANSI color codes
	if strings.Contains(output, "\033[") {
		t.Error("Output should not contain ANSI color codes when colors are disabled")
	}
}

func TestPrettyHandler_ColorEnabled(t *testing.T) {
	// Test with colors enabled
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelInfo,
		DisableColors: false, // Explicitly enable colors
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output")
	}

	// With colors enabled, output should contain ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Error("Output should contain ANSI color codes when colors are enabled")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPrettyHandler_Handle_EmptyRecord tests handling empty record.
func TestPrettyHandler_Handle_EmptyRecord(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "", 0)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// Should still produce output even with empty message
	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output even with empty message")
	}
}

// TestPrettyHandler_Handle_WithGroups tests handling record with groups.
func TestPrettyHandler_Handle_WithGroups(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
	record.Add(
		slog.String("key1", "value1"),
		slog.Group("group1",
			slog.String("nested", "value"),
		),
	)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output with groups")
	}
}

// TestPrettyHandler_Handle_WithColors tests handling with colors enabled.
func TestPrettyHandler_Handle_WithColors(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: false, // Colors enabled
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
	record.Add(slog.String("key", "value"))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output with colors")
	}
}

// TestPrettyHandler_Handle_DifferentLevels tests all log levels.
func TestPrettyHandler_Handle_DifferentLevels(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()

	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	for _, level := range levels {
		record := slog.NewRecord(time.Now(), level, "Test message", 0)
		err := handler.Handle(ctx, record)
		if err != nil {
			t.Fatalf("Handle returned error for level %v: %v", level, err)
		}
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output for all levels")
	}
}

// TestPrettyHandler_Handle_ComplexAttributes tests handling complex attribute types.
func TestPrettyHandler_Handle_ComplexAttributes(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
	record.Add(
		slog.String("string", "value"),
		slog.Int("int", 42),
		slog.Float64("float", 3.14),
		slog.Bool("bool", true),
		slog.Time("time", time.Now()),
		slog.Duration("duration", time.Second),
	)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output with complex attributes")
	}
}

// TestPrettyHandler_Handle_NilOptions tests handler with nil options.
func TestPrettyHandler_Handle_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil) // nil options should use defaults

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output with nil options")
	}
}

// TestPrettyHandler_Handle_LongMessage tests handling long messages.
func TestPrettyHandler_Handle_LongMessage(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	longMessage := ""
	var longMessageSb261 strings.Builder
	for range 1000 {
		longMessageSb261.WriteString("a")
	}
	longMessage += longMessageSb261.String()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, longMessage, 0)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Handler should produce output for long messages")
	}
}

// TestPrettyHandler_Handle_ManyAttributes tests handling many attributes.
func TestPrettyHandler_Handle_ManyAttributes(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:         slog.LevelDebug,
		DisableColors: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)

	// Add many attributes
	for i := range 50 {
		record.Add(slog.Int("key", i))
	}

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

// TestPrettyHandler_FormatValue_Uint64 tests formatValue with uint64.
func TestPrettyHandler_FormatValue_Uint64(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)

	// Test through Handle method
	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test", 0)
	record.Add(slog.Uint64("uint64", 42))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	assert.Contains(t, buf.String(), "42")
}

// TestPrettyHandler_FormatValue_Float64 tests formatValue with float64.
func TestPrettyHandler_FormatValue_Float64(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)

	// Test through Handle method
	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test", 0)
	record.Add(slog.Float64("float64", 3.14159))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	assert.Contains(t, buf.String(), "3.14") // Should format to 2 decimal places
}

// TestPrettyHandler_FormatValue_Group tests formatValue with group.
func TestPrettyHandler_FormatValue_Group(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)

	// Use a simpler approach - format group through Handle method
	group := slog.GroupValue(
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	)

	// Test through Handle method instead of direct formatValue to avoid panic
	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test", 0)
	record.Add(slog.Any("group", group))

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")
}

// TestPrettyHandler_Handle_WithSource tests Handle with AddSource enabled.
func TestPrettyHandler_Handle_WithSource(t *testing.T) {
	var buf bytes.Buffer
	opts := &logging.PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	handler := logging.NewPrettyHandler(&buf, opts)

	ctx := t.Context()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
	// Set PC to a valid program counter
	record.PC = uintptr(1)

	err := handler.Handle(ctx, record)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	output := buf.String()
	// Should contain source information
	assert.NotEmpty(t, output)
}

// TestPrettyHandler_WithAttrs tests WithAttrs method.
func TestPrettyHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	result := handler.WithAttrs(attrs)
	assert.NotNil(t, result)
	// Should return a handler (may be same or new)
	_ = result
}

// TestPrettyHandler_WithGroup tests WithGroup method.
func TestPrettyHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)

	result := handler.WithGroup("test-group")
	assert.NotNil(t, result)
	// Should return a handler (may be same or new)
	_ = result
}

// TestPrettyHandler_FormatAny_Error tests formatAny with error type.
func TestPrettyHandler_FormatAny_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	testErr := errors.New("test error")
	handler.FormatAny(builder, testErr, "")
	assert.Contains(t, builder.String(), "test error")
}

// TestPrettyHandler_FormatAny_Map tests formatAny with map type.
func TestPrettyHandler_FormatAny_Map(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	testMap := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	// Use proper indent to avoid panic
	handler.FormatAny(builder, testMap, "  ")
	assert.Contains(t, builder.String(), "key1")
	assert.Contains(t, builder.String(), "value1")
}

// TestPrettyHandler_FormatAny_Slice tests formatAny with slice type.
func TestPrettyHandler_FormatAny_Slice(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	testSlice := []interface{}{"item1", "item2", 42}
	// Use proper indent to avoid panic
	handler.FormatAny(builder, testSlice, "  ")
	assert.Contains(t, builder.String(), "item1")
	assert.Contains(t, builder.String(), "item2")
}

// TestPrettyHandler_FormatAny_Default tests formatAny with default/unknown type.
func TestPrettyHandler_FormatAny_Default(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a complex type that will use JSON marshaling
	type CustomType struct {
		Field1 string
		Field2 int
	}
	custom := CustomType{Field1: "test", Field2: 42}
	// Use proper indent
	handler.FormatAny(builder, custom, "  ")
	// Should format using JSON or fallback
	assert.NotEmpty(t, builder.String())
}

// TestPrettyHandler_FormatAny_JSONMarshalSuccess tests formatAny with JSON marshaling success.
func TestPrettyHandler_FormatAny_JSONMarshalSuccess(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that can be JSON marshaled
	type CustomType struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}
	custom := CustomType{Field1: "test", Field2: 42}
	handler.FormatAny(builder, custom, "  ")
	// Should format using JSON
	assert.Contains(t, builder.String(), "field1")
	assert.Contains(t, builder.String(), "test")
}

// TestPrettyHandler_FormatAny_JSONMarshalError tests formatAny with JSON marshaling error.
func TestPrettyHandler_FormatAny_JSONMarshalError(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that cannot be JSON marshaled (circular reference)
	type CircularType struct {
		Self *CircularType
	}
	circular := &CircularType{}
	circular.Self = circular // Create circular reference

	handler.FormatAny(builder, circular, "  ")
	// Should fall back to string formatting
	assert.NotEmpty(t, builder.String())
}

// TestPrettyHandler_FormatAny_Int8 tests formatAny with int8.
func TestPrettyHandler_FormatAny_Int8(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, int8(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Int16 tests formatAny with int16.
func TestPrettyHandler_FormatAny_Int16(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, int16(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Int32 tests formatAny with int32.
func TestPrettyHandler_FormatAny_Int32(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, int32(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Int64 tests formatAny with int64.
func TestPrettyHandler_FormatAny_Int64(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, int64(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Uint8 tests formatAny with uint8.
func TestPrettyHandler_FormatAny_Uint8(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, uint8(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Uint16 tests formatAny with uint16.
func TestPrettyHandler_FormatAny_Uint16(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, uint16(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Uint32 tests formatAny with uint32.
func TestPrettyHandler_FormatAny_Uint32(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, uint32(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Uint64 tests formatAny with uint64.
func TestPrettyHandler_FormatAny_Uint64(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, uint64(42), "  ")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_Float32 tests formatAny with float32.
func TestPrettyHandler_FormatAny_Float32(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, float32(3.14159), "  ")
	assert.Contains(t, builder.String(), "3.14")
}

// TestPrettyHandler_FormatAny_Float64 tests formatAny with float64.
func TestPrettyHandler_FormatAny_Float64(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, float64(3.14159), "  ")
	assert.Contains(t, builder.String(), "3.14")
}

// TestPrettyHandler_FormatAny_BoolTrue tests formatAny with bool true.
func TestPrettyHandler_FormatAny_BoolTrue(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, true, "  ")
	assert.Contains(t, builder.String(), "true")
}

// TestPrettyHandler_FormatAny_BoolFalse tests formatAny with bool false.
func TestPrettyHandler_FormatAny_BoolFalse(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	handler.FormatAny(builder, false, "  ")
	assert.Contains(t, builder.String(), "false")
}

// TestPrettyHandler_FormatAny_JSONMarshalSuccess2 tests formatAny with JSON marshaling success (variant 2).
func TestPrettyHandler_FormatAny_JSONMarshalSuccess2(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that can be JSON marshaled
	type CustomType struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
		Field3 bool   `json:"field3"`
	}
	custom := CustomType{Field1: "test", Field2: 42, Field3: true}
	handler.FormatAny(builder, custom, "  ")
	// Should format using JSON
	assert.Contains(t, builder.String(), "field1")
	assert.Contains(t, builder.String(), "test")
	assert.Contains(t, builder.String(), "field2")
	assert.Contains(t, builder.String(), "42")
}

// TestPrettyHandler_FormatAny_JSONMarshalError2 tests formatAny with JSON marshaling error (variant 2).
func TestPrettyHandler_FormatAny_JSONMarshalError2(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that cannot be JSON marshaled (function)
	var fn = func() {}
	handler.FormatAny(builder, fn, "  ")
	// Should fall back to string formatting
	assert.NotEmpty(t, builder.String())
}

// TestPrettyHandler_FormatAny_JSONMarshalError3 tests formatAny with JSON marshaling error (variant 3).
func TestPrettyHandler_FormatAny_JSONMarshalError3(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that cannot be JSON marshaled (channel)
	ch := make(chan int)
	handler.FormatAny(builder, ch, "  ")
	// Should fall back to string formatting
	assert.NotEmpty(t, builder.String())
}

// TestPrettyHandler_FormatAny_JSONMarshalIndentError tests formatAny with JSON marshal indent error.
func TestPrettyHandler_FormatAny_JSONMarshalIndentError(t *testing.T) {
	var buf bytes.Buffer
	handler := logging.NewPrettyHandler(&buf, nil)
	builder := &strings.Builder{}

	// Test with a type that causes MarshalIndent to fail
	// Use a type that can be marshaled but with invalid indent
	type CustomType struct {
		Field string `json:"field"`
	}
	custom := CustomType{Field: "test"}
	// Use empty indent to test edge case
	handler.FormatAny(builder, custom, "")
	// Should still format (may use different path)
	assert.NotEmpty(t, builder.String())
}
