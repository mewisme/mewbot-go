package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DataDir is the root directory for all persisted files.
const DataDir = "data"

func dataPath(filename string) (string, error) {
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("filename must not contain '..'")
	}
	return filepath.Join(DataDir, filename), nil
}

// Load reads the contents of data/<filename>. Returns an empty string if the
// file does not exist.
func Load(filename string) (string, error) {
	path, err := dataPath(filename)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read data file: %w", err)
	}
	return string(b), nil
}

// Save writes contents to data/<filename>, creating directories as needed.
func Save(filename, contents string) error {
	path, err := dataPath(filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write data file: %w", err)
	}
	return nil
}
