package deployments

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestMergeDeploymentLogLinesCombinesPersistedAndSnapshot(t *testing.T) {
	base := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	persisted := []*deploymentsv1.DeploymentLogLine{
		testLogLine(base.Add(1*time.Second), "one"),
		testLogLine(base.Add(3*time.Second), "three"),
	}
	snapshot := []*deploymentsv1.DeploymentLogLine{
		testLogLine(base.Add(2*time.Second), "two"),
		testLogLine(base.Add(3*time.Second), "three"),
		testLogLine(base.Add(4*time.Second), "four"),
	}

	got := mergeDeploymentLogLines(10, persisted, snapshot)
	if len(got) != 4 {
		t.Fatalf("expected 4 merged lines, got %d", len(got))
	}
	for i, want := range []string{"one", "two", "three", "four"} {
		if got[i].Line != want {
			t.Fatalf("merged line %d = %q, want %q", i, got[i].Line, want)
		}
	}
}

func TestMergeDeploymentLogLinesAppliesTailAfterDedupe(t *testing.T) {
	base := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	got := mergeDeploymentLogLines(2,
		[]*deploymentsv1.DeploymentLogLine{
			testLogLine(base.Add(1*time.Second), "one"),
			testLogLine(base.Add(2*time.Second), "two"),
		},
		[]*deploymentsv1.DeploymentLogLine{
			testLogLine(base.Add(2*time.Second), "two"),
			testLogLine(base.Add(3*time.Second), "three"),
		},
	)
	if len(got) != 2 {
		t.Fatalf("expected 2 tailed lines, got %d", len(got))
	}
	if got[0].Line != "two" || got[1].Line != "three" {
		t.Fatalf("expected tail [two three], got [%s %s]", got[0].Line, got[1].Line)
	}
}

func TestBuildLogStreamingActiveStatusClassifiers(t *testing.T) {
	t.Parallel()

	if !isBuildStreamingActive(int32(deploymentsv1.BuildStatus_BUILD_PENDING)) || !isBuildStreamingActive(int32(deploymentsv1.BuildStatus_BUILD_BUILDING)) {
		t.Fatal("expected pending and building build statuses to stream")
	}
	if isBuildStreamingActive(int32(deploymentsv1.BuildStatus_BUILD_SUCCESS)) || isBuildStreamingActive(int32(deploymentsv1.BuildStatus_BUILD_FAILED)) {
		t.Fatal("expected completed build statuses not to stream by themselves")
	}
	if !isDeploymentBuildLogStreamingActiveStatus(int32(deploymentsv1.DeploymentStatus_BUILDING)) || !isDeploymentBuildLogStreamingActiveStatus(int32(deploymentsv1.DeploymentStatus_DEPLOYING)) {
		t.Fatal("expected building and deploying deployment statuses to keep build logs streaming")
	}
	if isDeploymentBuildLogStreamingActiveStatus(int32(deploymentsv1.DeploymentStatus_RUNNING)) || isDeploymentBuildLogStreamingActiveStatus(int32(deploymentsv1.DeploymentStatus_FAILED)) {
		t.Fatal("expected terminal deployment statuses not to keep build logs streaming")
	}
}

func testLogLine(ts time.Time, line string) *deploymentsv1.DeploymentLogLine {
	return &deploymentsv1.DeploymentLogLine{
		DeploymentId: "dep-123",
		Line:         line,
		Timestamp:    timestamppb.New(ts),
	}
}

func writeDockerLogFrame(buf *bytes.Buffer, streamType byte, payload string) {
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	buf.Write(header)
	buf.WriteString(payload)
}
