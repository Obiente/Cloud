package protosync

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
)

// ProtoField represents a field in a protobuf message that corresponds to a GORM model field
type ProtoField struct {
	Name      string
	ProtoName string
	GoType    string
	Required  bool
}

// ModelValidator validates GORM models against proto definitions
type ModelValidator struct {
	Errors []string
}

// NewModelValidator creates a new validator
func NewModelValidator() *ModelValidator {
	return &ModelValidator{
		Errors: []string{},
	}
}

// AddError adds an error to the validator
func (v *ModelValidator) AddError(format string, args ...interface{}) {
	v.Errors = append(v.Errors, fmt.Sprintf(format, args...))
}

// IsValid returns true if there are no validation errors
func (v *ModelValidator) IsValid() bool {
	return len(v.Errors) == 0
}

// ErrorsString returns all errors as a string
func (v *ModelValidator) ErrorsString() string {
	return strings.Join(v.Errors, "\n")
}

// ValidateDeployment validates the Deployment model against the proto definition
func ValidateDeployment(modelType reflect.Type) *ModelValidator {
	validator := NewModelValidator()
	protoFields := getDeploymentProtoFields()
	
	// Check each proto field exists in the model
	for _, protoField := range protoFields {
		field, found := modelType.FieldByName(protoField.Name)
		
		if !found {
			validator.AddError("Model missing field %s corresponding to proto field %s", 
				protoField.Name, protoField.ProtoName)
			continue
		}
		
		// Check if the field has a gorm tag
		gormTag := field.Tag.Get("gorm")
		if gormTag == "" {
			validator.AddError("Field %s is missing gorm tag", protoField.Name)
		}
		
		// Check if the field has a json tag that matches the proto field name
		jsonTag := field.Tag.Get("json")
		jsonName := strings.Split(jsonTag, ",")[0]
		
		snakeCaseProtoName := toSnakeCase(protoField.ProtoName)
		if jsonName != snakeCaseProtoName && jsonName != protoField.ProtoName {
			validator.AddError("Field %s has json tag %s but should match proto field %s", 
				protoField.Name, jsonName, protoField.ProtoName)
		}
		
		// Check field type compatibility
		// This could be expanded to check for more precise type compatibility
		fieldType := field.Type.String()
		if !isCompatibleType(fieldType, protoField.GoType) {
			validator.AddError("Field %s has type %s but proto field %s has type %s", 
				protoField.Name, fieldType, protoField.ProtoName, protoField.GoType)
		}
	}
	
	return validator
}

// getDeploymentProtoFields returns the fields in the proto Deployment message
func getDeploymentProtoFields() []ProtoField {
	// This could be generated automatically by analyzing the proto definitions
	return []ProtoField{
		{Name: "ID", ProtoName: "id", GoType: "string", Required: true},
		{Name: "Name", ProtoName: "name", GoType: "string", Required: true},
		{Name: "Domain", ProtoName: "domain", GoType: "string", Required: true},
		{Name: "CustomDomains", ProtoName: "custom_domains", GoType: "[]string", Required: false},
		{Name: "Type", ProtoName: "type", GoType: "int32", Required: true},
		{Name: "RepositoryURL", ProtoName: "repository_url", GoType: "*string", Required: false},
		{Name: "Branch", ProtoName: "branch", GoType: "string", Required: true},
		{Name: "BuildCommand", ProtoName: "build_command", GoType: "*string", Required: false},
		{Name: "InstallCommand", ProtoName: "install_command", GoType: "*string", Required: false},
		{Name: "Status", ProtoName: "status", GoType: "int32", Required: true},
		{Name: "HealthStatus", ProtoName: "health_status", GoType: "string", Required: true},
		{Name: "LastDeployedAt", ProtoName: "last_deployed_at", GoType: "time.Time", Required: true},
		{Name: "BandwidthUsage", ProtoName: "bandwidth_usage", GoType: "int64", Required: true},
		{Name: "StorageUsage", ProtoName: "storage_usage", GoType: "int64", Required: true},
		{Name: "CreatedAt", ProtoName: "created_at", GoType: "time.Time", Required: true},
		{Name: "BuildTime", ProtoName: "build_time", GoType: "int32", Required: true},
		{Name: "Size", ProtoName: "size", GoType: "string", Required: true},
		{Name: "Environment", ProtoName: "environment", GoType: "int32", Required: true},
	}
}

