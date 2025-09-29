package headless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/substrate-cli/consumer-service-cli/internal/interfaces"
)

type WebsiteData struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Screenshot string `json:"screenshot"`
}

type GPTRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type GPTResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	Content string `json:"content"`
}

func GenerateClonePromptByVision(targetURL string, llmClient interfaces.LLMClient, isRepo bool) (map[string]any, error) {
	// 1. Extract website data
	data, err := extractWebsiteData(targetURL)

	if err != nil {
		log.Println(err)
		log.Println("Error while parsing screenshot")
		return nil, errors.New("unable to clone website")
	}
	// 2. Get GPT Vision analysis
	if data.Screenshot == "" {
		log.Println("Screenshot is nil")
		return nil, errors.New("unable to clone website")
	}
	analysis, err := llmClient.VisionAnalysis(data.Screenshot, data.URL, data.Title, isRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPT analysis: %w", err)
	}
	var resp map[string]any
	err = json.Unmarshal([]byte(analysis), &resp)
	if err != nil {
		log.Println("error unmarshalling json")
		return nil, errors.New("Unable to clone website")
	}
	// 3. Generate final prompt
	// prompt := generateFinalPrompt(data, analysis)
	return resp, nil
}

func extractWebsiteData(targetURL string) (*WebsiteData, error) {
	// Launch browser
	launcher := launcher.New().Headless(true).NoSandbox(true)
	browser := rod.New().ControlURL(launcher.MustLaunch()).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage()
	defer page.MustClose()

	// Set viewport and navigate
	page.MustSetViewport(1920, 1080, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	page = page.Context(ctx)

	if err := page.Navigate(targetURL); err != nil {
		return nil, err
	}

	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	data := &WebsiteData{URL: targetURL}

	// Get title
	data.Title = page.MustInfo().Title

	// Get screenshot
	if screenshot, err := page.Screenshot(true, nil); err == nil {
		data.Screenshot = base64.StdEncoding.EncodeToString(screenshot)
	}

	return data, nil
}

// Get GPT Vision analysis

func fixImageURLWithRod(imageURL string) (string, error) {
	if strings.Contains(strings.ToLower(imageURL), "svg") {
		return convertSVGToPNGWithRod(imageURL)
	}
	return imageURL, nil
}

func convertSVGToPNGWithRod(svgURL string) (string, error) {
	// Launch browser
	l := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(l).MustConnect()
	defer browser.MustClose()

	// Create new page
	page := browser.MustPage()
	defer page.MustClose()

	// Navigate to SVG URL
	err := page.Navigate(svgURL)
	if err != nil {
		return "", fmt.Errorf("failed to navigate to SVG: %v", err)
	}

	// Wait for page to load
	page.MustWaitLoad()
	time.Sleep(1 * time.Second) // Extra wait for SVG rendering

	// Take screenshot
	screenshot, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: intPtr(90),
	})
	if err != nil {
		return "", fmt.Errorf("failed to take screenshot: %v", err)
	}

	// Convert to base64 data URL
	base64Image := base64.StdEncoding.EncodeToString(screenshot)
	return fmt.Sprintf("data:image/png;base64,%s", base64Image), nil
}

func intPtr(i int) *int {
	return &i
}
