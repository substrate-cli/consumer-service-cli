package utils

import (
	"fmt"
	"log"
	"os"
)

func WriteFiles(filePath string, content string) error {
	log.Println("Directory =>", filePath)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
