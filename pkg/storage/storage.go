package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteFile(target string, data []byte) error {
	dir := filepath.Dir(target)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return fmt.Errorf("error writing file to disk: %v", err)
	}

	return nil
}

func ReadFile(source string) ([]byte, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("error reading file from disk: %v", err)
	}
	return data, nil
}

func EnsureDir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}
	return nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
