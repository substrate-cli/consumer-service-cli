package llm

import (
	"context"
	"log"
	"strings"

	"github.com/openai/openai-go/v2"
	option "github.com/openai/openai-go/v2/option"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
)

type OpenAIClient struct {
	APIKey string
}

func (openAIClient *OpenAIClient) CallPrecheck(prompt string) (string, error) {
	log.Println("Inside OpenAI Engine, Assigning Prompt => ", prompt)
	log.Println("Calling OpenAI precheck, prompt => ", prompt)
	apiKey := openAIClient.APIKey
	// maxTokens := utils.GetOpenAIMaxTokens()
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	message, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		// MaxTokens: openai.Int(int64(maxTokens)),
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
			openai.SystemMessage(*utils.GetSystemPromptForPrecheck()),
		},
	})
	if err != nil {
		log.Println("error calling openai api")
		log.Println(err)
		return "", err
	}
	log.Println("Output tokens, ", message.Usage.CompletionTokens)
	log.Println("Input tokens, ", message.Usage.PromptTokens)
	raw := message.Choices[0].Message.Content
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}

func (openAIClient *OpenAIClient) CallConstructBackendPrompt(prompt string) (string, error) {
	log.Println("Inside OpenAI Backend Construct, Assigning Prompt => ", prompt)
	log.Println("Calling OpenAI precheck, prompt => ", prompt)
	apiKey := openAIClient.APIKey
	// maxTokens := utils.GetOpenAIMaxTokensPrecheck()
	client := openai.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		// MaxTokens: openai.Int(int64(maxTokens)),
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
			openai.SystemMessage(*utils.GetSystemPromptForBackendPromptConstruct()),
		},
	})
	if err != nil {
		log.Println("error calling openai api")
		log.Println(err)
		return "", err
	}
	raw := message.Choices[0].Message.Content
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}
