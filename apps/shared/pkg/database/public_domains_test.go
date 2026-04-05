package database

import "testing"

func TestDefaultMyObienteCloudLabelShortensUUIDBackedIDs(t *testing.T) {
	t.Parallel()

	got := DefaultMyObienteCloudLabel("db-123e4567-e89b-12d3-a456-426614174000")
	want := "db-123e4567e89b12d3"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDefaultMyObienteCloudLabelLeavesLegacyIDsAlone(t *testing.T) {
	t.Parallel()

	got := DefaultMyObienteCloudLabel("db-123")
	if got != "db-123" {
		t.Fatalf("expected legacy label to stay unchanged, got %q", got)
	}
}

func TestDefaultMyObienteCloudDomainUsesShortenedLabel(t *testing.T) {
	t.Parallel()

	got := DefaultMyObienteCloudDomain("db-123e4567-e89b-12d3-a456-426614174000")
	want := "db-123e4567e89b12d3.my.obiente.cloud"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExtractDefaultPublicLabelNormalizesDomain(t *testing.T) {
	t.Parallel()

	got := ExtractDefaultPublicLabel("DB-123E4567E89B12D3.my.obiente.cloud.")
	if got != "db-123e4567e89b12d3" {
		t.Fatalf("expected normalized label, got %q", got)
	}
}
