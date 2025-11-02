package database

import (
	"time"

	"gorm.io/gorm"
)

// Deployment represents a deployment in the database
type Deployment struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	Name           string    `gorm:"column:name" json:"name"`
	Domain         string    `gorm:"column:domain" json:"domain"`
	CustomDomains  string    `gorm:"column:custom_domains;type:jsonb" json:"custom_domains"` // Stored as JSON array
	Type           int32     `gorm:"column:type" json:"type"`                                // DeploymentType enum
	BuildStrategy  int32     `gorm:"column:build_strategy;default:0" json:"build_strategy"` // BuildStrategy enum
	RepositoryURL     *string   `gorm:"column:repository_url" json:"repository_url"`
	Branch            string    `gorm:"column:branch" json:"branch"`
	BuildCommand      *string   `gorm:"column:build_command" json:"build_command"`
	InstallCommand    *string   `gorm:"column:install_command" json:"install_command"`
	DockerfilePath    *string   `gorm:"column:dockerfile_path" json:"dockerfile_path"` // Path to Dockerfile (relative to repo root)
	ComposeFilePath   *string   `gorm:"column:compose_file_path" json:"compose_file_path"` // Path to compose file (relative to repo root)
	GitHubIntegrationID *string `gorm:"column:github_integration_id;index" json:"github_integration_id"` // GitHub integration ID for autodeploys
	Status         int32     `gorm:"column:status;default:0" json:"status"` // DeploymentStatus enum
	HealthStatus   string    `gorm:"column:health_status" json:"health_status"`
	Environment    int32     `gorm:"column:environment" json:"environment"` // Environment enum
	Groups         string    `gorm:"column:groups;type:jsonb;default:'[]'::jsonb" json:"groups"` // Optional groups/labels for organizing deployments (stored as JSON array)
	BandwidthUsage int64     `gorm:"column:bandwidth_usage;default:0" json:"bandwidth_usage"`
	StorageBytes   int64     `gorm:"column:storage_bytes;default:0" json:"storage_bytes"`
	BuildTime      int32     `gorm:"column:build_time;default:0" json:"build_time"`
	Size           string    `gorm:"column:size" json:"size"`
	LastDeployedAt time.Time `gorm:"column:last_deployed_at" json:"last_deployed_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index" json:"deleted_at"` // Soft delete timestamp
	OrganizationID string    `gorm:"column:organization_id;index" json:"organization_id"`
	CreatedBy      string    `gorm:"column:created_by;index" json:"created_by"`

	// Runtime/resource config for quotas/orchestrator
	Image        *string `gorm:"column:image" json:"image"`
	Port         *int32  `gorm:"column:port" json:"port"`
	Replicas     *int32  `gorm:"column:replicas" json:"replicas"`
	MemoryBytes  *int64  `gorm:"column:memory_bytes" json:"memory_bytes"`
	CPUShares    *int64  `gorm:"column:cpu_shares" json:"cpu_shares"`
	EnvVars      string  `gorm:"column:env_vars;type:jsonb" json:"env_vars"` // Legacy: Stored as JSON object {"KEY": "value"} for backward compatibility
	EnvFileContent string `gorm:"column:env_file_content;type:text" json:"env_file_content"` // Raw .env file content with comments
	ComposeYaml  string  `gorm:"column:compose_yaml;type:text" json:"compose_yaml"` // Docker Compose YAML content
}

func (Deployment) TableName() string {
	return "deployments"
}

