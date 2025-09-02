package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	// "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sshfz/consumer-service-substrate/internal/utils"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	option "github.com/anthropics/anthropic-sdk-go/option"
)

func CallAnthropicTemp(prompt string) (map[string]interface{}, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	res, err := readAsMap("/Users/shreyash/Desktop/projects/substrate/system/consumer-service/internal/helpers/tempresponse.json")
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return res, nil
}

func CallAnthropicApi(prompt string) (*string, error) {

	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type AnthropicRequest struct {
		Model     string    `json:"model"`
		Messages  []Message `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
		system    string
	}
	// type AnthropicResponse struct {
	// 	Content string `json:"content"`
	// }
	cliApiKey := utils.GetCLIApiKey()
	apiKey := utils.GetAnthropicKey()
	if cliApiKey != nil {
		apiKey = *cliApiKey
	}
	model := utils.GetAnthropicModel()
	maxTokens := utils.GetAnthropicMaxTokens()
	systemPrompt := utils.GetSystemPromptForFullStackCode(3000)

	reqBody := AnthropicRequest{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		system:    *systemPrompt,
		MaxTokens: maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	url := utils.GetAnthropicURL()

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	type ContentItem struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type AnthropicResponse struct {
		Content []ContentItem `json:"content"`
	}

	var res AnthropicResponse
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}
	// log.Print("Actual response => ", res.Content)
	log.Println("Response:", res.Content[0].Text)
	str := res.Content[0].Text
	cleaned := strings.TrimPrefix(str, "```json\n")
	cleaned = strings.TrimSuffix(cleaned, "\n```")
	return &cleaned, nil
}

func CallAnthropicPrecheck(prompt string) (string, error) {
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

func CallAnthropicStreamForFullStack(prompt string, serverPort int) (map[string]interface{}, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	cliApiKey := utils.GetCLIApiKey()
	apiKey := utils.GetAnthropicKey()
	if cliApiKey != nil {
		apiKey = *cliApiKey
	}
	maxTokens := utils.GetAnthropicMaxTokens()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: *utils.GetSystemPromptForFullStackCode(serverPort)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			log.Println("error calling anthropic api")
			log.Println(err)
			return nil, err
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
	// log.Println("\n✅ Full Streamed Response:", finalOutput, "mmmmmmmmm")

	if stream.Err() != nil {
		log.Println("error calling anthropic api")
		log.Println(err)
		return nil, err
	}
	return data, nil
}

func CallAnthropicStream(prompt string) (map[string]interface{}, error) {
	log.Println("Inside Anthropic Engine, Assigning Prompt => ", prompt)
	cliApiKey := utils.GetCLIApiKey()
	apiKey := utils.GetAnthropicKey()
	if cliApiKey != nil {
		apiKey = *cliApiKey
	}
	maxTokens := utils.GetAnthropicMaxTokens()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	stream := client.Messages.NewStreaming(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: *utils.GetSystemPromptForCode()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			log.Println("error calling anthropic api")
			log.Println(err)
			return nil, err
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
	// content, err := os.ReadFile("/Users/shreyash/Desktop/projects/substrate/system/consumer-service/cmd/app/test.txt")
	// if err != nil {

	// }
	// finalOutput = string(content)
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

func readAsMap(filename string) (map[string]interface{}, error) {
	var result map[string]interface{}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	return result, err
}

func CallAnthropicConstructBackendPrompt(prompt string) (string, error) {
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
