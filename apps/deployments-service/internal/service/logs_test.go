package deployments

import (
	"testing"
	"time"
)

func TestParseTimestampedDockerLogLine(t *testing.T) {
	ts, line := parseTimestampedDockerLogLine("2026-04-05T12:34:56.123456789Z service started")
	if line != "service started" {
		t.Fatalf("expected parsed line, got %q", line)
	}
	if ts.Format(time.RFC3339Nano) != "2026-04-05T12:34:56.123456789Z" {
		t.Fatalf("expected parsed timestamp, got %s", ts.Format(time.RFC3339Nano))
	}
}

func TestParseDockerServiceLogOutput(t *testing.T) {
	output := "2026-04-05T12:00:00Z first line\nsecond line without timestamp\n\n"
	lines := parseDockerServiceLogOutput("dep-123", output)
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}
	if lines[0].DeploymentId != "dep-123" {
		t.Fatalf("expected deployment id to be preserved")
	}
	if lines[0].Line != "first line" {
		t.Fatalf("expected first parsed line, got %q", lines[0].Line)
	}
	if lines[1].Line != "second line without timestamp" {
		t.Fatalf("expected raw fallback line, got %q", lines[1].Line)
	}
}
