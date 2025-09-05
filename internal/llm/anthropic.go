package llm

import (
	"context"
	"encoding/json"
	"fmt"

	// "fmt"
	"log"
	"os"
	"strings"

	"github.com/sshfz/consumer-service-substrate/internal/utils"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	option "github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicClient struct {
	APIKey string
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
			{Text: *utils.GetSystemPromptForPrecheck()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},

		Model: anthropic.ModelClaude4Opus20250514,
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
			{Text: *utils.GetSystemPromptForBackendPromptConstruct()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},

		Model: anthropic.ModelClaude4Opus20250514,
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

func CallAnthropicError(errorMatch []map[string]string) (map[string]interface{}, error) {
	log.Println("Inside error match...")
	//
	var userBlocks []anthropic.ContentBlockParamUnion
	for _, it := range errorMatch {
		path := it["filePath"]
		code := it["actualCode"]
		line := it["error_line_number"] // string; if int, fmt.Sprint(line)
		actualErrorInTheFile := it["error"]

		// Put each file as its own block. Use a fence to keep code intact.
		block := fmt.Sprintf(
			"FILE: %s\nERROR_LINE: %s\n\nERROR_FOUND_FILE: %s\nACTUAL_CODE:\n```\n%s\n```",
			path, line, actualErrorInTheFile, code,
		)
		userBlocks = append(userBlocks, anthropic.NewTextBlock(block))
	}

	//
	cliApiKey := utils.GetCLIApiKey()
	apiKey := utils.GetAnthropicKey()
	if cliApiKey != nil {
		apiKey = *cliApiKey
	}
	maxTokens := utils.GetAnthropicMaxTokens()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: *utils.GetSystemPromptForFix()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(userBlocks...),
		},
		Model: anthropic.ModelClaude4Opus20250514,
	})

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			log.Println(err)
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				// fmt.Printf("%x\n", deltaVariant.Text)
				print(deltaVariant.Text)
			}

		}
	}

	var finalOutput string
	for _, block := range message.Content {
		if block.Type == "text" {
			finalOutput += block.Text
		}
	}

	cleaned := strings.TrimPrefix(finalOutput, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")

	var data map[string]interface{}
	err := json.Unmarshal([]byte(cleaned), &data)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("\n✅ Full Streamed Response:", finalOutput, "mmmmmmmmm")

	if stream.Err() != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return nil, err
	}
	return data, nil
}

func CallAnthropicUpdateRequestPrecheck(newprompt string, existingPrompt string) (string, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", newprompt)
	log.Println("Calling anthropic precheck, prompt => ", newprompt)
	cliApiKey := utils.GetCLIApiKey()
	apiKey := utils.GetAnthropicKey()
	if cliApiKey != nil {
		apiKey = *cliApiKey
	}
	maxTokens := utils.GetAnthropicMaxTokensPrecheck()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: *utils.GetSystemPromptForUpdatePrecheck(existingPrompt)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(newprompt)),
		},

		Model: anthropic.ModelClaude4Opus20250514,
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
