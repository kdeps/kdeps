package resolver

import (
	"errors"
	"fmt"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
)

// decodeChatBlock decodes fields in the chat block, handling Base64 decoding where necessary.
func (dr *DependencyResolver) decodeChatBlock(chatBlock *pklLLM.ResourceChat) error {
	// Decode Prompt
	if err := decodeField(&chatBlock.Prompt, "Prompt", utils.SafeDerefString, ""); err != nil {
		return err
	}

	// Decode Role
	if err := decodeField(&chatBlock.Role, "Role", utils.SafeDerefString, RoleHuman); err != nil {
		return err
	}

	// Decode JSONResponseKeys
	if chatBlock.JSONResponseKeys != nil {
		decodedKeys, err := utils.DecodeStringSlice(chatBlock.JSONResponseKeys, "JSONResponseKeys")
		if err != nil {
			return fmt.Errorf("failed to decode JSONResponseKeys: %w", err)
		}
		chatBlock.JSONResponseKeys = decodedKeys
	}

	// Decode Scenario
	if err := decodeScenario(chatBlock, dr.Logger); err != nil {
		return err
	}

	// Decode Files
	if err := decodeFiles(chatBlock); err != nil {
		return err
	}

	// Decode Tools
	if err := decodeTools(chatBlock, dr.Logger); err != nil {
		return err
	}

	return nil
}

// decodeField decodes a single field, handling Base64 if needed, and uses a default value if the field is nil.
func decodeField(field **string, fieldName string, deref func(*string) string, defaultValue string) error {
	if field == nil || *field == nil {
		*field = &defaultValue
	}
	original := deref(*field)
	logger := logging.GetLogger()
	logger.Debug("Decoding field", "fieldName", fieldName, "original", original)
	decoded, err := utils.DecodeBase64IfNeeded(original)
	if err != nil {
		logger.Warn("Base64 decoding failed, using original value", "fieldName", fieldName, "error", err)
		decoded = original
	}
	if decoded == "" && original != "" {
		logger.Warn("Decoded value is empty, preserving original", "fieldName", fieldName, "original", original)
		decoded = original
	}
	*field = &decoded
	logger.Debug("Decoded field", "fieldName", fieldName, "decoded", decoded)
	return nil
}

// decodeScenario decodes the Scenario field, handling nil and empty cases.
func decodeScenario(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	if chatBlock.Scenario == nil {
		logger.Info("Scenario is nil, initializing empty slice")
		emptyScenario := make([]pklLLM.MultiChat, 0)
		chatBlock.Scenario = &emptyScenario
		return nil
	}

	logger.Info("Decoding Scenario", "length", len(*chatBlock.Scenario))
	decodedScenario := make([]pklLLM.MultiChat, 0, len(*chatBlock.Scenario))
	for i, entry := range *chatBlock.Scenario {
		// MultiChat is a struct, not a pointer, so we can always access it
		decodedEntry := pklLLM.MultiChat{}
		if entry.Role != nil {
			decodedRole, err := utils.DecodeBase64IfNeeded(utils.SafeDerefString(entry.Role))
			if err != nil {
				logger.Error("Failed to decode scenario role", "index", i, "error", err)
				return err
			}
			decodedEntry.Role = &decodedRole
		} else {
			logger.Warn("Scenario role is nil", "index", i)
			defaultRole := RoleHuman
			decodedEntry.Role = &defaultRole
		}
		if entry.Prompt != nil {
			decodedPrompt, err := utils.DecodeBase64IfNeeded(utils.SafeDerefString(entry.Prompt))
			if err != nil {
				logger.Error("Failed to decode scenario prompt", "index", i, "error", err)
				return err
			}
			decodedEntry.Prompt = &decodedPrompt
		} else {
			logger.Warn("Scenario prompt is nil", "index", i)
			emptyPrompt := ""
			decodedEntry.Prompt = &emptyPrompt
		}
		logger.Info("Decoded Scenario entry", "index", i, "role", *decodedEntry.Role, "prompt", *decodedEntry.Prompt)
		decodedScenario = append(decodedScenario, decodedEntry)
	}
	chatBlock.Scenario = &decodedScenario
	return nil
}

// decodeFiles decodes the Files field, handling Base64 if needed.
func decodeFiles(chatBlock *pklLLM.ResourceChat) error {
	if chatBlock.Files == nil {
		return nil
	}
	decodedFiles := make([]string, len(*chatBlock.Files))
	for i, file := range *chatBlock.Files {
		decodedFile, err := utils.DecodeBase64IfNeeded(file)
		if err != nil {
			return fmt.Errorf("failed to decode Files[%d]: %w", i, err)
		}
		decodedFiles[i] = decodedFile
	}
	chatBlock.Files = &decodedFiles
	return nil
}

