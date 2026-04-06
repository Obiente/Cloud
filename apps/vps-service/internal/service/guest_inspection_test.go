package vps

import (
	"testing"
	"time"
)

func TestNormalizeSystemdUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "appends service suffix", input: "nginx", want: "nginx.service"},
		{name: "keeps explicit service", input: "ssh.service", want: "ssh.service"},
		{name: "allows template unit", input: "docker@123.service", want: "docker@123.service"},
		{name: "rejects injection", input: "ssh.service; rm -rf /", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeSystemdUnit(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseJournalctlOutput(t *testing.T) {
	t.Parallel()

	output := []byte(`
{"MESSAGE":"nginx started","PRIORITY":"6","__REALTIME_TIMESTAMP":"1712396100000000"}
{"MESSAGE":"unit failed","PRIORITY":"3","__REALTIME_TIMESTAMP":"1712396200000000"}
`)

	logs, err := parseJournalctlOutput(output)
	if err != nil {
		t.Fatalf("parseJournalctlOutput() error = %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].GetLine() != "nginx started" || logs[0].GetStderr() {
		t.Fatalf("unexpected first log: %+v", logs[0])
	}
	if logs[1].GetLine() != "unit failed" || !logs[1].GetStderr() {
		t.Fatalf("unexpected second log: %+v", logs[1])
	}
	if logs[1].GetLineNumber() != 2 {
		t.Fatalf("expected second line number 2, got %d", logs[1].GetLineNumber())
	}
}

func TestParseSystemctlServicesOutput(t *testing.T) {
	t.Parallel()

	output := []byte(`[
{"unit":"ssh.service","load":"loaded","active":"active","sub":"running","description":"OpenBSD Secure Shell server"},
{"unit":"postgresql.service","load":"loaded","active":"failed","sub":"failed","description":"PostgreSQL RDBMS"}
]`)

	services, err := parseSystemctlServicesOutput(output)
	if err != nil {
		t.Fatalf("parseSystemctlServicesOutput() error = %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0].GetName() != "ssh.service" || services[0].GetActiveState() != "active" {
		t.Fatalf("unexpected first service: %+v", services[0])
	}
	if services[1].GetSubState() != "failed" {
		t.Fatalf("unexpected second service: %+v", services[1])
	}
}

func TestParseJournalTimestamp(t *testing.T) {
	t.Parallel()

	ts := parseJournalTimestamp("1712396100000000")
	if ts.UTC().Format(time.RFC3339) != "2024-04-06T09:35:00Z" {
		t.Fatalf("unexpected timestamp: %s", ts.UTC().Format(time.RFC3339))
	}
}
