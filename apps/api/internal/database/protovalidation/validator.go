package protovalidation

import (
	"fmt"
	"reflect"
	"strings"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/database"
)

// ValidationError represents a validation error between GORM model and proto definition
type ValidationError struct {
	ModelType  string
	ProtoType  string
	FieldName  string
	Issue      string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("[%s<->%s] Field %s: %s", e.ModelType, e.ProtoType, e.FieldName, e.Issue)
}

// Validator validates GORM models against proto definitions
type Validator struct {
	Errors []ValidationError
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		Errors: []ValidationError{},
	}
}

// AddError adds an error to the validator
func (v *Validator) AddError(modelType, protoType, field, issue string) {
	v.Errors = append(v.Errors, ValidationError{
		ModelType:  modelType,
		ProtoType:  protoType,
		FieldName:  field,
		Issue:      issue,
	})
}

// IsValid returns true if there are no validation errors
func (v *Validator) IsValid() bool {
	return len(v.Errors) == 0
}

// ErrorsString returns all errors as a string
func (v *Validator) ErrorsString() string {
	if v.IsValid() {
		return "No validation errors"
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d validation errors:\n", len(v.Errors)))
	
	for i, err := range v.Errors {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.String()))
	}
	
	return sb.String()
}

// ValidateDeployment validates the Deployment model against the proto definition
func ValidateDeployment(v *Validator) {
	modelType := reflect.TypeOf(database.Deployment{})
	protoType := reflect.TypeOf(deploymentsv1.Deployment{})
	
	// Basic check to ensure fields exist in both types
	validateModelField(v, modelType, protoType, "ID", "Id")
	validateModelField(v, modelType, protoType, "Name", "Name")
	validateModelField(v, modelType, protoType, "Domain", "Domain")
	validateModelField(v, modelType, protoType, "CustomDomains", "CustomDomains")
	validateModelField(v, modelType, protoType, "Type", "Type")
	validateModelField(v, modelType, protoType, "RepositoryURL", "RepositoryUrl")
	validateModelField(v, modelType, protoType, "Branch", "Branch")
	validateModelField(v, modelType, protoType, "BuildCommand", "BuildCommand")
	validateModelField(v, modelType, protoType, "InstallCommand", "InstallCommand")
	validateModelField(v, modelType, protoType, "Status", "Status")
	validateModelField(v, modelType, protoType, "HealthStatus", "HealthStatus")
	validateModelField(v, modelType, protoType, "LastDeployedAt", "LastDeployedAt")
	validateModelField(v, modelType, protoType, "CreatedAt", "CreatedAt")
	validateModelField(v, modelType, protoType, "BuildTime", "BuildTime")
	validateModelField(v, modelType, protoType, "Size", "Size")
	validateModelField(v, modelType, protoType, "Environment", "Environment")
	validateModelField(v, modelType, protoType, "BandwidthUsage", "BandwidthUsage")
	validateModelField(v, modelType, protoType, "StorageUsage", "StorageUsage")
	
	// Validate JSON tags match proto field names
	validateJsonTags(v, modelType, "deployment")
}

// ValidateOrganization validates the Organization model against the proto definition
func ValidateOrganization(v *Validator) {
	// Example for when you have an Organization model
	// This is a placeholder - implement this when you have the Organization model
	v.AddError("Organization", "organizationsv1.Organization", "*", "Validation not implemented yet")
}

// ValidateAllModels validates all models against their proto definitions
func ValidateAllModels() *Validator {
	v := NewValidator()
	
	// Add validations for each model
	ValidateDeployment(v)
	// ValidateOrganization(v) // Uncomment when implemented
	
	return v
}

// Helper function to validate that a model field matches a proto field
func validateModelField(v *Validator, modelType, protoType reflect.Type, modelFieldName, protoFieldName string) {
	modelField, found := modelType.FieldByName(modelFieldName)
	if !found {
		v.AddError(modelType.Name(), protoType.Name(), modelFieldName, 
			fmt.Sprintf("Model field missing but exists in proto as %s", protoFieldName))
		return
	}
	
	protoField, found := protoType.FieldByName(protoFieldName)
	if !found {
		v.AddError(modelType.Name(), protoType.Name(), modelFieldName,
			fmt.Sprintf("Field exists in model but missing in proto (expected: %s)", protoFieldName))
		return
	}
	
	// Check if field types are compatible
	// This is a simplified check - you may need more sophisticated type checking
	if !areTypesCompatible(modelField.Type, protoField.Type) {
		v.AddError(modelType.Name(), protoType.Name(), modelFieldName,
			fmt.Sprintf("Incompatible types: model=%s, proto=%s", 
				modelField.Type.String(), protoField.Type.String()))
	}
}

// Helper function to check if model and proto field types are compatible
func areTypesCompatible(modelType, protoType reflect.Type) bool {
	// This is a simplified compatibility check
	// You may need to expand it based on your specific needs
	
	// Check for strings
	if modelType.Kind() == reflect.String && protoType.Kind() == reflect.String {
		return true
	}
	
	// Allow JSON string columns that store string arrays to map to repeated string proto fields
	if modelType.Kind() == reflect.String &&
		(protoType.Kind() == reflect.Slice || protoType.Kind() == reflect.Array) &&
		protoType.Elem().Kind() == reflect.String {
		return true
	}
	
	// Check for integers
	if (modelType.Kind() == reflect.Int || modelType.Kind() == reflect.Int32 || modelType.Kind() == reflect.Int64) &&
		(protoType.Kind() == reflect.Int || protoType.Kind() == reflect.Int32 || protoType.Kind() == reflect.Int64) {
		return true
	}
	
	// Check for pointers (optional fields)
	if modelType.Kind() == reflect.Ptr && protoType.Kind() == reflect.Ptr {
		return areTypesCompatible(modelType.Elem(), protoType.Elem())
	}
	
	// Special case for time.Time which maps to google.protobuf.Timestamp
	if modelType.String() == "time.Time" && strings.Contains(protoType.String(), "Timestamp") {
		return true
	}
	
	// Check for slices and arrays
	if (modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) &&
		(protoType.Kind() == reflect.Slice || protoType.Kind() == reflect.Array) {
		return areTypesCompatible(modelType.Elem(), protoType.Elem())
	}
	
	// For JSONB fields, just check if they have a corresponding proto field
	if strings.Contains(modelType.String(), "jsonb") {
		return true
	}
	
	return modelType.Kind() == protoType.Kind()
}

// Helper function to validate JSON tags
func validateJsonTags(v *Validator, modelType reflect.Type, expectedPrefix string) {
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Check JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			v.AddError(modelType.Name(), "proto", field.Name, "Missing json tag")
			continue
		}
		
		// Check if the JSON tag follows snake_case convention
		jsonName := strings.Split(jsonTag, ",")[0]
		if !isSnakeCase(jsonName) {
			v.AddError(modelType.Name(), "proto", field.Name, 
				fmt.Sprintf("JSON tag '%s' should use snake_case", jsonName))
		}
	}
}

// Helper function to check if a string is in snake_case
func isSnakeCase(s string) bool {
	return !strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") && 
		!strings.Contains(s, "--") && 
		!strings.HasPrefix(s, "_") &&
		!strings.HasSuffix(s, "_")
}
