package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/SamuelFan1/axis-node/internal/monitoring"
)

type serviceStatusRunner func(ctx context.Context, serviceName string) (string, error)
type healthStatusRunner func(ctx context.Context, url string) (int, error)

type CloudflaredProvider struct {
	serviceName        string
	monitorServiceName string
	healthURL          string
	statusRunner       serviceStatusRunner
	healthRunner       healthStatusRunner
}

type cloudflaredPayload struct {
	ServiceName          string `json:"service_name"`
	ServiceStatus        string `json:"service_status"`
	MonitorServiceName   string `json:"monitor_service_name,omitempty"`
	MonitorServiceStatus string `json:"monitor_service_status,omitempty"`
	HealthURL            string `json:"health_url,omitempty"`
	HealthStatusCode     int    `json:"health_status_code,omitempty"`
}

func NewCloudflaredProvider(serviceName, monitorServiceName, healthURL string, timeout time.Duration) *CloudflaredProvider {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	return &CloudflaredProvider{
		serviceName:        strings.TrimSpace(serviceName),
		monitorServiceName: strings.TrimSpace(monitorServiceName),
		healthURL:          strings.TrimSpace(healthURL),
		statusRunner:       systemctlStatusRunner,
		healthRunner: func(ctx context.Context, url string) (int, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return 0, err
			}
			resp, err := client.Do(req)
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
			return resp.StatusCode, nil
		},
	}
}

func (p *CloudflaredProvider) Name() string {
	return "cloudflared"
}

func (p *CloudflaredProvider) Kind() string {
	return "service_health"
}

func (p *CloudflaredProvider) Collect(ctx context.Context) (monitoring.SourceSnapshot, error) {
	payload := cloudflaredPayload{
		ServiceName:        p.serviceName,
		MonitorServiceName: p.monitorServiceName,
		HealthURL:          p.healthURL,
	}
	summary := map[string]interface{}{}
	status := monitoring.SourceStatusOK
	var issues []string

	if p.serviceName == "" {
		status = monitoring.SourceStatusError
		issues = append(issues, "cloudflared service name is empty")
	} else {
		serviceStatus, err := p.statusRunner(ctx, p.serviceName)
		payload.ServiceStatus = serviceStatus
		summary["service_status"] = serviceStatus
		if err != nil || serviceStatus != "active" {
			status = monitoring.SourceStatusError
			if err != nil {
				issues = append(issues, "cloudflared service check failed")
			} else {
				issues = append(issues, "cloudflared is not active")
			}
		}
	}

	if p.monitorServiceName != "" {
		monitorStatus, err := p.statusRunner(ctx, p.monitorServiceName)
		payload.MonitorServiceStatus = monitorStatus
		summary["monitor_service_status"] = monitorStatus
		if err != nil || monitorStatus != "active" {
			status = monitoring.SourceStatusError
			if err != nil {
				issues = append(issues, "cloudflared monitor service check failed")
			} else {
				issues = append(issues, "cloudflared monitor service is not active")
			}
		}
	}

	if p.healthURL != "" {
		code, err := p.healthRunner(ctx, p.healthURL)
		payload.HealthStatusCode = code
		summary["health_status_code"] = code
		if err != nil || code != http.StatusOK {
			status = monitoring.SourceStatusError
			if err != nil {
				issues = append(issues, "health url check failed")
			} else {
				issues = append(issues, "health url returned non-200")
			}
		}
	}

	summary["healthy"] = status == monitoring.SourceStatusOK
	rawPayload, _ := json.Marshal(payload)
	return monitoring.SourceSnapshot{
		Name:        p.Name(),
		Kind:        p.Kind(),
		Status:      status,
		CollectedAt: time.Now(),
		Summary:     summary,
		Payload:     rawPayload,
		Error:       strings.Join(issues, "; "),
	}, nil
}

func systemctlStatusRunner(ctx context.Context, serviceName string) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