// BeforeCreate hook to set timestamps
func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.LastDeployedAt.IsZero() {
		d.LastDeployedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (d *Deployment) BeforeUpdate(tx *gorm.DB) error {
	d.LastDeployedAt = time.Now()
	return nil
}

// OrganizationPlan defines default limits for an organization plan
type OrganizationPlan struct {
	ID                 string `gorm:"primaryKey" json:"id"`
	Name               string `gorm:"uniqueIndex" json:"name"`
	CPUCores           int    `json:"cpu_cores"`
	MemoryBytes        int64  `json:"memory_bytes"`
	DeploymentsMax     int    `json:"deployments_max"`
	BandwidthBytesMonth int64 `json:"bandwidth_bytes_month"`
	StorageBytes       int64  `json:"storage_bytes"`
}

func (OrganizationPlan) TableName() string { return "organization_plans" }

// OrgQuota allows per-organization overrides of plan limits
type OrgQuota struct {
	OrganizationID     string `gorm:"primaryKey" json:"organization_id"`
	PlanID             string `gorm:"index" json:"plan_id"`
	CPUCoresOverride   *int   `json:"cpu_cores_override"`
	MemoryBytesOverride *int64 `json:"memory_bytes_override"`
	DeploymentsMaxOverride *int `json:"deployments_max_override"`
	BandwidthBytesMonthOverride *int64 `json:"bandwidth_bytes_month_override"`
	StorageBytesOverride *int64 `json:"storage_bytes_override"`
}

func (OrgQuota) TableName() string { return "org_quotas" }

// OrgRole represents a reusable role definition within an organization (scoped permissions)
type OrgRole struct {
	ID             string `gorm:"primaryKey" json:"id"`
	OrganizationID string `gorm:"index" json:"organization_id"`
	Name           string `gorm:"index" json:"name"`
	// JSON-encoded list of permission strings, e.g., ["deployments.view","deployments.create","deployments.scale"]
	Permissions    string `gorm:"type:jsonb" json:"permissions"`
}

func (OrgRole) TableName() string { return "org_roles" }

// OrgRoleBinding binds a user to roles (optionally limited to a resource)
type OrgRoleBinding struct {
	ID             string `gorm:"primaryKey" json:"id"`
	OrganizationID string `gorm:"index" json:"organization_id"`
	UserID         string `gorm:"index" json:"user_id"`
	RoleID         string `gorm:"index" json:"role_id"`
	// Optional scoping to a deployment/resource; empty means org-wide
	ResourceType string `json:"resource_type"`
	ResourceID   string `gorm:"index" json:"resource_id"`
    // Optional selector for richer scoping (e.g., {"environment":"production"})
    ResourceSelector string `gorm:"type:jsonb" json:"resource_selector"`
}

func (OrgRoleBinding) TableName() string { return "org_role_bindings" }

// DeploymentUsageHourly stores hourly aggregated metrics to reduce raw data volume
type DeploymentUsageHourly struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	DeploymentID        string    `gorm:"index;not null" json:"deployment_id"`
	OrganizationID      string    `gorm:"index;not null" json:"organization_id"` // Denormalized for easier querying
	Hour                time.Time `gorm:"index" json:"hour"`                      // Truncated to hour
	AvgCPUUsage         float64   `json:"avg_cpu_usage"`
	AvgMemoryUsage      int64     `json:"avg_memory_usage"`
	BandwidthRxBytes    int64     `json:"bandwidth_rx_bytes"`  // Sum of incremental values
	BandwidthTxBytes    int64     `json:"bandwidth_tx_bytes"`  // Sum of incremental values
	DiskReadBytes       int64     `json:"disk_read_bytes"`     // Sum of incremental values
	DiskWriteBytes      int64     `json:"disk_write_bytes"`    // Sum of incremental values
	RequestCount        int64     `json:"request_count"`
	ErrorCount          int64     `json:"error_count"`
	SampleCount         int64     `json:"sample_count"`        // Number of raw metrics aggregated
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (DeploymentUsageHourly) TableName() string { return "deployment_usage_hourly" }

// Organization and members
type Organization struct {
    ID        string    `gorm:"primaryKey" json:"id"`
    Name      string    `json:"name"`
    Slug      string    `gorm:"uniqueIndex" json:"slug"`
    Plan      string    `json:"plan"`
    Status    string    `json:"status"`
    Domain    *string   `json:"domain"`
    Credits   int64     `gorm:"column:credits;default:0" json:"credits"` // Credits in cents ($0.01 units)
    CreatedAt time.Time `json:"created_at"`
}

func (Organization) TableName() string { return "organizations" }

type OrganizationMember struct {
    ID             string    `gorm:"primaryKey" json:"id"`
    OrganizationID string    `gorm:"index" json:"organization_id"`
    UserID         string    `gorm:"index" json:"user_id"`
    Role           string    `json:"role"`
    Status         string    `json:"status"`
    JoinedAt       time.Time `json:"joined_at"`
}

func (OrganizationMember) TableName() string { return "organization_members" }

// CreditTransaction tracks all credit additions and removals for audit and history
type CreditTransaction struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	OrganizationID string    `gorm:"index;not null" json:"organization_id"`
	AmountCents    int64     `json:"amount_cents"` // Positive for additions, negative for removals
	BalanceAfter   int64     `json:"balance_after"` // Credit balance after this transaction
	Type           string    `json:"type"`         // "payment", "admin_add", "admin_remove", "usage", "refund", etc.
	Source         string    `json:"source"`        // "stripe", "admin", "system", etc.
	Note           *string   `json:"note"`          // Optional note/reason
	CreatedBy      *string   `gorm:"index" json:"created_by"` // User ID who initiated (nullable for system/automatic)
	CreatedAt      time.Time `json:"created_at"`
}

func (CreditTransaction) TableName() string { return "credit_transactions" }

// GitHubIntegration stores GitHub OAuth tokens for users and organizations
type GitHubIntegration struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      *string   `gorm:"index;uniqueIndex:idx_user_github" json:"user_id"`      // Zitadel user ID (nullable)
	OrganizationID *string `gorm:"index;uniqueIndex:idx_org_github" json:"organization_id"` // Organization ID (nullable)
	Token       string    `gorm:"column:token" json:"token"`                             // Encrypted GitHub access token
	Username    string    `gorm:"column:username" json:"username"`                       // GitHub username
	Scope       string    `gorm:"column:scope" json:"scope"`                             // Granted OAuth scopes
	ConnectedAt time.Time `gorm:"column:connected_at" json:"connected_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (GitHubIntegration) TableName() string { return "github_integrations" }
