package platform

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultDashboardURL = "http://localhost:3000"
	DefaultZitadelURL   = "http://localhost:8080"
	DefaultDomain       = "localhost"
)

func DashboardURL() string {
	for _, key := range []string{"DASHBOARD_URL", "NUXT_PUBLIC_REQUEST_HOST"} {
		if value := normalizeURL(os.Getenv(key)); value != "" {
			return value
		}
	}

	return DefaultDashboardURL
}

func SupportEmail() string {
	return strings.TrimSpace(os.Getenv("SUPPORT_EMAIL"))
}

func ZitadelURL() string {
	for _, key := range []string{"ZITADEL_BASE_URL", "ZITADEL_URL"} {
		if value := normalizeURL(os.Getenv(key)); value != "" {
			return value
		}
	}

	return DefaultZitadelURL
}

func Domain() string {
	domain := strings.TrimSpace(os.Getenv("DOMAIN"))
	if domain == "" {
		return DefaultDomain
	}

	return strings.TrimSuffix(domain, ".")
}

func RegistryURL() string {
	registryURL := strings.TrimSpace(os.Getenv("REGISTRY_URL"))
	if registryURL == "" {
		return fmt.Sprintf("https://registry.%s", Domain())
	}

	domain := Domain()
	registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-obiente.cloud}", domain)
	registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-localhost}", domain)
	registryURL = strings.ReplaceAll(registryURL, "${DOMAIN}", domain)
	return registryURL
}

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
	return strings.TrimSuffix(value, "/")
}
