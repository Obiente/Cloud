package orchestrator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSanitizeEnvironment_BooleanValues(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  web:
    image: nginx:alpine
    environment:
      ENABLE_SSL: true
      DEBUG_MODE: false
      PORT: 8080
      APP_NAME: "myapp"
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result to verify environment variables are strings
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	web := services["web"].(map[string]interface{})
	env := web["environment"].(map[string]interface{})

	// Check that boolean values are converted to strings
	if enableSSL, ok := env["ENABLE_SSL"].(string); !ok {
		t.Errorf("ENABLE_SSL should be a string, got %T: %v", env["ENABLE_SSL"], env["ENABLE_SSL"])
	} else if enableSSL != "true" {
		t.Errorf("ENABLE_SSL should be 'true', got %q", enableSSL)
	}

	if debugMode, ok := env["DEBUG_MODE"].(string); !ok {
		t.Errorf("DEBUG_MODE should be a string, got %T: %v", env["DEBUG_MODE"], env["DEBUG_MODE"])
	} else if debugMode != "false" {
		t.Errorf("DEBUG_MODE should be 'false', got %q", debugMode)
	}

	// Check that numbers are converted to strings
	if port, ok := env["PORT"].(string); !ok {
		t.Errorf("PORT should be a string, got %T: %v", env["PORT"], env["PORT"])
	} else if port != "8080" {
		t.Errorf("PORT should be '8080', got %q", port)
	}

	// Check that strings remain strings
	if appName, ok := env["APP_NAME"].(string); !ok {
		t.Errorf("APP_NAME should be a string, got %T: %v", env["APP_NAME"], env["APP_NAME"])
	} else if appName != "myapp" {
		t.Errorf("APP_NAME should be 'myapp', got %q", appName)
	}
}

func TestSanitizeEnvironment_DollarSignEscaping(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  api:
    image: myapp/api:latest
    environment:
      DATABASE_URL: "postgresql://user:p@ss$word@localhost/db"
      SECRET_KEY: "a1b2c3$d4e5$f6g7"
      API_TOKEN: "tok$en$here"
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result to verify $ characters are escaped
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	api := services["api"].(map[string]interface{})
	env := api["environment"].(map[string]interface{})

	// Check that $ is escaped to $$
	if dbURL, ok := env["DATABASE_URL"].(string); !ok {
		t.Errorf("DATABASE_URL should be a string, got %T: %v", env["DATABASE_URL"], env["DATABASE_URL"])
	} else if !strings.Contains(dbURL, "$$") {
		t.Errorf("DATABASE_URL should contain escaped $$ characters, got %q", dbURL)
	} else if strings.Count(dbURL, "$$") != 1 {
		t.Errorf("DATABASE_URL should have 1 escaped $ character, got %q", dbURL)
	}

	if secretKey, ok := env["SECRET_KEY"].(string); !ok {
		t.Errorf("SECRET_KEY should be a string, got %T: %v", env["SECRET_KEY"], env["SECRET_KEY"])
	} else if expected := "a1b2c3$$d4e5$$f6g7"; secretKey != expected {
		t.Errorf("SECRET_KEY should be %q, got %q", expected, secretKey)
	}

	if apiToken, ok := env["API_TOKEN"].(string); !ok {
		t.Errorf("API_TOKEN should be a string, got %T: %v", env["API_TOKEN"], env["API_TOKEN"])
	} else if expected := "tok$$en$$here"; apiToken != expected {
		t.Errorf("API_TOKEN should be %q, got %q", expected, apiToken)
	}
}

func TestSanitizeEnvironment_ArrayFormat(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  worker:
    image: worker:latest
    environment:
      - ENABLE_FEATURE=true
      - MAX_WORKERS=10
      - DB_PASS=my$ecret$pass
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Check that the sanitized YAML contains properly escaped values
	if !strings.Contains(sanitizedYaml, "ENABLE_FEATURE=true") {
		t.Errorf("Sanitized YAML should contain 'ENABLE_FEATURE=true'")
	}

	if !strings.Contains(sanitizedYaml, "DB_PASS=my$$ecret$$pass") {
		t.Errorf("Sanitized YAML should contain 'DB_PASS=my$$ecret$$pass', got:\n%s", sanitizedYaml)
	}
}

func TestSanitizeEnvironment_NullValues(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  cache:
    image: redis:alpine
    environment:
      OPTIONAL_CONFIG: null
      EMPTY_VALUE: ""
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	cache := services["cache"].(map[string]interface{})
	env := cache["environment"].(map[string]interface{})

	// Check that null values are converted to empty strings
	if optionalConfig, ok := env["OPTIONAL_CONFIG"].(string); !ok {
		t.Errorf("OPTIONAL_CONFIG should be a string, got %T: %v", env["OPTIONAL_CONFIG"], env["OPTIONAL_CONFIG"])
	} else if optionalConfig != "" {
		t.Errorf("OPTIONAL_CONFIG should be empty string, got %q", optionalConfig)
	}

	// Empty strings should remain empty
	if emptyValue, ok := env["EMPTY_VALUE"].(string); !ok {
		t.Errorf("EMPTY_VALUE should be a string, got %T: %v", env["EMPTY_VALUE"], env["EMPTY_VALUE"])
	} else if emptyValue != "" {
		t.Errorf("EMPTY_VALUE should be empty string, got %q", emptyValue)
	}
}
