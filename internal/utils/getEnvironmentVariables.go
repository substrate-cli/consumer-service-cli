package utils

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type configuration struct {
	anthropicKey               string
	anthropicURL               string
	anthropicModel             string
	anthropicMaxTokens         string
	anthropicMaxTokensPrecheck string
	apiServerUrl               string
	openAIKey                  string
	port                       string
	mode                       string
}

var config *configuration
var cliApiKey *string

func init() {
	_ = godotenv.Load()

	config = &configuration{
		anthropicKey:               os.Getenv("ANTHROPIC_KEY"),
		anthropicURL:               os.Getenv("ANTHROPIC_URL"),
		anthropicModel:             os.Getenv("ANTHROPIC_MODEL"),
		anthropicMaxTokens:         os.Getenv("ANTHROPIC_MAX_TOKENS"),
		anthropicMaxTokensPrecheck: os.Getenv("ANTHROPIC_MAX_TOKENS_PRECHECK"),
		apiServerUrl:               os.Getenv("API_SERVER_URL"),
		openAIKey:                  os.Getenv("OPENAI_KEY"),
		port:                       os.Getenv("PORT"),
		mode:                       os.Getenv("MODE"),
	}
}

func GetAnthropicKey() string {
	return config.anthropicKey
}

func GetAnthropicURL() string {
	return config.anthropicURL
}

func GetAnthropicModel() string {
	return config.anthropicModel
}

func GetMode() string {
	return config.mode
}

func SetCLIApiKey(key string) {
	mode := GetMode()
	if mode == "cli" {
		cliApiKey = &key
	} else {
		log.Println("unable to update api key")
	}
}

func GetCLIApiKey() *string {
	return cliApiKey
}

func GetAnthropicMaxTokens() int {
	maxTokens, err := strconv.Atoi(config.anthropicMaxTokens)
	if err != nil {
		log.Println("Setting max tokens to 1024")
		return 1024
	}
	log.Println("Setting max tokens to", maxTokens)
	return maxTokens
}

func GetAnthropicMaxTokensPrecheck() int {
	maxTokens, err := strconv.Atoi(config.anthropicMaxTokensPrecheck)
	if err != nil {
		log.Println("Setting max tokens to 1024")
		return 1024
	}
	log.Println("Setting max tokens to", maxTokens)
	return maxTokens
}

func GetAPIServerUrl() string {
	return config.apiServerUrl
}

func GetOpenAIKey() string {
	return config.openAIKey
}

func GetAppPort() string {
	return config.port
}
