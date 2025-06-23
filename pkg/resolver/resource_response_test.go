package resolver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	err = afero.WriteFile(fs, "/test/response.pkl", testContent, 0o644)
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
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create a temporary directory for PKL files
	tmpDir := t.TempDir()
	responseFile := filepath.Join(tmpDir, "response.pkl")

	dr := &resolver.DependencyResolver{
		Fs:              fs,
		Logger:          logger,
		Context:         ctx,
		ResponsePklFile: responseFile,
	}

	// Test when file doesn't exist
	_, err := dr.EvalPklFormattedResponseFile()
	if err == nil {
		t.Error("expected error when file doesn't exist")
	}

	// Test with invalid file extension
	invalidFile := filepath.Join(tmpDir, "response.txt")
	testContent := []byte("test content")
	err = afero.WriteFile(fs, invalidFile, testContent, 0o644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	dr.ResponsePklFile = invalidFile
	_, err = dr.EvalPklFormattedResponseFile()
	if err == nil {
		t.Error("expected error for invalid file extension")
	}
}

func TestEnsureResponsePklFileNotExists_Additional(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("FileDoesNotExist", func(t *testing.T) {
		tmpDir := t.TempDir()
		responseFile := filepath.Join(tmpDir, "response.pkl")

		resolver := &resolver.DependencyResolver{
			Fs:              fs,
			Logger:          logger,
			Context:         ctx,
			ResponsePklFile: responseFile,
		}

		err := resolver.EnsureResponsePklFileNotExists()
		assert.NoError(t, err)
	})

	t.Run("FileExistsAndDeleted", func(t *testing.T) {
		tmpDir := t.TempDir()
		responseFile := filepath.Join(tmpDir, "response.pkl")

		resolver := &resolver.DependencyResolver{
			Fs:              fs,
			Logger:          logger,
			Context:         ctx,
			ResponsePklFile: responseFile,
		}

		// Create the file first
		err := afero.WriteFile(fs, responseFile, []byte("test content"), 0o644)
		require.NoError(t, err)

		// Verify file exists
		exists, err := afero.Exists(fs, responseFile)
		require.NoError(t, err)
		assert.True(t, exists)

		// Call the function
		err = resolver.EnsureResponsePklFileNotExists()
		assert.NoError(t, err)

		// Verify file was deleted
		exists, err = afero.Exists(fs, responseFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("FileExistsButDeleteFails", func(t *testing.T) {
		// Use a read-only filesystem to simulate delete failure
		baseFs := afero.NewOsFs()
		roFs := afero.NewReadOnlyFs(baseFs)
		tmpDir := t.TempDir()
		responseFile := filepath.Join(tmpDir, "response.pkl")

		resolver := &resolver.DependencyResolver{
			Fs:              roFs,
			Logger:          logger,
			Context:         ctx,
			ResponsePklFile: responseFile,
		}

		// Create the file in the base filesystem
		err := afero.WriteFile(baseFs, responseFile, []byte("test content"), 0o644)
		require.NoError(t, err)

		// Call the function - should fail to delete due to read-only filesystem
		err = resolver.EnsureResponsePklFileNotExists()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete old response file")
	})
}

func TestFormatResponseData_EdgeCases(t *testing.T) {
	t.Run("NilResponse", func(t *testing.T) {
		result := resolver.FormatResponseData(nil)
		assert.Empty(t, result)
	})

	t.Run("NilData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: nil,
		}
		result := resolver.FormatResponseData(response)
		assert.Empty(t, result)
	})

	t.Run("EmptyData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []interface{}{},
		}
		result := resolver.FormatResponseData(response)
		assert.Empty(t, result)
	})

	t.Run("SingleDataItem", func(t *testing.T) {
		data := []interface{}{"test value"}
		response := &apiserverresponse.APIServerResponseBlock{
			Data: data,
		}
		result := resolver.FormatResponseData(response)
		assert.Contains(t, result, "response")
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "test value")
	})
}

func TestFormatResponseMeta_EdgeCases(t *testing.T) {
	t.Run("NilMeta", func(t *testing.T) {
		result := resolver.FormatResponseMeta("test-request", nil)
		assert.Contains(t, result, "requestID = \"test-request\"")
		assert.NotContains(t, result, "headers")
		assert.NotContains(t, result, "properties")
	})

	t.Run("NilHeadersAndProperties", func(t *testing.T) {
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    nil,
			Properties: nil,
		}
		result := resolver.FormatResponseMeta("test-request", meta)
		assert.Contains(t, result, "requestID = \"test-request\"")
		assert.NotContains(t, result, "headers")
		assert.NotContains(t, result, "properties")
	})

	t.Run("EmptyHeadersAndProperties", func(t *testing.T) {
		emptyHeaders := map[string]string{}
		emptyProperties := map[string]string{}
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &emptyHeaders,
			Properties: &emptyProperties,
		}
		result := resolver.FormatResponseMeta("test-request", meta)
		assert.Contains(t, result, "requestID = \"test-request\"")
		assert.NotContains(t, result, "headers")
		assert.NotContains(t, result, "properties")
	})
}

