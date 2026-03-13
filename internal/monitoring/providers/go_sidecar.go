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

type GoSidecarProvider struct {
	url    string
	client *http.Client
}

func NewGoSidecarProvider(url string, timeout time.Duration) *GoSidecarProvider {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &GoSidecarProvider{
		url: strings.TrimSpace(url),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *GoSidecarProvider) Name() string {
	return "go-sidecar"
}

func (p *GoSidecarProvider) Kind() string {
	return "service_workload"
}

func (p *GoSidecarProvider) Collect(ctx context.Context) (monitoring.SourceSnapshot, error) {
	if p == nil || p.url == "" {
		return monitoring.SourceSnapshot{}, fmt.Errorf("go-sidecar stats url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return monitoring.SourceSnapshot{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return monitoring.SourceSnapshot{}, fmt.Errorf("decode response: %w", err)
	}

	var summary map[string]interface{}
	if rawSummary, ok := decoded["summary"].(map[string]interface{}); ok {
		summary = rawSummary
	}

	return monitoring.SourceSnapshot{
		Name:        p.Name(),
		Kind:        p.Kind(),
		Status:      monitoring.SourceStatusOK,
		CollectedAt: time.Now(),
		Summary:     summary,
		Payload:     json.RawMessage(body),
	}, nil
}
