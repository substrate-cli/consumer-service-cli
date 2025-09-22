package llm

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	option "github.com/openai/openai-go/v2/option"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"

	vision "github.com/sashabaranov/go-openai"
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
			openai.SystemMessage(utils.GetSystemPromptForPrecheck()),
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

func (openAIClient *OpenAIClient) CallGithubTreeScan(prompt string) (string, error) {
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
			openai.SystemMessage(utils.GetSystemPromptForGithubTreeScan()),
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
			openai.SystemMessage(utils.GetSystemPromptForBackendPromptConstruct()),
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

func (openAIClient *OpenAIClient) VisionAnalysis(screenshot string, url string, title string, isRepo bool) (string, error) {
	if screenshot == "" {
		return "No screenshot available", nil
	}

	// Create OpenAI client
	apiKey := openAIClient.APIKey
	client := vision.NewClient(apiKey)

	// Get prompt
	sys := utils.GetVisionAnalysisPromptForUrl(url, title)
	if isRepo {
		sys = utils.GetVisionAnalysisForRepo()
	}

	// Create the request
	req := vision.ChatCompletionRequest{
		Model: vision.GPT4o,
		Messages: []vision.ChatCompletionMessage{
			{
				Role: vision.ChatMessageRoleUser,
				MultiContent: []vision.ChatMessagePart{
					{
						Type: vision.ChatMessagePartTypeText,
						Text: sys,
					},
					{
						Type: vision.ChatMessagePartTypeImageURL,
						ImageURL: &vision.ChatMessageImageURL{
							URL: fmt.Sprintf("data:image/png;base64,%s", screenshot),
						},
					},
				},
			},
		},
		MaxTokens:   16000,
		Temperature: 0.3,
	}

	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Make the request
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	raw := resp.Choices[0].Message.Content
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}

func (openAIClient *OpenAIClient) CallPrePromptForGithubClone(description string) (string, error) {
	log.Println("Inside OpenAI Engine, Assigning Prompt => ", description)
	log.Println("Calling OpenAI precheck, prompt => ", description)
	apiKey := openAIClient.APIKey
	// maxTokens := utils.GetOpenAIMaxTokens()
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	message, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		// MaxTokens: openai.Int(int64(maxTokens)),
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(description),
			openai.SystemMessage(utils.GetSystemPromptForPrePromptGithub()),
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
