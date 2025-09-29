package utils

import (
	"log"
	"os"
	"path/filepath"
)

func GetHomeDirectory() (string, error) {
	if GetBundle() == "dockeuihuigr" {
		return "/apps", nil
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("unable to fetch home directory")
		return "", err
	}
	rootProjectPath := filepath.Join(homedir, "Desktop")
	return rootProjectPath, nil
}
