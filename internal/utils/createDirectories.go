package utils

import (
	"log"
	"os"
	"path/filepath"
)

func CreateDirectories(baseDir string, fileMap map[string]string) error {
	for relPath, content := range fileMap {
		fullPath := filepath.Join(baseDir, relPath)
		dir := filepath.Dir(fullPath)

		// Create parent directories recursively
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Println(err)
			return err
		}

		// Write file content
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err // Some other error
	}
	return info.IsDir(), nil
}
