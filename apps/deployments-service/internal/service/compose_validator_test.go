package deployments

import "testing"

func TestParseDockerComposeErrorsPreservesComposeDiagnostic(t *testing.T) {
	composeYaml := `services:
  app:
    image: nginx:latest
`

	output := `service "app" refers to undefined volume data: invalid compose project`
	errors := parseDockerComposeErrors(output, composeYaml)

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if got, want := errors[0].Message, output; got != want {
		t.Fatalf("expected message %q, got %q", want, got)
	}
}

func TestParseDockerComposeErrorsStripsOnlyValidatingFilePrefix(t *testing.T) {
	composeYaml := `services:
  app:
    image: nginx:latest
`

	output := `validating /tmp/compose-validate-123/docker-compose.yml: service "app" refers to undefined volume data: invalid compose project`
	errors := parseDockerComposeErrors(output, composeYaml)

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	want := `service "app" refers to undefined volume data: invalid compose project`
	if got := errors[0].Message; got != want {
		t.Fatalf("expected message %q, got %q", want, got)
	}
}
