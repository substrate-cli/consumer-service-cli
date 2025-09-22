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
	defaultModel               string
	openaiMaxTokens            string
	openaiMaxTokensPrecheck    string
	amqpUrl                    string
	safeOrigins                string
	redisAddr                  string
}

var config *configuration
var cliApiKey *string
var currentModel *string

func init() {
	_ = godotenv.Load()

	config = &configuration{
		anthropicKey:               os.Getenv("ANTHROPIC_KEY"),
		anthropicURL:               os.Getenv("ANTHROPIC_URL"),
		anthropicModel:             os.Getenv("ANTHROPIC_MODEL"),
		anthropicMaxTokens:         os.Getenv("ANTHROPIC_MAX_TOKENS"),
		anthropicMaxTokensPrecheck: os.Getenv("ANTHROPIC_MAX_TOKENS_PRECHECK"),
		openaiMaxTokens:            os.Getenv("OPENAI_MAX_TOKENS"),
		openaiMaxTokensPrecheck:    os.Getenv("OPENAI_MAX_TOKENS_PRECHECK"),
		apiServerUrl:               os.Getenv("API_SERVER_URL"),
		openAIKey:                  os.Getenv("OPENAI_KEY"),
		port:                       os.Getenv("PORT"),
		mode:                       os.Getenv("MODE"),
		defaultModel:               os.Getenv("DEFAULT_MODEL"),
		amqpUrl:                    os.Getenv("AMQP_URL"),
		safeOrigins:                os.Getenv("SAFE_ORIGINS"),
		redisAddr:                  os.Getenv("REDIS_ADDR"),
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

func GetSafeOrigins() string {
	return config.safeOrigins
}

func GetRedisAddr() string {
	return config.redisAddr
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

func SetModel(modelName string) {
	currentModel = &modelName
}

func GetModel() *string {
	return currentModel
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

func GetOpenAIMaxTokens() int {
	maxTokens, err := strconv.Atoi(config.anthropicMaxTokens)
	if err != nil {
		log.Println("Setting max tokens to 1024")
		return 1024
	}
	log.Println("Setting max tokens to", maxTokens)
	return maxTokens
}

func GetOpenAIMaxTokensPrecheck() int {
	maxTokens, err := strconv.Atoi(config.anthropicMaxTokensPrecheck)
	if err != nil {
		log.Println("Setting max tokens to 1024")
		return 1024
	}
	log.Println("Setting max tokens to", maxTokens)
	return maxTokens
}

func GetAppPort() string {
	return config.port
}

func GetDefaultModel() string {
	return config.defaultModel
}

func GetAMQPUrl() string {
	return config.amqpUrl
}
