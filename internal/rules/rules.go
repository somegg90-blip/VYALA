package rules

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed *.yaml
var rulesFS embed.FS

func ExtractToTemp() (string, error) {
	dir, err := os.MkdirTemp("", "pqc-rules")
	if err != nil {
		return "", err
	}
	entries, err := rulesFS.ReadDir(".")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := rulesFS.ReadFile(entry.Name())
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0644); err != nil {
			return "", err
		}
	}
	return dir, nil
}