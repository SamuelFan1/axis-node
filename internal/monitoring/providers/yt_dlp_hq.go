package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SamuelFan1/axis-node/internal/monitoring"
)

const ytDlpHQServiceID = "yt-dlp-hq"

type YtDlpHQProvider struct {
	url    string
	client *http.Client
}

type ytDlpHQReadinessPayload struct {
	ServiceID      string `json:"service_id"`
	ServiceHealthy bool   `json:"service_healthy"`
	SessionLoaded  bool   `json:"session_loaded"`
	Ready          bool   `json:"ready"`
	Version        string `json:"version,omitempty"`
}

func NewYtDlpHQProvider(url string, timeout time.Duration) *YtDlpHQProvider {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &YtDlpHQProvider{
		url: strings.TrimSpace(url),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *YtDlpHQProvider) Name() string {
	return ytDlpHQServiceID
}

func (p *YtDlpHQProvider) Kind() string {
	return "service_readiness"
}

func (p *YtDlpHQProvider) Collect(ctx context.Context) (monitoring.SourceSnapshot, error) {
	now := time.Now()
	summary := map[string]interface{}{
		"detected":        false,
		"service_healthy": false,
		"session_loaded":  false,
		"ready":           false,
	}

	if p == nil || p.url == "" {
		return monitoring.SourceSnapshot{
			Name:        ytDlpHQServiceID,
			Kind:        "service_readiness",
			Status:      monitoring.SourceStatusError,
			CollectedAt: now,
			Summary:     summary,
			Error:       "yt-dlp-hq readiness url is empty",
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return monitoring.SourceSnapshot{
			Name:        p.Name(),
			Kind:        p.Kind(),
			Status:      monitoring.SourceStatusError,
			CollectedAt: now,
			Summary:     summary,
			Error:       fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("read response: %w", err)
	}

	readiness, detected := decodeYtDlpHQReadiness(body)
	if !detected {
		return monitoring.SourceSnapshot{
			Name:        p.Name(),
			Kind:        p.Kind(),
			Status:      monitoring.SourceStatusError,
			CollectedAt: now,
			Summary:     summary,
			Payload:     json.RawMessage(body),
			Error:       fmt.Sprintf("unexpected readiness response status=%d", resp.StatusCode),
		}, nil
	}

	summary["detected"] = true
	summary["service_healthy"] = readiness.ServiceHealthy
	summary["session_loaded"] = readiness.SessionLoaded
	summary["ready"] = readiness.Ready
	if strings.TrimSpace(readiness.Version) != "" {
		summary["version"] = readiness.Version
	}

	status := monitoring.SourceStatusOK
	errText := ""
	if resp.StatusCode != http.StatusOK || !readiness.Ready || !readiness.ServiceHealthy || !readiness.SessionLoaded {
		status = monitoring.SourceStatusError
		errText = "yt-dlp-hq is not ready"
	}

	return monitoring.SourceSnapshot{
		Name:        p.Name(),
		Kind:        p.Kind(),
		Status:      status,
		CollectedAt: now,
		Summary:     summary,
		Payload:     json.RawMessage(body),
		Error:       errText,
	}, nil
}

func decodeYtDlpHQReadiness(body []byte) (ytDlpHQReadinessPayload, bool) {
	var direct ytDlpHQReadinessPayload
	if err := json.Unmarshal(body, &direct); err == nil && direct.ServiceID == ytDlpHQServiceID {
		return direct, true
	}

	var wrapped struct {
		Detail ytDlpHQReadinessPayload `json:"detail"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Detail.ServiceID == ytDlpHQServiceID {
		return wrapped.Detail, true
	}

	return ytDlpHQReadinessPayload{}, false
}
