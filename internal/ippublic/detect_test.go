package ippublic

import (
	"testing"
)

func TestDetect(t *testing.T) {
	ip := Detect()
	// May be empty in sandbox/offline, but must not panic
	if ip != "" {
		// Basic validation: should not contain spaces
		for _, r := range ip {
			if r == ' ' || r == '\n' {
				t.Errorf("Detect returned invalid IP with whitespace: %q", ip)
				break
			}
		}
	}
}
