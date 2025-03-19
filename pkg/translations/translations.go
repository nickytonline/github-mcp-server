package translations

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type TranslationHelperFunc func(key string, defaultValue string) string

func NullTranslationHelper(key string, defaultValue string) string {
	return defaultValue
}

func TranslationHelper() (TranslationHelperFunc, func()) {
	var translationKeyMap = map[string]string{}
	v := viper.New()

	v.SetEnvPrefix("GITHUB_MCP_")
	v.AutomaticEnv()

	// Load from JSON file
	v.SetConfigName("github-mcp-server")
	v.SetConfigType("json")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		// ignore error if file not found as it is not required
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Could not read JSON config: %v", err)
		}
	}

	// create a function that takes both a key, and a default value and returns either the default value or an override value
	return func(key string, defaultValue string) string {
			key = strings.ToUpper(key)
			if value, exists := translationKeyMap[key]; exists {
				return value
			}
			// check if the env var exists
			if value, exists := os.LookupEnv("GITHUB_MCP_" + key); exists {
				// TODO I could not get Viper to play ball reading the env var
				translationKeyMap[key] = value
				return value
			}

			v.SetDefault(key, defaultValue)
			translationKeyMap[key] = v.GetString(key)
			return translationKeyMap[key]
		}, func() {
			// dump the translationKeyMap to a json file
			DumpTranslationKeyMap(translationKeyMap)
		}
}

// dump translationKeyMap to a json file called github-mcp-server.json
func DumpTranslationKeyMap(translationKeyMap map[string]string) {
	file, err := os.Create("github-mcp-server.json")
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	// marshal the map to json
	jsonData, err := json.MarshalIndent(translationKeyMap, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling map to JSON: %v", err)
	}

	// write the json data to the file
	if _, err := file.Write(jsonData); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
}
