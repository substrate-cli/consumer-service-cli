package webhooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/substrate-cli/consumer-service-cli/internal/utils"
)

func CodeGenerationAction(status string, payload map[string]interface{}) error {
	apiServerUrl := utils.GetAPIServerUrl()
	log.Println("inside code generation...")
	log.Println("Proceeding to call code completion webhook...")
	url := fmt.Sprintf("%s/api/webhook/code-generation", apiServerUrl)
	jsonStr, err := json.Marshal(payload)
	if err != nil {
		fmt.Errorf("Unable to parse json, ", err)
		return err
	}
	streamified := string(jsonStr)
	switch status {
	case "finished":
		payload = map[string]interface{}{
			"stream":      streamified,
			"ARCHIVE_KEY": "streamchat",
			"status":      status,
			"type":        "spin",
		}
	case "failed":

	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println(err)
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

func PrecheckAction(status string, message string) error {
	log.Println("inside precheck to call api-server...")
	apiServerUrl := utils.GetAPIServerUrl()
	url := fmt.Sprintf("%s/api/webhook/precheck", apiServerUrl)
	var payload map[string]interface{}
	switch status {
	case "failed":
		payload = map[string]interface{}{
			"ARCHIVE_KEY": "streamchat",
			"type":        "precheck",
			"stream":      message,
			"status":      status,
		}
	case "finished":
		payload = map[string]interface{}{
			"ARCHIVE_KEY": "streamchat",
			"type":        "precheck",
			"stream":      message,
			"status":      status,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Print(err)
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("error calling api-server")
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	return nil
}

func ErrorAction(status string, errorMessage string, message string) error {
	log.Println("inside error action to call api-server...")
	apiServerUrl := utils.GetAPIServerUrl()
	url := fmt.Sprintf("%s/api/webhook/error", apiServerUrl)
	var payload map[string]interface{}
	switch status {
	case "failed":
		payload = map[string]interface{}{
			"ARCHIVE_KEY": "streamchat",
			"type":        "error",
			"stream":      errorMessage,
			"status":      status,
			"error":       message,
		}
	case "finished":
		payload = map[string]interface{}{
			"ARCHIVE_KEY": "streamchat",
			"type":        "error",
			"stream":      errorMessage,
			"status":      status,
			"error":       message,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Print(err)
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("error calling api-server")
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	return nil
}
