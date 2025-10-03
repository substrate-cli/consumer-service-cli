package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/substrate-cli/consumer-service-cli/internal/utils"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	APIKey string
	Spec   string
}

func (geminiClient *GeminiClient) CallPrecheck(prompt string) (string, error) {
	log.Println("Inside Gemini Engine, Assigning Prompt => ", prompt)
	log.Println("Calling gemini precheck, prompt => ", prompt)

	ctx := context.Background()

	// Create client
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiClient.APIKey))
	if err != nil {
		log.Println("error creating gemini client")
		log.Println(err)
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiClient.Spec)

	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	maxTokensInt32 := int32(maxTokens)
	model.MaxOutputTokens = &maxTokensInt32

	systemPrompt := utils.GetSystemPromptForPrecheck()
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("error calling gemini api")
		log.Println(err)
		return "", err
	}

	if resp.UsageMetadata != nil {
		log.Println("Output tokens: ", resp.UsageMetadata.CandidatesTokenCount)
		log.Println("Input tokens: ", resp.UsageMetadata.PromptTokenCount)
		log.Println("Total tokens: ", resp.UsageMetadata.TotalTokenCount)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}

func (geminiClient *GeminiClient) CallConstructBackendPrompt(prompt string) (string, error) {
	log.Println("Inside Gemini Backend Construct, Assigning Prompt => ", prompt)
	log.Println("Calling Gemini backend construct, prompt => ", prompt)

	ctx := context.Background()

	// Create client
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiClient.APIKey))
	if err != nil {
		log.Println("error creating gemini client")
		log.Println(err)
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiClient.Spec)

	systemPrompt := utils.GetSystemPromptForBackendPromptConstruct()
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	maxTokens := utils.GetOpenAIMaxTokensPrecheck()
	maxTokensInt32 := int32(maxTokens)
	model.MaxOutputTokens = &maxTokensInt32

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("error calling gemini api")
		log.Println(err)
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}

func (geminiClient *GeminiClient) CallGithubTreeScan(prompt string) (string, error) {
	log.Println("Inside Gemini Engine, Assigning Prompt => ", prompt)
	log.Println("Calling Gemini github tree scan, prompt => ", prompt)

	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiClient.APIKey))
	if err != nil {
		log.Println("error creating gemini client")
		log.Println(err)
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiClient.Spec)

	systemPrompt := utils.GetSystemPromptForGithubTreeScan()
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	maxTokens := utils.GetOpenAIMaxTokensPrecheck()
	maxTokensInt32 := int32(maxTokens)
	model.MaxOutputTokens = &maxTokensInt32

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("error calling gemini api")
		log.Println(err)
		return "", err
	}

	if resp.UsageMetadata != nil {
		log.Println("Output tokens: ", resp.UsageMetadata.CandidatesTokenCount)
		log.Println("Input tokens: ", resp.UsageMetadata.PromptTokenCount)
		log.Println("Total tokens: ", resp.UsageMetadata.TotalTokenCount)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}

func (geminiClient *GeminiClient) VisionAnalysis(screenshot string, url string, title string, isRepo bool) (string, error) {
	if screenshot == "" {
		return "No screenshot available", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiClient.APIKey))
	if err != nil {
		log.Println("error creating gemini client")
		return "", fmt.Errorf("failed to create gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(geminiClient.Spec)

	maxTokens := int32(16000)
	model.MaxOutputTokens = &maxTokens

	temperature := float32(0.3)
	model.Temperature = &temperature
	var systemPrompt string
	if isRepo {
		systemPrompt = utils.GetVisionAnalysisForRepo()
	} else {
		systemPrompt = utils.GetVisionAnalysisPromptForUrl(url, title)
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(screenshot)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Create multimodal prompt with text and image
	resp, err := model.GenerateContent(ctx,
		genai.ImageData("png", imageData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if resp.UsageMetadata != nil {
		log.Println("Output tokens: ", resp.UsageMetadata.CandidatesTokenCount)
		log.Println("Input tokens: ", resp.UsageMetadata.PromptTokenCount)
		log.Println("Total tokens: ", resp.UsageMetadata.TotalTokenCount)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	log.Println("vision => ", cleaned)
	return cleaned, nil
}

func (geminiClient *GeminiClient) CallPrePromptForGithubClone(description string) (string, error) {
	log.Println("Inside Gemini Engine, Assigning Prompt => ", description)
	log.Println("Calling Gemini pre-prompt for github, prompt => ", description)

	ctx := context.Background()

	// Create client
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiClient.APIKey))
	if err != nil {
		log.Println("error creating gemini client")
		log.Println(err)
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(geminiClient.Spec)
	systemPrompt := utils.GetSystemPromptForPrePromptGithub()
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	maxTokens := utils.GetOpenAIMaxTokensPrecheck()
	maxTokensInt32 := int32(maxTokens)
	model.MaxOutputTokens = &maxTokensInt32

	resp, err := model.GenerateContent(ctx, genai.Text(description))
	if err != nil {
		log.Println("error calling gemini api")
		log.Println(err)
		return "", err
	}

	if resp.UsageMetadata != nil {
		log.Println("Output tokens: ", resp.UsageMetadata.CandidatesTokenCount)
		log.Println("Input tokens: ", resp.UsageMetadata.PromptTokenCount)
		log.Println("Total tokens: ", resp.UsageMetadata.TotalTokenCount)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	cleaned := strings.TrimPrefix(raw, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	return cleaned, nil
}
