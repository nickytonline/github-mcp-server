package github

import (
	"encoding/json"
	"testing"

	"github.com/github/github-mcp-http/pkg/translations"
	"github.com/stretchr/testify/require"
)

// TestAllArrayParametersHaveItems verifies that all array-type tool parameters
// have an "items" schema defined, which is required by OpenAI's API
func TestAllArrayParametersHaveItems(t *testing.T) {
	translator, _ := translations.TranslationHelper()

	// Get all tools from toolsets
	toolsetGroup := DefaultToolsetGroup(false, nil, nil, nil, translator, 500)

	// Enable all toolsets to get all tools
	_ = toolsetGroup.EnableToolsets([]string{"all"})

	for _, toolset := range toolsetGroup.Toolsets {
		for _, serverTool := range toolset.GetAvailableTools() {
			toolName := serverTool.Tool.Name
			inputSchema := serverTool.Tool.InputSchema

			// Marshal to JSON to inspect the schema
			schemaJSON, err := json.Marshal(inputSchema)
			require.NoError(t, err, "failed to marshal schema for tool %s", toolName)

			var schema map[string]interface{}
			err = json.Unmarshal(schemaJSON, &schema)
			require.NoError(t, err, "failed to unmarshal schema for tool %s", toolName)

			// Check properties
			properties, ok := schema["properties"].(map[string]interface{})
			if !ok {
				continue // No properties, skip
			}

			for propName, propValue := range properties {
				propDef, ok := propValue.(map[string]interface{})
				if !ok {
					continue
				}

				// If this is an array type, it MUST have an "items" property
				if propType, hasType := propDef["type"].(string); hasType && propType == "array" {
					_, hasItems := propDef["items"]
					require.True(t, hasItems,
						"tool %s has array parameter %s without 'items' schema definition. "+
							"OpenAI requires all array parameters to have an 'items' schema.",
						toolName, propName)
				}
			}
		}
	}
}
