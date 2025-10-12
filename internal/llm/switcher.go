package llm

import (
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v2"
	"github.com/substrate-cli/consumer-service-cli/internal/interfaces"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
)

func NewLLMClient(provider string) (interfaces.LLMClient, error) {
	cliApiKey := utils.GetCLIApiKey()
	switch provider {
	case "anthropic":
		apiKey := utils.GetAnthropicKey()
		if cliApiKey != nil {
			apiKey = *cliApiKey
		}
		return &AnthropicClient{APIKey: apiKey, Spec: anthropic.ModelClaudeOpus4_1_20250805}, nil
	case "openai":
		apiKey := utils.GetOpenAIKey()
		if cliApiKey != nil {
			apiKey = *cliApiKey
		}
		return &OpenAIClient{APIKey: apiKey, Spec: openai.ChatModelGPT5}, nil
	case "gemini":
		apiKey := utils.GetGeminiApiKey()
		if cliApiKey != nil {
			apiKey = *cliApiKey
		}
		return &GeminiClient{APIKey: apiKey, Spec: "gemini-2.5-flash"}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
