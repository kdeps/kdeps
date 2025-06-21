package resolver_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
)

func TestFormatResponseData_Comprehensive(t *testing.T) {
	// Test with nil response
	result := resolver.FormatResponseData(nil)
	if result != "" {
		t.Errorf("expected empty string for nil response, got %s", result)
	}

	// Test with empty data
	emptyResponse := &apiserverresponse.APIServerResponseBlock{
		Data: []interface{}{},
	}
	result = resolver.FormatResponseData(emptyResponse)
	if result != "" {
		t.Errorf("expected empty string for empty data, got %s", result)
	}

	// Test with nil data
	nilDataResponse := &apiserverresponse.APIServerResponseBlock{
		Data: nil,
	}
	result = resolver.FormatResponseData(nilDataResponse)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %s", result)
	}

	// Test with valid data
	validResponse := &apiserverresponse.APIServerResponseBlock{
		Data: []interface{}{
			"test string",
			map[string]interface{}{"key": "value"},
		},
	}
	result = resolver.FormatResponseData(validResponse)
	if result == "" {
		t.Error("expected non-empty result for valid data")
	}
	if len(result) < 50 {
		t.Errorf("expected substantial result, got %s", result)
	}
}

func TestFormatResponseMeta_Comprehensive(t *testing.T) {
	// Test with nil meta
	result := resolver.FormatResponseMeta("test-id", nil)
	if result == "" {
		t.Error("expected non-empty result for nil meta")
	}
	if len(result) < 20 {
		t.Errorf("expected substantial result, got %s", result)
	}

	// Test with empty meta
	emptyMeta := &apiserverresponse.APIServerResponseMetaBlock{
		Headers:    &map[string]string{},
		Properties: &map[string]string{},
	}
	result = resolver.FormatResponseMeta("test-id", emptyMeta)
	if result == "" {
		t.Error("expected non-empty result for empty meta")
	}

	// Test with valid meta
	validMeta := &apiserverresponse.APIServerResponseMetaBlock{
		Headers: &map[string]string{
			"Content-Type": "application/json",
		},
		Properties: &map[string]string{
			"status": "success",
		},
	}
	result = resolver.FormatResponseMeta("test-id", validMeta)
	if result == "" {
		t.Error("expected non-empty result for valid meta")
	}
}

func TestFormatMap_Comprehensive(t *testing.T) {
	// Test with empty map
	emptyMap := make(map[interface{}]interface{})
	result := resolver.FormatMap(emptyMap)
	if result == "" {
		t.Error("expected non-empty result for empty map")
	}

	// Test with simple map
	simpleMap := map[interface{}]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	result = resolver.FormatMap(simpleMap)
	if result == "" {
		t.Error("expected non-empty result for simple map")
	}
	if len(result) < 30 {
		t.Errorf("expected substantial result, got %s", result)
	}

	// Test with nested map
	nestedMap := map[interface{}]interface{}{
		"outer": map[string]interface{}{
			"inner": "value",
		},
	}
	result = resolver.FormatMap(nestedMap)
	if result == "" {
		t.Error("expected non-empty result for nested map")
	}
}

func TestFormatValue_Comprehensive(t *testing.T) {
	// Test with nil
	result := resolver.FormatValue(nil)
	if result != "null" {
		t.Errorf("expected 'null' for nil, got %s", result)
	}

	// Test with string
	result = resolver.FormatValue("test string")
	if result == "" {
		t.Error("expected non-empty result for string")
	}

	// Test with number
	result = resolver.FormatValue(42)
	if result == "" {
		t.Error("expected non-empty result for number")
	}

	// Test with map[string]interface{}
	stringMap := map[string]interface{}{
		"key": "value",
	}
	result = resolver.FormatValue(stringMap)
	if result == "" {
		t.Error("expected non-empty result for string map")
	}

	// Test with map[interface{}]interface{}
	interfaceMap := map[interface{}]interface{}{
		"key": "value",
	}
	result = resolver.FormatValue(interfaceMap)
	if result == "" {
		t.Error("expected non-empty result for interface map")
	}

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
	}
	testStruct := TestStruct{Name: "test", Value: 42}
	result = resolver.FormatValue(testStruct)
	if result == "" {
		t.Error("expected non-empty result for struct")
	}

	// Test with pointer to struct
	result = resolver.FormatValue(&testStruct)
	if result == "" {
		t.Error("expected non-empty result for struct pointer")
	}
}

