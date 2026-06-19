package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SamuelFan1/axis-node/internal/monitoring"
)

func TestYtDlpHQProviderCollectReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service_id":"yt-dlp-hq","service_healthy":true,"session_loaded":true,"ready":true,"version":"1.2.0"}`))
	}))
	defer server.Close()

	source, err := NewYtDlpHQProvider(server.URL, time.Second).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusOK {
		t.Fatalf("expected ok status, got %+v", source)
	}
	if source.Summary["detected"] != true || source.Summary["ready"] != true || source.Summary["session_loaded"] != true {
		t.Fatalf("unexpected summary: %+v", source.Summary)
	}
}

func TestYtDlpHQProviderCollectSessionMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":{"service_id":"yt-dlp-hq","service_healthy":true,"session_loaded":false,"ready":false}}`))
	}))
	defer server.Close()

	source, err := NewYtDlpHQProvider(server.URL, time.Second).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %+v", source)
	}
	if source.Summary["detected"] != true || source.Summary["ready"] != false || source.Summary["session_loaded"] != false {
		t.Fatalf("unexpected summary: %+v", source.Summary)
	}
}

func TestYtDlpHQProviderCollectUnreachableIsNotDetected(t *testing.T) {
	source, err := NewYtDlpHQProvider("http://127.0.0.1:1/health/ready", 10*time.Millisecond).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if source.Status != monitoring.SourceStatusError {
		t.Fatalf("expected error status, got %+v", source)
	}
	if source.Summary["detected"] != false {
		t.Fatalf("expected detected=false, got %+v", source.Summary)
	}
}