func TestFormatMap_EdgeCases(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		result := resolver.FormatMap(map[interface{}]interface{}{})
		assert.Contains(t, result, "new Mapping {")
		assert.Contains(t, result, "}")
	})

	t.Run("MapWithSpecialCharacters", func(t *testing.T) {
		m := map[interface{}]interface{}{
			"key\"with\"quotes":   "value",
			"key\nwith\nnewlines": "value",
		}
		result := resolver.FormatMap(m)
		assert.Contains(t, result, "new Mapping {")
		assert.Contains(t, result, "key\\\"with\\\"quotes")
		assert.Contains(t, result, "key\nwith\nnewlines")
	})
}

func TestFormatValue_EdgeCases(t *testing.T) {
	t.Run("NilValue", func(t *testing.T) {
		result := resolver.FormatValue(nil)
		assert.Equal(t, "null", result)
	})

	t.Run("StringValue", func(t *testing.T) {
		result := resolver.FormatValue("test string")
		assert.Contains(t, result, "\"\"\"")
		assert.Contains(t, result, "test string")
	})

	t.Run("IntValue", func(t *testing.T) {
		result := resolver.FormatValue(42)
		assert.Contains(t, result, "\"\"\"")
		assert.Contains(t, result, "42")
	})

	t.Run("PointerToValue", func(t *testing.T) {
		value := "test"
		ptr := &value
		result := resolver.FormatValue(ptr)
		assert.Contains(t, result, "test")
	})

	t.Run("NilPointer", func(t *testing.T) {
		var ptr *string
		result := resolver.FormatValue(ptr)
		// The actual output formats nil pointers as <nil> in triple quotes
		assert.Contains(t, result, "<nil>")
		assert.Contains(t, result, "\"\"\"")
	})
}

func TestStructToMap_EdgeCases(t *testing.T) {
	t.Run("PointerToStruct", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
			Field2 int
		}
		value := TestStruct{Field1: "test", Field2: 42}
		ptr := &value

		result := resolver.StructToMap(ptr)
		assert.Equal(t, "test", result["Field1"])
		assert.Equal(t, 42, result["Field2"])
	})

	t.Run("StructWithDifferentTypes", func(t *testing.T) {
		type TestStruct struct {
			StringField string
			IntField    int
			BoolField   bool
			FloatField  float64
		}
		value := TestStruct{
			StringField: "test",
			IntField:    42,
			BoolField:   true,
			FloatField:  3.14,
		}

		result := resolver.StructToMap(value)
		assert.Equal(t, "test", result["StringField"])
		assert.Equal(t, 42, result["IntField"])
		assert.Equal(t, true, result["BoolField"])
		assert.Equal(t, 3.14, result["FloatField"])
	})
}

func TestFormatErrors_EdgeCases(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("NilErrors", func(t *testing.T) {
		result := resolver.FormatErrors(nil, logger)
		assert.Empty(t, result)
	})

	t.Run("EmptyErrorsSlice", func(t *testing.T) {
		errors := []*apiserverresponse.APIServerErrorsBlock{}
		result := resolver.FormatErrors(&errors, logger)
		assert.Empty(t, result)
	})

	t.Run("NilErrorInSlice", func(t *testing.T) {
		errors := []*apiserverresponse.APIServerErrorsBlock{nil}
		result := resolver.FormatErrors(&errors, logger)
		assert.Empty(t, result)
	})

	t.Run("ErrorWithEmptyMessage", func(t *testing.T) {
		errors := []*apiserverresponse.APIServerErrorsBlock{
			{
				Code:    500,
				Message: "",
			},
		}
		result := resolver.FormatErrors(&errors, logger)
		assert.Contains(t, result, "code = 500")
		assert.Contains(t, result, "message = #\"\"\"")
	})
}

func TestDecodeErrorMessage_EdgeCases(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("EmptyMessage", func(t *testing.T) {
		result := resolver.DecodeErrorMessage("", logger)
		assert.Empty(t, result)
	})

	t.Run("NonBase64Message", func(t *testing.T) {
		result := resolver.DecodeErrorMessage("plain text message", logger)
		assert.Equal(t, "plain text message", result)
	})

	t.Run("InvalidBase64Message", func(t *testing.T) {
		result := resolver.DecodeErrorMessage("invalid-base64!", logger)
		assert.Equal(t, "invalid-base64!", result)
	})
}

func TestEvalPklFormattedResponseFile_EdgeCases(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("FileDoesNotExist", func(t *testing.T) {
		tmpDir := t.TempDir()
		responseFile := filepath.Join(tmpDir, "response.pkl")

		resolver := &resolver.DependencyResolver{
			Fs:              fs,
			Logger:          logger,
			Context:         ctx,
			ResponsePklFile: responseFile,
		}

		_, err := resolver.EvalPklFormattedResponseFile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PKL file does not exist")
	})

	t.Run("InvalidFileExtension", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidFile := filepath.Join(tmpDir, "response.txt") // Wrong extension

		resolver := &resolver.DependencyResolver{
			Fs:              fs,
			Logger:          logger,
			Context:         ctx,
			ResponsePklFile: invalidFile,
		}

		// Create the file
		err := afero.WriteFile(fs, invalidFile, []byte("test"), 0o644)
		require.NoError(t, err)

		_, err = resolver.EvalPklFormattedResponseFile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file must have .pkl extension")
	})
}
