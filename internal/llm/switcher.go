package llm

import (
	"fmt"

	"github.com/sshfz/consumer-service-substrate/internal/interfaces"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
)

func NewLLMClient(provider string) (interfaces.LLMClient, error) {
	cliApiKey := utils.GetCLIApiKey()
	switch provider {
	case "anthropic":
		apiKey := utils.GetAnthropicKey()
		if cliApiKey != nil {
			apiKey = *cliApiKey
		}
		return &AnthropicClient{APIKey: apiKey}, nil
	case "openai":
		apiKey := utils.GetOpenAIKey()
		if cliApiKey != nil {
			apiKey = *cliApiKey
		}
		return &OpenAIClient{APIKey: apiKey}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
