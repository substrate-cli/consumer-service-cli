package utils

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Configuration struct {
	anthropicKey               string
	anthropicURL               string
	anthropicModel             string
	anthropicMaxTokens         string
	anthropicMaxTokensPrecheck string
	apiServerUrl               string
	openAIKey                  string
}

var config *Configuration

func init() {
	_ = godotenv.Load()

	config = &Configuration{
		anthropicKey:               os.Getenv("ANTHROPIC_KEY"),
		anthropicURL:               os.Getenv("ANTHROPIC_URL"),
		anthropicModel:             os.Getenv("ANTHROPIC_MODEL"),
		anthropicMaxTokens:         os.Getenv("ANTHROPIC_MAX_TOKENS"),
		anthropicMaxTokensPrecheck: os.Getenv("ANTHROPIC_MAX_TOKENS_PRECHECK"),
		apiServerUrl:               os.Getenv("API_SERVER_URL"),
		openAIKey:                  os.Getenv("OPENAI_KEY"),
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
