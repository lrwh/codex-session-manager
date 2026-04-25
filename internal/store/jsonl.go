package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func WriteJSONLines[T any](path string, items []T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.jsonl")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	encoder := json.NewEncoder(tempFile)
	for _, item := range items {
		if err := encoder.Encode(item); err != nil {
			tempFile.Close()
			return err
		}
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func ReadJSONLines[T any](path string) ([]T, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)

	items := make([]T, 0, 128)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