// decodeTools decodes the Tools field, handling nested parameters and nil cases.
func decodeTools(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	if chatBlock == nil {
		logger.Error("chatBlock is nil in decodeTools")
		return errors.New("chatBlock cannot be nil")
	}

	if chatBlock.Tools == nil {
		logger.Info("Tools is nil, initializing empty slice")
		emptyTools := make([]pklLLM.Tool, 0)
		chatBlock.Tools = &emptyTools
		return nil
	}

	logger.Info("Decoding Tools", "length", len(*chatBlock.Tools))
	decodedTools := make([]pklLLM.Tool, 0, len(*chatBlock.Tools))
	var errs []error

	for i, entry := range *chatBlock.Tools {
		// Tool is a struct, not a pointer, so we can always access it
		logger.Debug("Processing tool entry", "index", i, "name", utils.SafeDerefString(entry.Name), "script", utils.SafeDerefString(entry.Script))
		decodedTool, err := decodeToolEntry(&entry, i, logger)
		if err != nil {
			logger.Error("Failed to decode tool entry", "index", i, "error", err)
			errs = append(errs, err)
			continue
		}
		logger.Info("Decoded Tools entry", "index", i, "name", utils.SafeDerefString(decodedTool.Name))
		decodedTools = append(decodedTools, *decodedTool)
	}
	chatBlock.Tools = &decodedTools

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// decodeToolEntry decodes a single Tool entry.
func decodeToolEntry(entry *pklLLM.Tool, index int, logger *logging.Logger) (*pklLLM.Tool, error) {
	if entry == nil {
		logger.Error("Tool entry is nil", "index", index)
		return nil, fmt.Errorf("tool entry at index %d is nil", index)
	}

	decodedTool := &pklLLM.Tool{}
	logger.Debug("Decoding tool", "index", index, "raw_name", entry.Name, "raw_script", entry.Script)

	// Decode Name
	if entry.Name != nil {
		nameStr := utils.SafeDerefString(entry.Name)
		logger.Debug("Checking if name is Base64", "index", index, "name", nameStr, "isBase64", utils.IsBase64Encoded(nameStr))
		if utils.IsBase64Encoded(nameStr) {
			if err := decodeField(&decodedTool.Name, fmt.Sprintf("Tools[%d].Name", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Name = entry.Name
			logger.Debug("Preserving non-Base64 tool name", "index", index, "name", nameStr)
		}
	} else {
		logger.Warn("Tool name is nil", "index", index)
		emptyName := ""
		decodedTool.Name = &emptyName
	}

	// Decode Script
	if entry.Script != nil {
		scriptStr := utils.SafeDerefString(entry.Script)
		logger.Debug("Checking if script is Base64", "index", index, "script_length", len(scriptStr), "isBase64", utils.IsBase64Encoded(scriptStr))
		if utils.IsBase64Encoded(scriptStr) {
			if err := decodeField(&decodedTool.Script, fmt.Sprintf("Tools[%d].Script", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Script = entry.Script
			logger.Debug("Preserving non-Base64 tool script", "index", index, "script_length", len(scriptStr))
		}
	} else {
		logger.Warn("Tool script is nil", "index", index)
		emptyScript := ""
		decodedTool.Script = &emptyScript
	}

	// Decode Description
	if entry.Description != nil {
		descStr := utils.SafeDerefString(entry.Description)
		logger.Debug("Checking if description is Base64", "index", index, "description", descStr, "isBase64", utils.IsBase64Encoded(descStr))
		if utils.IsBase64Encoded(descStr) {
			if err := decodeField(&decodedTool.Description, fmt.Sprintf("Tools[%d].Description", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Description = entry.Description
			logger.Debug("Preserving non-Base64 tool description", "index", index, "description", descStr)
		}
	} else {
		logger.Warn("Tool description is nil", "index", index)
		emptyDesc := ""
		decodedTool.Description = &emptyDesc
	}

	// Decode Parameters
	if entry.Parameters != nil {
		params, err := decodeToolParameters(entry.Parameters, index, logger)
		if err != nil {
			return nil, err
		}
		decodedTool.Parameters = params
		logger.Debug("Decoded tool parameters", "index", index, "param_count", len(*params))
	} else {
		logger.Warn("Tool parameters are nil", "index", index)
		emptyParams := make(map[string]pklLLM.ToolProperties)
		decodedTool.Parameters = &emptyParams
	}

	return decodedTool, nil
}

// decodeToolParameters decodes tool parameters.
func decodeToolParameters(params *map[string]pklLLM.ToolProperties, index int, logger *logging.Logger) (*map[string]pklLLM.ToolProperties, error) {
	decodedParams := make(map[string]pklLLM.ToolProperties, len(*params))
	for paramName, param := range *params {
		// ToolProperties is a struct, not a pointer, so we can always access it
		decodedParam := pklLLM.ToolProperties{Required: param.Required}

		// Decode Type
		if param.Type != nil {
			typeStr := utils.SafeDerefString(param.Type)
			logger.Debug("Checking if parameter type is Base64", "index", index, "paramName", paramName, "type", typeStr, "isBase64", utils.IsBase64Encoded(typeStr))
			if utils.IsBase64Encoded(typeStr) {
				if err := decodeField(&decodedParam.Type, fmt.Sprintf("Tools[%d].Parameters[%s].Type", index, paramName), utils.SafeDerefString, ""); err != nil {
					return nil, err
				}
			} else {
				decodedParam.Type = param.Type
				logger.Debug("Preserving non-Base64 parameter type", "index", index, "paramName", paramName, "type", typeStr)
			}
		} else {
			logger.Warn("Parameter type is nil", "index", index, "paramName", paramName)
			emptyType := ""
			decodedParam.Type = &emptyType
		}

		// Decode Description
		if param.Description != nil {
			descStr := utils.SafeDerefString(param.Description)
			logger.Debug("Checking if parameter description is Base64", "index", index, "paramName", paramName, "description", descStr, "isBase64", utils.IsBase64Encoded(descStr))
			if utils.IsBase64Encoded(descStr) {
				if err := decodeField(&decodedParam.Description, fmt.Sprintf("Tools[%d].Parameters[%s].Description", index, paramName), utils.SafeDerefString, ""); err != nil {
					return nil, err
				}
			} else {
				decodedParam.Description = param.Description
				logger.Debug("Preserving non-Base64 parameter description", "index", index, "paramName", paramName, "description", descStr)
			}
		} else {
			logger.Warn("Parameter description is nil", "index", index, "paramName", paramName)
			emptyDesc := ""
			decodedParam.Description = &emptyDesc
		}

		decodedParams[paramName] = decodedParam
	}
	return &decodedParams, nil
}

// encodeChat encodes a ResourceChat for Pkl storage.
func encodeChat(chat *pklLLM.ResourceChat, logger *logging.Logger) *pklLLM.ResourceChat {
	var encodedScenario *[]pklLLM.MultiChat
	if chat.Scenario != nil && len(*chat.Scenario) > 0 {
		encodedEntries := make([]pklLLM.MultiChat, 0, len(*chat.Scenario))
		for i, entry := range *chat.Scenario {
			// MultiChat is a struct, not a pointer, so we can always access it
			role := utils.SafeDerefString(entry.Role)
			if role == "" {
				role = RoleHuman
				logger.Info("Setting default role for scenario entry", "index", i, "role", role)
			}
			prompt := utils.SafeDerefString(entry.Prompt)
			logger.Info("Encoding scenario entry", "index", i, "role", role, "prompt", prompt)
			encodedRole := utils.EncodeValue(role)
			encodedPrompt := utils.EncodeValue(prompt)
			encodedEntries = append(encodedEntries, pklLLM.MultiChat{
				Role:   &encodedRole,
				Prompt: &encodedPrompt,
			})
		}
		if len(encodedEntries) > 0 {
			encodedScenario = &encodedEntries
		} else {
			logger.Warn("No valid scenario entries after encoding", "original_length", len(*chat.Scenario))
		}
	} else {
		logger.Info("Scenario is nil or empty in encodeChat")
	}

	var encodedTools *[]pklLLM.Tool
	if chat.Tools != nil {
		encodedEntries := encodeTools(chat.Tools)
		encodedTools = &encodedEntries
	}

	var encodedFiles *[]string
	if chat.Files != nil {
		encodedEntries := make([]string, len(*chat.Files))
		for i, file := range *chat.Files {
			encodedEntries[i] = utils.EncodeValue(file)
		}
		encodedFiles = &encodedEntries
	}

	encodedModel := utils.EncodeValue(chat.Model)
	encodedRole := utils.EncodeValue(utils.SafeDerefString(chat.Role))
	encodedPrompt := utils.EncodeValue(utils.SafeDerefString(chat.Prompt))
	encodedResponse := utils.EncodeValuePtr(chat.Response)
	encodedJSONResponseKeys := encodeJSONResponseKeys(chat.JSONResponseKeys)

	timeoutDuration := chat.TimeoutDuration
	if timeoutDuration == nil {
		timeoutDuration = &pkl.Duration{Value: 60, Unit: pkl.Second}
	}

	timestamp := chat.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond}
	}

	return &pklLLM.ResourceChat{
		Model:            encodedModel,
		Prompt:           &encodedPrompt,
		Role:             &encodedRole,
		Scenario:         encodedScenario,
		Tools:            encodedTools,
		JSONResponse:     chat.JSONResponse,
		JSONResponseKeys: encodedJSONResponseKeys,
		Response:         encodedResponse,
		Files:            encodedFiles,
		File:             chat.File,
		Timestamp:        timestamp,
		TimeoutDuration:  timeoutDuration,
	}
}

// encodeJSONResponseKeys encodes JSON response keys.
func encodeJSONResponseKeys(keys *[]string) *[]string {
	if keys == nil {
		return nil
	}
	encoded := make([]string, len(*keys))
	for i, v := range *keys {
		encoded[i] = utils.EncodeValue(v)
	}
	return &encoded
}
