package providers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/SamuelFan1/axis-node/internal/monitoring"
)

func TestCloudflaredProviderCollectHealthy(t *testing.T) {
	provider := &CloudflaredProvider{
		serviceName:        "cloudflared",
		monitorServiceName: "cloudflared-health-monitor",
		healthURL:          "http://localhost:8085/health/",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			return "active", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			return http.StatusOK, nil
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusOK {
		t.Fatalf("expected ok status, got %s", source.Status)
	}
	if healthy, ok := source.Summary["healthy"].(bool); !ok || !healthy {
		t.Fatalf("expected healthy summary to be true, got %#v", source.Summary["healthy"])
	}
	if source.Name != "cloudflared" {
		t.Fatalf("expected source name cloudflared, got %s", source.Name)
	}
}

func TestCloudflaredProviderCollectInactiveService(t *testing.T) {
	provider := &CloudflaredProvider{
		serviceName: "cloudflared",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			return "inactive", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			return http.StatusOK, nil
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %s", source.Status)
	}
	if source.Error == "" {
		t.Fatal("expected error message for inactive service")
	}
}

func TestCloudflaredProviderCollectFailedHealthCheck(t *testing.T) {
	provider := &CloudflaredProvider{
		serviceName: "cloudflared",
		healthURL:   "http://localhost:8085/health/",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			return "active", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			return 0, errors.New("connection refused")
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %s", source.Status)
	}
	if healthy, ok := source.Summary["healthy"].(bool); !ok || healthy {
		t.Fatalf("expected healthy summary to be false, got %#v", source.Summary["healthy"])
	}
}

func TestCloudflaredProviderCollectNon200HealthCheck(t *testing.T) {
	provider := &CloudflaredProvider{
		serviceName: "cloudflared",
		healthURL:   "http://localhost:8085/health/",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			return "active", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			return http.StatusServiceUnavailable, nil
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %s", source.Status)
	}
	if healthy, ok := source.Summary["healthy"].(bool); !ok || healthy {
		t.Fatalf("expected healthy summary to be false, got %#v", source.Summary["healthy"])
	}
}

func TestCloudflaredProviderCollectHTTPModeHealthy(t *testing.T) {
	provider := &CloudflaredProvider{
		mode:      "http",
		readyURL:  "http://127.0.0.1:2000/ready",
		healthURL: "http://localhost:8085/health/",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			t.Fatalf("systemd status runner should not be called in http mode")
			return "", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			switch url {
			case "http://127.0.0.1:2000/ready", "http://localhost:8085/health/":
				return http.StatusOK, nil
			default:
				t.Fatalf("unexpected health url: %s", url)
				return 0, nil
			}
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusOK {
		t.Fatalf("expected ok status, got %s", source.Status)
	}
	if source.Name != "cloudflared" {
		t.Fatalf("expected source name cloudflared, got %s", source.Name)
	}
	if mode, ok := source.Summary["mode"].(string); !ok || mode != "http" {
		t.Fatalf("expected http mode summary, got %#v", source.Summary["mode"])
	}
}

func TestCloudflaredProviderCollectHTTPModeFailedReadyCheck(t *testing.T) {
	provider := &CloudflaredProvider{
		mode:      "http",
		readyURL:  "http://127.0.0.1:2000/ready",
		healthURL: "http://localhost:8085/health/",
		statusRunner: func(ctx context.Context, serviceName string) (string, error) {
			t.Fatalf("systemd status runner should not be called in http mode")
			return "", nil
		},
		healthRunner: func(ctx context.Context, url string) (int, error) {
			if url == "http://127.0.0.1:2000/ready" {
				return http.StatusServiceUnavailable, nil
			}
			return http.StatusOK, nil
		},
	}

	source, err := provider.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %s", source.Status)
	}
	if healthy, ok := source.Summary["healthy"].(bool); !ok || healthy {
		t.Fatalf("expected healthy summary to be false, got %#v", source.Summary["healthy"])
	}
	if source.Error == "" {
		t.Fatal("expected error message for failed ready check")
	}
}
