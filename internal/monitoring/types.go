package monitoring

import (
	"encoding/json"
	"time"
)

const (
	SchemaVersion     = "1"
	SourceStatusOK    = "ok"
	SourceStatusError = "error"
)

type Snapshot struct {
	SchemaVersion string           `json:"schema_version"`
	CollectedAt   time.Time        `json:"collected_at"`
	Sources       []SourceSnapshot `json:"sources"`
}

type SourceSnapshot struct {
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Status      string                 `json:"status"`
	CollectedAt time.Time              `json:"collected_at"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	Payload     json.RawMessage        `json:"payload,omitempty"`
	Error       string                 `json:"error,omitempty"`
}