func TestStructToMap_Comprehensive(t *testing.T) {
	type TestStruct struct {
		Name  string
		Value int
		Flag  bool
	}

	testStruct := TestStruct{
		Name:  "test",
		Value: 42,
		Flag:  true,
	}

	result := resolver.StructToMap(testStruct)
	if len(result) != 3 {
		t.Errorf("expected 3 fields, got %d", len(result))
	}

	if result["Name"] != "test" {
		t.Errorf("expected Name to be 'test', got %v", result["Name"])
	}

	if result["Value"] != 42 {
		t.Errorf("expected Value to be 42, got %v", result["Value"])
	}

	if result["Flag"] != true {
		t.Errorf("expected Flag to be true, got %v", result["Flag"])
	}

	// Test with pointer
	result = resolver.StructToMap(&testStruct)
	if len(result) != 3 {
		t.Errorf("expected 3 fields for pointer, got %d", len(result))
	}
}

func TestFormatDataValue_Comprehensive(t *testing.T) {
	// Test with string
	result := resolver.FormatDataValue("test string")
	if result == "" {
		t.Error("expected non-empty result for string")
	}
	if len(result) < 100 {
		t.Errorf("expected substantial result, got %s", result)
	}

	// Test with map
	testMap := map[string]interface{}{
		"key": "value",
	}
	result = resolver.FormatDataValue(testMap)
	if result == "" {
		t.Error("expected non-empty result for map")
	}
}

func TestFormatErrors_Comprehensive(t *testing.T) {
	logger := logging.NewTestLogger()

	// Test with nil errors
	result := resolver.FormatErrors(nil, logger)
	if result != "" {
		t.Errorf("expected empty string for nil errors, got %s", result)
	}

	// Test with empty errors
	emptyErrors := []*apiserverresponse.APIServerErrorsBlock{}
	result = resolver.FormatErrors(&emptyErrors, logger)
	if result != "" {
		t.Errorf("expected empty string for empty errors, got %s", result)
	}

	// Test with valid errors
	validErrors := []*apiserverresponse.APIServerErrorsBlock{
		{
			Code:    400,
			Message: "Bad Request",
		},
		{
			Code:    500,
			Message: "Internal Server Error",
		},
	}
	result = resolver.FormatErrors(&validErrors, logger)
	if result == "" {
		t.Error("expected non-empty result for valid errors")
	}
	if len(result) < 50 {
		t.Errorf("expected substantial result, got %s", result)
	}

	// Test with nil error in slice
	mixedErrors := []*apiserverresponse.APIServerErrorsBlock{
		{
			Code:    400,
			Message: "Bad Request",
		},
		nil,
		{
			Code:    500,
			Message: "Internal Server Error",
		},
	}
	result = resolver.FormatErrors(&mixedErrors, logger)
	if result == "" {
		t.Error("expected non-empty result for mixed errors")
	}
}

func TestDecodeErrorMessage_Comprehensive(t *testing.T) {
	logger := logging.NewTestLogger()

	// Test with empty message
	result := resolver.DecodeErrorMessage("", logger)
	if result != "" {
		t.Errorf("expected empty string for empty message, got %s", result)
	}

	// Test with regular message
	result = resolver.DecodeErrorMessage("test message", logger)
	if result != "test message" {
		t.Errorf("expected 'test message', got %s", result)
	}

	// Test with base64 encoded message
	base64Message := "dGVzdCBtZXNzYWdl" // "test message" in base64
	result = resolver.DecodeErrorMessage(base64Message, logger)
	if result != "test message" {
		t.Errorf("expected decoded 'test message', got %s", result)
	}
}

func TestDependencyResolver_EnsureResponsePklFileNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:              fs,
		Logger:          logger,
		Context:         ctx,
		ResponsePklFile: "/test/response.pkl",
	}

	// Test when file doesn't exist
	err := dr.EnsureResponsePklFileNotExists()
	if err != nil {
		t.Errorf("unexpected error when file doesn't exist: %v", err)
	}

	// Test when file exists
	testContent := []byte("test content")
	err = afero.WriteFile(fs, "/test/response.pkl", testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = dr.EnsureResponsePklFileNotExists()
	if err != nil {
		t.Errorf("unexpected error when removing existing file: %v", err)
	}

	// Verify file was removed
	exists, err := afero.Exists(fs, "/test/response.pkl")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if exists {
		t.Error("file should have been removed")
	}
}

func TestDependencyResolver_EvalPklFormattedResponseFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	dr := &resolver.DependencyResolver{
		Fs:              fs,
		Logger:          logger,
		Context:         ctx,
		ResponsePklFile: "/test/response.pkl",
	}

	// Test when file doesn't exist
	_, err := dr.EvalPklFormattedResponseFile()
	if err == nil {
		t.Error("expected error when file doesn't exist")
	}

	// Test with invalid file extension
	dr.ResponsePklFile = "/test/response.txt"
	testContent := []byte("test content")
	err = afero.WriteFile(fs, "/test/response.txt", testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = dr.EvalPklFormattedResponseFile()
	if err == nil {
		t.Error("expected error for invalid file extension")
	}
}
