package helpers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func IsClonableWebApp(url string) (bool, string) {
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Sprintf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	body := string(bodyBytes)

	// Check if it's HTML
	if strings.Contains(contentType, "text/html") && strings.Contains(strings.ToLower(body), "<html") {
		return true, ""
	}

	// If response looks like JSON, plain text, etc.
	if strings.Contains(contentType, "application/json") || !strings.Contains(body, "<html") {
		return false, "The provided URL looks like a backend API (likely Node.js/Express) and not a frontend web app."
	}

	return false, "The provided URL does not appear to be a clonable frontend application."
}
