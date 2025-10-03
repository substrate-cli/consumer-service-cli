package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// "fmt"
	"log"
	"os"
	"strings"

	"github.com/substrate-cli/consumer-service-cli/internal/utils"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	option "github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicClient struct {
	APIKey string
	Spec   anthropic.Model
}

func (anthropicClient *AnthropicClient) CallPrecheck(prompt string) (string, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	log.Println("Calling anthropic precheck, prompt => ", prompt)
	apiKey := anthropicClient.APIKey
	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: utils.GetSystemPromptForPrecheck()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},

		Model: anthropicClient.Spec,
	})
	if err != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return "", err
	}
	log.Println("Output tokens, ", message.Usage.OutputTokens)
	log.Println("Input tokens, ", message.Usage.InputTokens)
	raw := message.Content[0].Text
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}

func (anthropicClient *AnthropicClient) CallGithubTreeScan(prompt string) (string, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	log.Println("Calling anthropic precheck, prompt => ", prompt)
	apiKey := anthropicClient.APIKey
	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: utils.GetSystemPromptForGithubTreeScan()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},

		Model: anthropicClient.Spec,
	})
	if err != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return "", err
	}
	log.Println("Output tokens, ", message.Usage.OutputTokens)
	log.Println("Input tokens, ", message.Usage.InputTokens)
	raw := message.Content[0].Text
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}

func (anthropicClient *AnthropicClient) CallConstructBackendPrompt(prompt string) (string, error) {
	log.Println("Inside Anthropic Backend Construct, Assigning Prompt => ", prompt)
	log.Println("Calling anthropic precheck, prompt => ", prompt)
	apiKey := anthropicClient.APIKey
	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: utils.GetSystemPromptForBackendPromptConstruct()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},

		Model: anthropicClient.Spec,
	})
	if err != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return "", err
	}
	raw := message.Content[0].Text
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}

// // not usign below functions -----
func readAsMap(filename string) (map[string]interface{}, error) {
	var result map[string]interface{}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	return result, err
}

//for vision -------

type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *ImageSource `json:"source,omitempty"`
}

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	ID           string `json:"id"`
	Model        string `json:"model"`
	Role         string `json:"role"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Type         string `json:"type"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (claudeClient *AnthropicClient) VisionAnalysis(screenshot string, url string, title string, isRepo bool) (string, error) {
	if screenshot == "" {
		return "No screenshot available", nil
	}

	// Get prompt using your existing utils function
	sys := utils.GetVisionAnalysisPromptForUrl(url, title)
	if isRepo {
		sys = utils.GetVisionAnalysisForRepo()
	}

	// Create request payload
	request := ClaudeRequest{
		Model:     "claude-sonnet-4-20250514", // Use the latest vision-capable model
		MaxTokens: 16000,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: sys,
					},
					{
						Type: "image",
						Source: &ImageSource{
							Type:      "base64",
							MediaType: "image/png",
							Data:      screenshot, // Already base64 encoded from rod
						},
					},
				},
			},
		},
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", claudeClient.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	raw := claudeResp.Content[0].Text
	// Clean up the response similar to your OpenAI function
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}

func (anthropicClient *AnthropicClient) CallPrePromptForGithubClone(description string) (string, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", description)
	log.Println("Calling anthropic precheck, prompt => ", description)
	apiKey := anthropicClient.APIKey
	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.env("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: utils.GetSystemPromptForPrePromptGithub()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(description)),
		},

		Model: anthropicClient.Spec,
	})
	if err != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return "", err
	}
	log.Println("Output tokens, ", message.Usage.OutputTokens)
	log.Println("Input tokens, ", message.Usage.InputTokens)
	raw := message.Content[0].Text
	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return cleaned, nil
}
