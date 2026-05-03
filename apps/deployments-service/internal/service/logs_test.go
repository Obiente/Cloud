package deployments

import (
	"bytes"
	"encoding/binary"
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
	output := "2026-04-05T12:00:02Z second line\n2026-04-05T12:00:00Z first line\nline without timestamp\n\n"
	lines := parseDockerServiceLogOutput("dep-123", output)
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}
	if lines[0].DeploymentId != "dep-123" {
		t.Fatalf("expected deployment id to be preserved")
	}
	if lines[0].Line != "first line" {
		t.Fatalf("expected first parsed line, got %q", lines[0].Line)
	}
	if lines[2].Line != "line without timestamp" {
		t.Fatalf("expected raw fallback line, got %q", lines[1].Line)
	}
}

func TestReadDockerContainerLogLinesParsesFramesAndPartialLines(t *testing.T) {
	var stream bytes.Buffer
	writeDockerLogFrame(&stream, 1, "2026-04-05T12:00:00Z first ")
	writeDockerLogFrame(&stream, 1, "line\n2026-04-05T12:00:01Z second line\n")
	writeDockerLogFrame(&stream, 2, "2026-04-05T12:00:02Z error line\n")

	var lines []*dockerLogLine
	err := readDockerContainerLogLines(&stream, func(line *dockerLogLine) bool {
		lines = append(lines, line)
		return true
	})
	if err != nil {
		t.Fatalf("expected parser to succeed: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].line != "first line" {
		t.Fatalf("expected partial stdout frames to be joined, got %q", lines[0].line)
	}
	if lines[2].line != "error line" || !lines[2].stderr {
		t.Fatalf("expected stderr frame to be preserved, got line=%q stderr=%v", lines[2].line, lines[2].stderr)
	}
}

func writeDockerLogFrame(buf *bytes.Buffer, streamType byte, payload string) {
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	buf.Write(header)
	buf.WriteString(payload)
}
