package deployments

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

func TestStaticDockerfilePreservesNginxConfiguration(t *testing.T) {
	custom := "server {\n listen 80;\n location ~* \\.(css|js)$ {\n add_header X-Note \"it's $uri, 100% unchanged\";\n }\n}\n"
	proxy := "server {\n listen 80;\n location /api { proxy_pass http://api:3000; }\n}\n"
	for _, config := range []string{"", custom, proxy} {
		dockerfile := (&StaticStrategy{}).generateStaticDockerfile("example/build:revision", "/app/dist", config)
		match := regexp.MustCompile(`RUN printf '%s' '([A-Za-z0-9+/=]+)' \| base64 -d > /etc/nginx/conf.d/default.conf`).FindStringSubmatch(dockerfile)
		if len(match) != 2 {
			t.Fatal("config must be written without shell interpolation or echo escapes")
		}
		decoded, err := base64.StdEncoding.DecodeString(match[1])
		if err != nil {
			t.Fatal(err)
		}
		if config != "" && string(decoded) != config {
			t.Fatalf("custom nginx configuration was changed: %q", decoded)
		}
		if config == "" {
			for _, want := range []string{"server {\n", "try_files $uri $uri/ /index.html;", `location ~* \.(jpg`, `location ~ /\.`} {
				if !strings.Contains(string(decoded), want) {
					t.Fatalf("default config lost nginx syntax %q", want)
				}
			}
		}
		if validatesAtBuild := strings.Contains(dockerfile, "RUN nginx -t\n"); validatesAtBuild != (config == "") {
			t.Fatal("only the standalone default config can be validated without runtime DNS")
		}
	}
}
