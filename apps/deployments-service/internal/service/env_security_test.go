package deployments

import "testing"

func TestParseEnvFileToMapRejectsUnsafeNamesAndValues(t *testing.T) {
	t.Parallel()

	got, err := parseEnvFileToMap("NODE_ENV=production\n_token='abc;$PATH'\n# comment\n")
	if err != nil {
		t.Fatalf("parseEnvFileToMap returned error for valid env file: %v", err)
	}
	if got["NODE_ENV"] != "production" || got["_token"] != "abc;$PATH" {
		t.Fatalf("parseEnvFileToMap returned unexpected map: %#v", got)
	}

	cases := []struct {
		name    string
		content string
	}{
		{name: "leading digit", content: "1BAD=value"},
		{name: "docker flag injection", content: "--env=value"},
		{name: "space", content: "BAD KEY=value"},
		{name: "shell metacharacter", content: "BAD;rm=value"},
		{name: "path traversal", content: "../../SECRET=value"},
		{name: "null byte value", content: "SAFE=value\x00bad"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseEnvFileToMap(tc.content); err == nil {
				t.Fatalf("parseEnvFileToMap(%q) returned nil error", tc.content)
			}
		})
	}
}
