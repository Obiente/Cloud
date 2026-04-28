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

func TestValidateNoHostPortMappingsWarns(t *testing.T) {
	composeYaml := `services:
  gowhisper:
    image: example/gowhisper:latest
    ports:
      - "${WHISPER_GO_PORT:-8080}:8080"
`

	warnings := validateNoHostPortMappings(composeYaml)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if got := warnings[0].Severity; got != "warning" {
		t.Fatalf("expected severity warning, got %q", got)
	}
	if got := warnings[0].Message; got == "" || got == `Host port mappings are not supported. Found port mapping '${WHISPER_GO_PORT:-8080}:8080' in service 'gowhisper'. Please use the routing configuration instead of port mappings in your compose file.` {
		t.Fatalf("expected lenient warning message, got %q", got)
	}
}
