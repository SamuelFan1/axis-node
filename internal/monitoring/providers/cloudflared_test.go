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
