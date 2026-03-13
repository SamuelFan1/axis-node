package monitoring

import (
	"context"
	"time"
)

type Provider interface {
	Name() string
	Kind() string
	Collect(ctx context.Context) (SourceSnapshot, error)
}

type Collector struct {
	providers []Provider
}

func NewCollector(providers ...Provider) *Collector {
	return &Collector{providers: providers}
}

func (c *Collector) Collect(ctx context.Context) Snapshot {
	snapshot := Snapshot{
		SchemaVersion: SchemaVersion,
		CollectedAt:   time.Now(),
		Sources:       make([]SourceSnapshot, 0, len(c.providers)),
	}

	for _, provider := range c.providers {
		source, err := provider.Collect(ctx)
		if err != nil {
			snapshot.Sources = append(snapshot.Sources, SourceSnapshot{
				Name:        provider.Name(),
				Kind:        provider.Kind(),
				Status:      SourceStatusError,
				CollectedAt: time.Now(),
				Error:       err.Error(),
			})
			continue
		}
		if source.Name == "" {
			source.Name = provider.Name()
		}
		if source.Kind == "" {
			source.Kind = provider.Kind()
		}
		if source.Status == "" {
			source.Status = SourceStatusOK
		}
		if source.CollectedAt.IsZero() {
			source.CollectedAt = time.Now()
		}
		snapshot.Sources = append(snapshot.Sources, source)
	}

	return snapshot
}
