package helpers

import (
	"context"
	"log"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	option "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sshfz/consumer-service-substrate/internal/utils"
)

func CallPrecheck(prompt string) (string, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	log.Println("Calling anthropic precheck, prompt => ", prompt)
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
			{Text: utils.GetSystemPromptForPrecheck()},
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

func CallConstructBackendPrompt(prompt string) (string, error) {
	log.Println("Inside Anthropic Backend Construct, Assigning Prompt => ", prompt)
	log.Println("Calling anthropic precheck, prompt => ", prompt)
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
			{Text: utils.GetSystemPromptForBackendPromptConstruct()},
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
