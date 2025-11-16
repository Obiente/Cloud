package protovalidation

import (
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func TestValidateDeployment(t *testing.T) {
	validator := NewValidator()
	ValidateDeployment(validator)
	
	if !validator.IsValid() {
		t.Errorf("Deployment model validation failed:\n%s", validator.ErrorsString())
	}
}

func TestValidateAllModels(t *testing.T) {
	validator := ValidateAllModels()
	
	if !validator.IsValid() {
		t.Errorf("Model validation failed:\n%s", validator.ErrorsString())
	}
}

// This test ensures that the actual GORM model struct matches what we expect
// If the model structure changes without updating the validation, this test will fail
func TestDeploymentModelStructure(t *testing.T) {
	deployment := database.Deployment{}
	
	// Test required fields exist with the correct types
	// These checks will fail compilation if fields are missing or of wrong type
	checkStringField := func(s string) {}
	checkStringPtrField := func(s *string) {}
	checkInt32Field := func(i int32) {}
	checkInt64Field := func(i int64) {}
	
	checkStringField(deployment.ID)
	checkStringField(deployment.Name)
	checkStringField(deployment.Domain)
	checkStringField(deployment.CustomDomains) // JSON string array in DB
	checkInt32Field(deployment.Type)
	checkStringPtrField(deployment.RepositoryURL)
	checkStringField(deployment.Branch)
	checkStringPtrField(deployment.BuildCommand)
	checkStringPtrField(deployment.InstallCommand)
	checkInt32Field(deployment.Status)
	checkStringField(deployment.HealthStatus)
	checkInt32Field(deployment.BuildTime)
	checkStringField(deployment.Size)
	checkInt32Field(deployment.Environment)
	checkInt64Field(deployment.BandwidthUsage)
	checkInt64Field(deployment.StorageBytes)
	
	// Check that time fields exist (will fail compilation if missing)
	_ = deployment.LastDeployedAt
	_ = deployment.CreatedAt
	
	// Check foreign keys
	checkStringField(deployment.OrganizationID)
	checkStringField(deployment.CreatedBy)
}
