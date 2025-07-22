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

	// JSONResponseKeys are already in the correct format

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

	// Note: PKL evaluation errors should be caught at the evaluation level, not here
	// This decoder should only handle successfully evaluated PKL content

	decoded := original
	logger.Debug("Decoded field", "fieldName", fieldName, "decoded", decoded)
	*field = &decoded
	return nil
}

// decodeScenario decodes the Scenario field, handling nil and empty cases.
func decodeScenario(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	if chatBlock.Scenario == nil {
		logger.Info("Scenario is nil, initializing empty slice")
		emptyScenario := make([]*pklLLM.MultiChat, 0)
		chatBlock.Scenario = &emptyScenario
		return nil
	}

	logger.Info("Decoding Scenario", "length", len(*chatBlock.Scenario))
	decodedScenario := make([]*pklLLM.MultiChat, 0, len(*chatBlock.Scenario))
	for i, entry := range *chatBlock.Scenario {
		if entry == nil {
			logger.Warn("Scenario entry is nil", "index", i)
			continue
		}
		decodedEntry := &pklLLM.MultiChat{}
		if entry.Role != nil {
			decodedRole := utils.SafeDerefString(entry.Role)
			decodedEntry.Role = &decodedRole
		} else {
			logger.Warn("Scenario role is nil", "index", i)
			defaultRole := RoleHuman
			decodedEntry.Role = &defaultRole
		}
		if entry.Prompt != nil {
			decodedPrompt := utils.SafeDerefString(entry.Prompt)
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
	// Files are already in the correct format
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
		emptyTools := make([]*pklLLM.Tool, 0)
		chatBlock.Tools = &emptyTools
		return nil
	}

	logger.Info("Decoding Tools", "length", len(*chatBlock.Tools))
	decodedTools := make([]*pklLLM.Tool, 0, len(*chatBlock.Tools))
	var errs []error

	for i, entry := range *chatBlock.Tools {
		if entry == nil {
			logger.Warn("Tools entry is nil", "index", i)
			errs = append(errs, fmt.Errorf("tool entry at index %d is nil", i))
			continue
		}
		logger.Debug("Processing tool entry", "index", i, "name", utils.SafeDerefString(entry.Name), "script", utils.SafeDerefString(entry.Script))
		decodedTool, err := decodeToolEntry(entry, i, logger)
		if err != nil {
			logger.Error("Failed to decode tool entry", "index", i, "error", err)
			errs = append(errs, err)
			continue
		}
		logger.Info("Decoded Tools entry", "index", i, "name", utils.SafeDerefString(decodedTool.Name))
		decodedTools = append(decodedTools, decodedTool)
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
		logger.Debug("Processing tool name", "index", index, "name", nameStr)
		decodedTool.Name = entry.Name
	} else {
		logger.Warn("Tool name is nil", "index", index)
		emptyName := ""
		decodedTool.Name = &emptyName
	}

	// Decode Script
	if entry.Script != nil {
		scriptStr := utils.SafeDerefString(entry.Script)
		logger.Debug("Processing tool script", "index", index, "script_length", len(scriptStr))
		decodedTool.Script = entry.Script
	} else {
		logger.Warn("Tool script is nil", "index", index)
		emptyScript := ""
		decodedTool.Script = &emptyScript
	}

	// Decode Description
	if entry.Description != nil {
		descStr := utils.SafeDerefString(entry.Description)
		logger.Debug("Processing tool description", "index", index, "description", descStr)
		decodedTool.Description = entry.Description
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
		emptyParams := make(map[string]*pklLLM.ToolProperties)
		decodedTool.Parameters = &emptyParams
	}

	return decodedTool, nil
}

// decodeToolParameters decodes tool parameters.
func decodeToolParameters(params *map[string]*pklLLM.ToolProperties, index int, logger *logging.Logger) (*map[string]*pklLLM.ToolProperties, error) {
	decodedParams := make(map[string]*pklLLM.ToolProperties, len(*params))
	for paramName, param := range *params {
		if param == nil {
			logger.Info("Tools parameter is nil", "index", index, "paramName", paramName)
			continue
		}
		decodedParam := &pklLLM.ToolProperties{Required: param.Required}

		// Decode Type
		if param.Type != nil {
			typeStr := utils.SafeDerefString(param.Type)
			logger.Debug("Processing parameter type", "index", index, "paramName", paramName, "type", typeStr)
			decodedParam.Type = param.Type
			logger.Debug("Preserving parameter type", "index", index, "paramName", paramName, "type", typeStr)
		} else {
			logger.Warn("Parameter type is nil", "index", index, "paramName", paramName)
			emptyType := ""
			decodedParam.Type = &emptyType
		}

		// Decode Description
		if param.Description != nil {
			descStr := utils.SafeDerefString(param.Description)
			logger.Debug("Processing parameter description", "index", index, "paramName", paramName, "description", descStr)
			decodedParam.Description = param.Description
			logger.Debug("Preserving non-Base64 parameter description", "index", index, "paramName", paramName, "description", descStr)
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
	var encodedScenario *[]*pklLLM.MultiChat
	if chat.Scenario != nil && len(*chat.Scenario) > 0 {
		encodedEntries := make([]*pklLLM.MultiChat, 0, len(*chat.Scenario))
		for i, entry := range *chat.Scenario {
			if entry == nil {
				logger.Warn("Skipping nil scenario entry in encodeChat", "index", i)
				continue
			}
			role := utils.SafeDerefString(entry.Role)
			if role == "" {
				role = RoleHuman
				logger.Info("Setting default role for scenario entry", "index", i, "role", role)
			}
			prompt := utils.SafeDerefString(entry.Prompt)
			logger.Info("Encoding scenario entry", "index", i, "role", role, "prompt", prompt)
			encodedRole := role
			encodedPrompt := prompt
			encodedEntries = append(encodedEntries, &pklLLM.MultiChat{
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

	var encodedTools *[]*pklLLM.Tool
	if chat.Tools != nil {
		encodedEntries := encodeTools(chat.Tools)
		encodedTools = &encodedEntries
	}

	var encodedFiles *[]string
	if chat.Files != nil {
		encodedFiles = chat.Files
	}

	encodedModel := chat.Model
	encodedRole := chat.Role
	encodedPrompt := chat.Prompt
	encodedResponse := chat.Response
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
		Prompt:           encodedPrompt,
		Role:             encodedRole,
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
		encoded[i] = v
	}
	return &encoded
}

// Exported for testing
func (dr *DependencyResolver) DecodeChatBlock(chatBlock *pklLLM.ResourceChat) error {
	return dr.decodeChatBlock(chatBlock)
}

func DecodeField(field **string, fieldName string, deref func(*string) string, defaultValue string) error {
	return decodeField(field, fieldName, deref, defaultValue)
}

func DecodeScenario(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	return decodeScenario(chatBlock, logger)
}
