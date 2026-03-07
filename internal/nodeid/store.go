package nodeid

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func LoadOrCreate(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		value := strings.TrimSpace(string(data))
		if value != "" {
			return value, nil
		}
	}

	value := uuid.NewString()
	if err := Save(path, value); err != nil {
		return "", err
	}
	return value, nil
}

func Save(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0o644)
}
