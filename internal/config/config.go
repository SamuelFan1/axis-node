package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerURL         string
	ManagementAddress string
	Region            string
	Hostname          string
	Status            string
	UUIDFile          string
	SharedToken       string
	ReportIntervalSec int
	DiskPath          string
}

func Load() (*Config, error) {
	loadEnvFile(getEnv("AXIS_NODE_ENV_FILE", ".env"))

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
		SharedToken:       getEnv("AXIS_NODE_SHARED_TOKEN", ""),
		ReportIntervalSec: getEnvInt("AXIS_NODE_REPORT_INTERVAL_SEC", 10),
		DiskPath:          getEnv("AXIS_NODE_DISK_PATH", "/"),
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
	if cfg.SharedToken == "" {
		return nil, fmt.Errorf("AXIS_NODE_SHARED_TOKEN is required")
	}
	if cfg.ReportIntervalSec <= 0 {
		cfg.ReportIntervalSec = 10
	}
	if strings.TrimSpace(cfg.DiskPath) == "" {
		cfg.DiskPath = "/"
	}

	return cfg, nil
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}

func getEnv(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
