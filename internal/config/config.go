package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerURL                  string
	ManagementAddress          string
	Region                     string
	Zone                       string
	Hostname                   string
	Status                     string
	UUIDFile                   string
	SharedToken                string
	ReportIntervalSec          int
	DiskPath                   string
	MonitoringEnabled          bool
	MonitoringGoSidecarEnabled bool
	MonitoringCFTunnelEnabled  bool
	SidecarStatsURL            string
	SidecarStatsTimeoutSec     int
	CFTunnelServiceName        string
	CFTunnelMonitorServiceName string
	CFTunnelHealthURL          string
	CFTunnelTimeoutSec         int
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
		ServerURL:                  getEnv("AXIS_NODE_SERVER_URL", "http://127.0.0.1:9090"),
		ManagementAddress:          getEnv("AXIS_NODE_MANAGEMENT_ADDRESS", ""),
		Region:                     getEnv("AXIS_NODE_REGION", ""),
		Zone:                       strings.ToUpper(strings.TrimSpace(getEnv("AXIS_NODE_ZONE", ""))),
		Hostname:                   hostname,
		Status:                     getEnv("AXIS_NODE_STATUS", "up"),
		UUIDFile:                   getEnv("AXIS_NODE_UUID_FILE", "/data/axis-node/node-uuid"),
		SharedToken:                getEnv("AXIS_NODE_SHARED_TOKEN", ""),
		ReportIntervalSec:          getEnvInt("AXIS_NODE_REPORT_INTERVAL_SEC", 10),
		DiskPath:                   getEnv("AXIS_NODE_DISK_PATH", "/"),
		MonitoringEnabled:          getEnvBool("AXIS_NODE_MONITORING_ENABLED", true),
		MonitoringGoSidecarEnabled: getEnvBool("AXIS_NODE_MONITORING_GO_SIDECAR_ENABLED", true),
		MonitoringCFTunnelEnabled:  getEnvBool("AXIS_NODE_MONITORING_CF_TUNNEL_ENABLED", false),
		SidecarStatsURL:            getEnv("AXIS_NODE_SIDECAR_STATS_URL", "http://127.0.0.1:8086/api/v1/internal/workload-stats"),
		SidecarStatsTimeoutSec:     getEnvInt("AXIS_NODE_SIDECAR_STATS_TIMEOUT_SEC", 3),
		CFTunnelServiceName:        getEnv("AXIS_NODE_MONITORING_CF_TUNNEL_SERVICE_NAME", "cloudflared"),
		CFTunnelMonitorServiceName: getEnv("AXIS_NODE_MONITORING_CF_TUNNEL_MONITOR_SERVICE_NAME", "cloudflared-health-monitor"),
		CFTunnelHealthURL:          getEnv("AXIS_NODE_MONITORING_CF_TUNNEL_HEALTH_URL", "http://localhost:8085/health/"),
		CFTunnelTimeoutSec:         getEnvInt("AXIS_NODE_MONITORING_CF_TUNNEL_TIMEOUT_SEC", 3),
	}

	if cfg.ManagementAddress == "" {
		return nil, fmt.Errorf("AXIS_NODE_MANAGEMENT_ADDRESS is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("AXIS_NODE_REGION is required")
	}
	if cfg.Zone == "" {
		return nil, fmt.Errorf("AXIS_NODE_ZONE is required")
	}
	if len(cfg.Zone) != 2 || !isAlpha2(cfg.Zone) {
		return nil, fmt.Errorf("AXIS_NODE_ZONE must be a 2-letter ISO-3166-1 alpha-2 country code")
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
	if cfg.SidecarStatsTimeoutSec <= 0 {
		cfg.SidecarStatsTimeoutSec = 3
	}
	if cfg.CFTunnelTimeoutSec <= 0 {
		cfg.CFTunnelTimeoutSec = 3
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

func isAlpha2(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
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

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
