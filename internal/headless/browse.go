package headless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// "time"

	"github.com/playwright-community/playwright-go"
)

type CloneResult struct {
	HTML        string            `json:"html"`
	CSS         []string          `json:"css"`
	Images      []string          `json:"images"`
	Scripts     []string          `json:"scripts"`
	Fonts       []string          `json:"fonts"`
	Assets      map[string]string `json:"assets"` // URL -> local path mapping
	Title       string            `json:"title"`
	Description string            `json:"description"`
}

type WebsiteCloner struct {
	browser   playwright.Browser
	outputDir string
}

type TreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
}

// GitTreeResponse represents the GitHub API tree response
type GitTreeResponse struct {
	SHA  string     `json:"sha"`
	URL  string     `json:"url"`
	Tree []TreeNode `json:"tree"`
}

func NewWebsiteCloner(outputDir string) (*WebsiteCloner, error) {
	// Install playwright browsers if not already installed
	err := playwright.Install()
	if err != nil {
		return nil, fmt.Errorf("could not install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %v", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-http2",
			"--disable-quic",
			"--no-sandbox",
			"--disable-web-security",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %v", err)
	}

	// Create output directory
	// os.MkdirAll(outputDir, 0755)

	return &WebsiteCloner{
		browser:   browser,
		outputDir: outputDir,
	}, nil
}

func (wc *WebsiteCloner) Close() error {
	return wc.browser.Close()
}

func (wc *WebsiteCloner) CloneSite(targetURL string) (*CloneResult, error) {
	page, err := wc.browser.NewPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	// Quick stealth setup
	page.SetViewportSize(1366, 768)
	page.SetExtraHTTPHeaders(map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})

	// Navigate
	_, err = page.Goto(targetURL, playwright.PageGotoOptions{Timeout: playwright.Float(1000000)})
	if err != nil {
		return nil, err
	}

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{Timeout: playwright.Float(5000)})

	result := &CloneResult{Assets: make(map[string]string)}
	result.Title, _ = page.Title()
	// result.CSS, _ = wc.extractStylesheets(page, targetURL)
	result.Fonts, err = wc.extractFonts(page, targetURL)
	if err != nil {
		log.Println("Unable to extract fonts")
		result.Fonts = []string{}
	}
	result.Images, _ = wc.extractImages(page, targetURL)

	// html, _ := page.Content()
	// result.HTML = wc.processHTML(html, targetURL, result)

	// go wc.downloadAssets(result) // Download in background
	//retruning fonts, title and image assets -----
	return result, nil
}
func (wc *WebsiteCloner) configureStealthMode(page playwright.Page) error {
	// Set a complete, realistic User-Agent
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Set viewport to common resolution
	err := page.SetViewportSize(1366, 768)
	if err != nil {
		return err
	}

	// Set realistic headers
	headers := map[string]string{
		"User-Agent":                userAgent,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br",
		"DNT":                       "1",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Cache-Control":             "max-age=0",
	}

	err = page.SetExtraHTTPHeaders(headers)
	if err != nil {
		return err
	}

	// Hide automation indicators
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			// Remove webdriver property
			Object.defineProperty(navigator, 'webdriver', {
				get: () => false,
			});
			
			// Mock plugins
			Object.defineProperty(navigator, 'plugins', {
				get: () => [1, 2, 3, 4, 5],
			});
			
			// Mock languages
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
			});
			
			// Mock permissions
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);
		`),
	})

	return err
}

func (wc *WebsiteCloner) isBlockedOrCaptcha(title, url string) bool {
	blockedIndicators := []string{
		"access denied",
		"blocked",
		"captcha",
		"robot",
		"automated",
		"security check",
		"unusual traffic",
		"sorry",
	}

	titleLower := strings.ToLower(title)
	urlLower := strings.ToLower(url)

	for _, indicator := range blockedIndicators {
		if strings.Contains(titleLower, indicator) || strings.Contains(urlLower, indicator) {
			return true
		}
	}

	return false
}

func (wc *WebsiteCloner) waitForPageReady(page playwright.Page) error {
	// Wait for basic DOM content
	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(10000),
	})
	if err != nil {
		return err
	}

	// Wait for network to be mostly idle
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(15000),
	})
	if err != nil {
		// Don't fail if network idle times out, just warn
		fmt.Printf("Warning: Network didn't become idle within timeout: %v\n", err)
	}

	// Additional wait for any lazy-loaded content
	time.Sleep(2 * time.Second)

	return nil
}

func (wc *WebsiteCloner) downloadAssetsWithTimeout(result *CloneResult, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- wc.downloadAssets(result)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("asset download timed out after %v", timeout)
	}
}

///////

func (wc *WebsiteCloner) extractStylesheets(page playwright.Page, baseURL string) ([]string, error) {
	// Get all link tags with rel="stylesheet"
	links, err := page.Locator("link[rel='stylesheet']").All()
	if err != nil {
		return nil, err
	}

	var stylesheets []string
	for _, link := range links {
		href, err := link.GetAttribute("href")
		if err != nil || href == "" {
			continue
		}

		absoluteURL := wc.resolveURL(baseURL, href)
		stylesheets = append(stylesheets, absoluteURL)
	}

	// Get inline styles
	styles, err := page.Locator("style").All()
	if err == nil {
		for i, style := range styles {
			content, err := style.TextContent()
			if err != nil {
				continue
			}

			filename := fmt.Sprintf("inline-style-%d.css", i)
			filepath := filepath.Join(wc.outputDir, filename)
			os.WriteFile(filepath, []byte(content), 0644)
			stylesheets = append(stylesheets, filename)
		}
	}

	return stylesheets, nil
}

func (wc *WebsiteCloner) extractImages(page playwright.Page, baseURL string) ([]string, error) {
	// images, err := page.Locator("img").All()

	assets, err := ExtractAllImageURLs(page, baseURL)
	if err != nil {
		return nil, err
	}

	limit := 100
	// Also check for background images in CSS
	if len(assets) > limit {
		assets = assets[:limit]
	}
	return assets, nil
}

func (wc *WebsiteCloner) extractScripts(page playwright.Page, baseURL string) ([]string, error) {
	scripts, err := page.Locator("script[src]").All()
	if err != nil {
		return nil, err
	}

	var scriptURLs []string
	for _, script := range scripts {
		src, err := script.GetAttribute("src")
		if err != nil || src == "" {
			continue
		}

		absoluteURL := wc.resolveURL(baseURL, src)
		scriptURLs = append(scriptURLs, absoluteURL)
	}

	return scriptURLs, nil
}

func (wc *WebsiteCloner) extractFonts(page playwright.Page, baseURL string) ([]string, error) {
	// Get Google Fonts and other font links
	fontLinks, err := page.Locator("link[href*='fonts']").All()
	if err != nil {
		return nil, err
	}

	var fontURLs []string
	for _, link := range fontLinks {
		href, err := link.GetAttribute("href")
		if err != nil || href == "" {
			continue
		}

		absoluteURL := wc.resolveURL(baseURL, href)
		fontURLs = append(fontURLs, absoluteURL)
	}

	return fontURLs, nil
}

func (wc *WebsiteCloner) resolveURL(baseURL, relativeURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	relative, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	return base.ResolveReference(relative).String()
}

func (wc *WebsiteCloner) processHTML(html, baseURL string, result *CloneResult) string {
	// Replace absolute URLs with local paths
	processedHTML := html

	// Replace CSS links
	for _, cssURL := range result.CSS {
		if strings.HasPrefix(cssURL, "http") {
			localPath := wc.getLocalPath(cssURL)
			processedHTML = strings.ReplaceAll(processedHTML, cssURL, localPath)
		}
	}

	// Replace image sources
	for _, imgURL := range result.Images {
		if strings.HasPrefix(imgURL, "http") {
			localPath := wc.getLocalPath(imgURL)
			processedHTML = strings.ReplaceAll(processedHTML, imgURL, localPath)
		}
	}

	// Replace script sources
	for _, scriptURL := range result.Scripts {
		if strings.HasPrefix(scriptURL, "http") {
			localPath := wc.getLocalPath(scriptURL)
			processedHTML = strings.ReplaceAll(processedHTML, scriptURL, localPath)
		}
	}

	// Clean up unnecessary scripts (analytics, tracking, etc.)
	processedHTML = wc.cleanHTML(processedHTML)

	return processedHTML
}

func (wc *WebsiteCloner) cleanHTML(html string) string {
	// Remove Google Analytics
	re := regexp.MustCompile(`<script[^>]*google-analytics[^>]*>.*?</script>`)
	html = re.ReplaceAllString(html, "")

	// Remove Google Tag Manager
	re = regexp.MustCompile(`<script[^>]*gtm\.js[^>]*>.*?</script>`)
	html = re.ReplaceAllString(html, "")

	// Remove Facebook Pixel
	re = regexp.MustCompile(`<script[^>]*facebook[^>]*>.*?</script>`)
	html = re.ReplaceAllString(html, "")

	return html
}

func (wc *WebsiteCloner) getLocalPath(fullURL string) string {
	u, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}

	// Extract filename from URL
	path := u.Path
	if path == "" || path == "/" {
		path = "index.html"
	}

	return filepath.Base(path)
}

func (wc *WebsiteCloner) downloadAssets(result *CloneResult) error {
	allAssets := []string{}
	allAssets = append(allAssets, result.CSS...)
	allAssets = append(allAssets, result.Images...)
	allAssets = append(allAssets, result.Scripts...)
	allAssets = append(allAssets, result.Fonts...)

	for _, assetURL := range allAssets {
		if !strings.HasPrefix(assetURL, "http") {
			continue // Skip already local files
		}

		err := wc.downloadFile(assetURL)
		if err != nil {
			fmt.Printf("Error downloading %s: %v\n", assetURL, err)
		}
	}

	return nil
}

func (wc *WebsiteCloner) downloadFile(fileURL string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	filename := wc.getLocalPath(fileURL)
	filepath := filepath.Join(wc.outputDir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (wc *WebsiteCloner) SaveResults(result *CloneResult) error {
	// Save HTML
	htmlPath := filepath.Join(wc.outputDir, "index.html")
	err := os.WriteFile(htmlPath, []byte(result.HTML), 0644)
	if err != nil {
		return err
	}

	// Save metadata
	metaPath := filepath.Join(wc.outputDir, "metadata.json")
	metaData, _ := json.MarshalIndent(result, "", "  ")
	err = os.WriteFile(metaPath, metaData, 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Website cloned successfully to: %s\n", wc.outputDir)
	return nil
}

func FetchRepoTree(repo string, owner string, defbranch string) (*GitTreeResponse, error) {
	// url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, branch)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, defbranch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("error ocurred while fetching github tree")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tree GitTreeResponse
	err = json.Unmarshal(body, &tree)
	if err != nil {
		return nil, err
	}
	return &tree, nil
}
