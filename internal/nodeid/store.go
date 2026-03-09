package nodeid

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const legacyUUIDPath = "./data/node-uuid"

func LoadOrCreate(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		value := strings.TrimSpace(string(data))
		if value != "" {
			return value, nil
		}
	}

	if path != legacyUUIDPath {
		if migrated, err := migrateLegacyFile(path); err != nil {
			return "", err
		} else if migrated {
			if data, err := os.ReadFile(path); err == nil {
				value := strings.TrimSpace(string(data))
				if value != "" {
					return value, nil
				}
			}
		}
	}

	value := uuid.NewString()
	if err := Save(path, value); err != nil {
		return "", err
	}
	return value, nil
}

func migrateLegacyFile(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	data, err := os.ReadFile(legacyUUIDPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return false, nil
	}

	if err := Save(path, value); err != nil {
		return false, err
	}
	if err := os.Remove(legacyUUIDPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

func Save(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0o644)
}
