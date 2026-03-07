package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerURL         string
	ManagementAddress string
	Region            string
	Hostname          string
	Status            string
	UUIDFile          string
}

func Load() (*Config, error) {
	hostname := strings.TrimSpace(os.Getenv("AXIS_NODE_HOSTNAME"))
	if hostname == "" {
		if value, err := os.Hostname(); err == nil {
			hostname = strings.TrimSpace(value)
		}
	}

	cfg := &Config{
		ServerURL:         getEnv("AXIS_NODE_SERVER_URL", "http://127.0.0.1:9090"),
		ManagementAddress: getEnv("AXIS_NODE_MANAGEMENT_ADDRESS", ""),
		Region:            getEnv("AXIS_NODE_REGION", ""),
		Hostname:          hostname,
		Status:            getEnv("AXIS_NODE_STATUS", "up"),
		UUIDFile:          getEnv("AXIS_NODE_UUID_FILE", "./data/node-uuid"),
	}

	if cfg.ManagementAddress == "" {
		return nil, fmt.Errorf("AXIS_NODE_MANAGEMENT_ADDRESS is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("AXIS_NODE_REGION is required")
	}
	if cfg.Hostname == "" {
		return nil, fmt.Errorf("unable to determine hostname")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return defaultValue
}