// isCompatibleType checks if a Go model type is compatible with the proto type
func isCompatibleType(modelType string, protoType string) bool {
	// Handle pointers
	modelType = strings.TrimPrefix(modelType, "*")
	protoType = strings.TrimPrefix(protoType, "*")
	
	// Check exact match
	if modelType == protoType {
		return true
	}
	
	// Handle common mappings
	switch protoType {
	case "int32":
		return modelType == "int" || modelType == "int32"
	case "int64":
		return modelType == "int64" || modelType == "int"
	case "[]string":
		return strings.Contains(modelType, "string") && (strings.HasPrefix(modelType, "[]") || strings.Contains(modelType, "jsonb"))
	}
	
	return false
}

// toSnakeCase converts a camelCase string to snake_case
func toSnakeCase(s string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	
	snake := matchFirstCap.ReplaceAllString(s, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ValidateAllModels validates all known models against their proto definitions
func ValidateAllModels() *ModelValidator {
	validator := NewModelValidator()
	
	// Add more model validations here as needed
	deploymentValidator := ValidateDeployment(reflect.TypeOf(struct{}{}))
	validator.Errors = append(validator.Errors, deploymentValidator.Errors...)
	
	return validator
}

// IsProtoCompatible checks if a GORM model is compatible with its proto definition
func IsProtoCompatible(modelType reflect.Type) bool {
	// Determine which validator to use based on the type name
	switch modelType.Name() {
	case "Deployment":
		return ValidateDeployment(modelType).IsValid()
	default:
		return true // No validation for this model type
	}
}

// GetProtoEnumValue converts a string representation to the corresponding proto enum value
func GetProtoEnumValue(enumType string, value string) (int32, error) {
	switch enumType {
	case "DeploymentType":
		switch strings.ToUpper(value) {
		case "UNSPECIFIED":
			return int32(deploymentsv1.DeploymentType_DEPLOYMENT_TYPE_UNSPECIFIED), nil
		case "DOCKER":
			return int32(deploymentsv1.DeploymentType_DOCKER), nil
		case "STATIC":
			return int32(deploymentsv1.DeploymentType_STATIC), nil
		case "NODE":
			return int32(deploymentsv1.DeploymentType_NODE), nil
		case "GO":
			return int32(deploymentsv1.DeploymentType_GO), nil
		case "PYTHON":
			return int32(deploymentsv1.DeploymentType_PYTHON), nil
		case "RUBY":
			return int32(deploymentsv1.DeploymentType_RUBY), nil
		case "RUST":
			return int32(deploymentsv1.DeploymentType_RUST), nil
		case "JAVA":
			return int32(deploymentsv1.DeploymentType_JAVA), nil
		case "PHP":
			return int32(deploymentsv1.DeploymentType_PHP), nil
		case "GENERIC":
			return int32(deploymentsv1.DeploymentType_GENERIC), nil
		default:
			return 0, fmt.Errorf("invalid DeploymentType value: %s", value)
		}
	case "DeploymentStatus":
		switch strings.ToUpper(value) {
		case "UNSPECIFIED":
			return int32(deploymentsv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED), nil
		case "CREATED":
			return int32(deploymentsv1.DeploymentStatus_CREATED), nil
		case "BUILDING":
			return int32(deploymentsv1.DeploymentStatus_BUILDING), nil
		case "RUNNING":
			return int32(deploymentsv1.DeploymentStatus_RUNNING), nil
		case "STOPPED":
			return int32(deploymentsv1.DeploymentStatus_STOPPED), nil
		case "FAILED":
			return int32(deploymentsv1.DeploymentStatus_FAILED), nil
		case "DEPLOYING":
			return int32(deploymentsv1.DeploymentStatus_DEPLOYING), nil
		default:
			return 0, fmt.Errorf("invalid DeploymentStatus value: %s", value)
		}
	case "Environment":
		switch strings.ToUpper(value) {
		case "UNSPECIFIED":
			return int32(deploymentsv1.Environment_ENVIRONMENT_UNSPECIFIED), nil
		case "PRODUCTION":
			return int32(deploymentsv1.Environment_PRODUCTION), nil
		case "STAGING":
			return int32(deploymentsv1.Environment_STAGING), nil
		case "DEVELOPMENT":
			return int32(deploymentsv1.Environment_DEVELOPMENT), nil
		default:
			return 0, fmt.Errorf("invalid Environment value: %s", value)
		}
	default:
		return 0, fmt.Errorf("unknown enum type: %s", enumType)
	}
}
