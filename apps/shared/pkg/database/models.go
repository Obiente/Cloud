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
	StartCommand      *string   `gorm:"column:start_command" json:"start_command"` // Start command for running the application
	DockerfilePath    *string   `gorm:"column:dockerfile_path" json:"dockerfile_path"` // Path to Dockerfile (relative to repo root)
	ComposeFilePath   *string   `gorm:"column:compose_file_path" json:"compose_file_path"` // Path to compose file (relative to repo root)
	BuildPath         *string   `gorm:"column:build_path" json:"build_path"` // Working directory for build (relative to repo root, defaults to ".")
	BuildOutputPath   *string   `gorm:"column:build_output_path" json:"build_output_path"` // Path to built output files (relative to repo root, auto-detected if empty)
	UseNginx          *bool     `gorm:"column:use_nginx;default:false" json:"use_nginx"` // Use nginx for static deployments
	NginxConfig       *string   `gorm:"column:nginx_config;type:text" json:"nginx_config"` // Custom nginx configuration (optional, uses default if empty)
	GitHubIntegrationID *string `gorm:"column:github_integration_id;index" json:"github_integration_id"` // GitHub integration ID for autodeploys
	Status         int32     `gorm:"column:status;default:0" json:"status"` // DeploymentStatus enum
	HealthStatus   string    `gorm:"column:health_status" json:"health_status"`
	Environment    int32     `gorm:"column:environment" json:"environment"` // Environment enum
	Groups         string    `gorm:"column:groups;type:jsonb" json:"groups"` // Optional groups/labels for organizing deployments (stored as JSON array)
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
	MaxVpsInstances    int    `gorm:"column:max_vps_instances;default:0" json:"max_vps_instances"` // Maximum VPS instances (0 = unlimited)
	BandwidthBytesMonth int64 `json:"bandwidth_bytes_month"`
	StorageBytes       int64  `json:"storage_bytes"`
	MinimumPaymentCents int64 `gorm:"column:minimum_payment_cents;default:0" json:"minimum_payment_cents"` // Minimum payment in cents to automatically upgrade to this plan
	MonthlyFreeCreditsCents int64 `gorm:"column:monthly_free_credits_cents;default:0" json:"monthly_free_credits_cents"` // Monthly free credits in cents granted to organizations on this plan
	TrialDays int `gorm:"column:trial_days;default:0" json:"trial_days"` // Number of trial days for Stripe subscriptions (0 = no trial)
	Description        string `gorm:"column:description;type:text" json:"description"` // Optional description of the plan
}

func (OrganizationPlan) TableName() string { return "organization_plans" }

// OrgQuota allows per-organization overrides of plan limits
type OrgQuota struct {
	OrganizationID     string `gorm:"primaryKey" json:"organization_id"`
	PlanID             string `gorm:"index" json:"plan_id"`
	CPUCoresOverride   *int   `json:"cpu_cores_override"`
	MemoryBytesOverride *int64 `json:"memory_bytes_override"`
	DeploymentsMaxOverride *int `json:"deployments_max_override"`
	MaxVpsInstancesOverride *int `gorm:"column:max_vps_instances_override" json:"max_vps_instances_override"` // Override for max VPS instances (0 = unlimited)
	BandwidthBytesMonthOverride *int64 `json:"bandwidth_bytes_month_override"`
	StorageBytesOverride *int64 `json:"storage_bytes_override"`
}

func (OrgQuota) TableName() string { return "org_quotas" }

// MonthlyCreditGrant tracks monthly free credit grants for metrics and recovery
type MonthlyCreditGrant struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID string    `gorm:"index;not null" json:"organization_id"`
	PlanID         string    `gorm:"index;not null" json:"plan_id"`
	GrantMonth     time.Time `gorm:"index;not null" json:"grant_month"` // First day of the month (YYYY-MM-01)
	AmountCents    int64     `gorm:"not null" json:"amount_cents"`
	GrantedAt      time.Time `gorm:"not null" json:"granted_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (MonthlyCreditGrant) TableName() string { return "monthly_credit_grants" }

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
	AvgMemoryUsage      float64   `json:"avg_memory_usage"` // Average memory bytes per second for the hour (byte-seconds / 3600)
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
    TotalPaidCents int64 `gorm:"column:total_paid_cents;default:0" json:"total_paid_cents"` // Total amount paid in cents (for safety check/auto-upgrade)
    AllowInterVMCommunication bool `gorm:"column:allow_inter_vm_communication;default:false" json:"allow_inter_vm_communication"` // Allow VMs in this organization to communicate with each other
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
    LastInviteSentAt *time.Time `gorm:"column:last_invite_sent_at" json:"last_invite_sent_at"` // Tracks when invite email was last successfully sent (for rate limiting)
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

// StripeWebhookEvent tracks processed Stripe webhook events for idempotency
type StripeWebhookEvent struct {
	ID          string     `gorm:"primaryKey" json:"id"` // Stripe event ID (evt_*)
	EventType   string     `gorm:"index;not null" json:"event_type"`
	ProcessedAt time.Time  `gorm:"not null" json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	// Extracted IDs for easier querying and display
	OrganizationID *string `gorm:"index;column:organization_id" json:"organization_id,omitempty"` // Organization ID if available
	CustomerID     *string `gorm:"index;column:customer_id" json:"customer_id,omitempty"`         // Stripe customer ID if available
	SubscriptionID *string `gorm:"index;column:subscription_id" json:"subscription_id,omitempty"` // Stripe subscription ID if available
	InvoiceID      *string `gorm:"index;column:invoice_id" json:"invoice_id,omitempty"`           // Stripe invoice ID if available
	CheckoutSessionID *string `gorm:"index;column:checkout_session_id" json:"checkout_session_id,omitempty"` // Stripe checkout session ID if available
}

func (StripeWebhookEvent) TableName() string { return "stripe_webhook_events" }

// BillingAccount stores Stripe customer and billing information for organizations
type BillingAccount struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	OrganizationID   string    `gorm:"index;not null;uniqueIndex:idx_org_billing" json:"organization_id"`
	StripeCustomerID *string    `gorm:"column:stripe_customer_id;uniqueIndex" json:"stripe_customer_id"`
	Status           string    `gorm:"column:status;default:ACTIVE" json:"status"` // "ACTIVE", "INACTIVE", "PAST_DUE", etc.
	BillingEmail     *string   `gorm:"column:billing_email" json:"billing_email"`
	CompanyName      *string   `gorm:"column:company_name" json:"company_name"`
	TaxID            *string   `gorm:"column:tax_id" json:"tax_id"`
	Address          *string   `gorm:"column:address;type:jsonb" json:"address"` // JSON-encoded address (nullable)
	BillingDate      *int      `gorm:"column:billing_date" json:"billing_date"` // Day of month (1-31) when billing occurs
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (BillingAccount) TableName() string { return "billing_accounts" }

// MonthlyBill represents a monthly bill/invoice for an organization
type MonthlyBill struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	OrganizationID string    `gorm:"index;not null" json:"organization_id"`
	BillingPeriodStart time.Time `gorm:"index;not null" json:"billing_period_start"` // Start of billing period
	BillingPeriodEnd   time.Time `gorm:"index;not null" json:"billing_period_end"`   // End of billing period
	AmountCents    int64     `gorm:"not null" json:"amount_cents"` // Total amount in cents
	Status         string    `gorm:"index;default:PENDING" json:"status"` // "PENDING", "PAID", "FAILED", "CANCELLED"
	PaidAt         *time.Time `gorm:"column:paid_at" json:"paid_at"` // When the bill was paid
	DueDate        time.Time `gorm:"index;not null" json:"due_date"` // When payment is due
	// Usage breakdown (stored as JSON for flexibility)
	UsageBreakdown string    `gorm:"column:usage_breakdown;type:jsonb" json:"usage_breakdown"` // JSON with CPU, Memory, Bandwidth, Storage costs
	Note           *string   `gorm:"column:note;type:text" json:"note"` // Optional note
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MonthlyBill) TableName() string { return "monthly_bills" }

// GitHubIntegration stores GitHub OAuth tokens for users and organizations
type GitHubIntegration struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      *string   `gorm:"index;uniqueIndex:idx_user_github" json:"user_id"`      // Zitadel user ID (nullable)
	OrganizationID *string `gorm:"index;uniqueIndex:idx_org_github" json:"organization_id"` // Organization ID (nullable)
	Token       string    `gorm:"column:token" json:"token"`                             // Encrypted GitHub access token
	RefreshToken *string  `gorm:"column:refresh_token" json:"refresh_token"`             // Refresh token (if using GitHub Apps with expiring tokens)
	Username    string    `gorm:"column:username" json:"username"`                       // GitHub username
	Scope       string    `gorm:"column:scope" json:"scope"`                             // Granted OAuth scopes
	TokenExpiresAt *time.Time `gorm:"column:token_expires_at" json:"token_expires_at"`    // Token expiration time (if applicable)
	ConnectedAt time.Time `gorm:"column:connected_at" json:"connected_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (GitHubIntegration) TableName() string { return "github_integrations" }

// BuildHistory stores historical build information for deployments
type BuildHistory struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	DeploymentID   string    `gorm:"index;not null" json:"deployment_id"`
	OrganizationID string    `gorm:"index;not null" json:"organization_id"`
	BuildNumber    int32     `gorm:"index:idx_deployment_build_number" json:"build_number"` // Sequential build number per deployment
	Status         int32     `gorm:"column:status;default:0" json:"status"`                  // BuildStatus enum (pending, building, success, failed)
	StartedAt      time.Time `gorm:"index" json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	BuildTime      int32     `json:"build_time"` // Duration in seconds
	TriggeredBy    string    `gorm:"index" json:"triggered_by"` // User ID who triggered the build
	
	// Build configuration snapshot (captured at build time)
	RepositoryURL    *string `gorm:"column:repository_url" json:"repository_url"`
	Branch           string  `gorm:"column:branch" json:"branch"`
	CommitSHA        *string `gorm:"column:commit_sha" json:"commit_sha"` // Git commit SHA if available
	BuildCommand     *string `gorm:"column:build_command" json:"build_command"`
	InstallCommand   *string `gorm:"column:install_command" json:"install_command"`
	StartCommand     *string `gorm:"column:start_command" json:"start_command"`
	DockerfilePath   *string `gorm:"column:dockerfile_path" json:"dockerfile_path"`
	ComposeFilePath  *string `gorm:"column:compose_file_path" json:"compose_file_path"`
	BuildStrategy    int32   `gorm:"column:build_strategy" json:"build_strategy"`
	
	// Build results
	ImageName   *string `gorm:"column:image_name" json:"image_name"`   // Built image name (for single container)
	ComposeYaml *string `gorm:"column:compose_yaml;type:text" json:"compose_yaml"` // Docker Compose YAML (for compose deployments)
	Size        *string `gorm:"column:size" json:"size"`                 // Human-readable bundle size
	Error       *string `gorm:"column:error;type:text" json:"error"`     // Error message if build failed
	
	// Build logs stored separately in build_logs table (linked by build_id)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BuildHistory) TableName() string { return "build_history" }

// BuildLog stores individual log lines for a build
type BuildLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BuildID   string    `gorm:"index;not null" json:"build_id"` // References build_history.id
	Line      string    `gorm:"type:text" json:"line"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
	Stderr    bool      `gorm:"default:false" json:"stderr"`
	LineNumber int32    `gorm:"index" json:"line_number"` // Sequential line number within the build
}

func (BuildLog) TableName() string { return "build_logs" }

// StrayContainer tracks containers that were running but don't exist in the database
// These are containers that were stopped by the cleanup process
type StrayContainer struct {
	ContainerID string    `gorm:"primaryKey;column:container_id" json:"container_id"`
	NodeID      string    `gorm:"index;column:node_id" json:"node_id"`
	StoppedAt   time.Time `gorm:"index;column:stopped_at" json:"stopped_at"`
	VolumesDeletedAt *time.Time `gorm:"column:volumes_deleted_at" json:"volumes_deleted_at"` // When volumes were deleted (if applicable)
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (StrayContainer) TableName() string { return "stray_containers" }

// GameServer represents a game server instance in the database
type GameServer struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	Name           string    `gorm:"column:name" json:"name"`
	Description    *string   `gorm:"column:description" json:"description"`
	GameType       int32     `gorm:"column:game_type" json:"game_type"` // GameType enum
	Status         int32     `gorm:"column:status;default:0" json:"status"` // GameServerStatus enum
	
	// Resource configuration
	MemoryBytes    int64     `gorm:"column:memory_bytes" json:"memory_bytes"`
	CPUCores       int32     `gorm:"column:cpu_cores" json:"cpu_cores"`
	Port           int32     `gorm:"column:port" json:"port"`
	
	// Docker configuration
	DockerImage    string    `gorm:"column:docker_image" json:"docker_image"`
	StartCommand   *string   `gorm:"column:start_command" json:"start_command"`
	
	// Environment variables (stored as JSON object)
	EnvVars        string    `gorm:"column:env_vars;type:jsonb" json:"env_vars"`
	
	// Game-specific configuration
	ServerVersion  *string   `gorm:"column:server_version" json:"server_version"`
	
	// Container information
	ContainerID    *string   `gorm:"column:container_id" json:"container_id"`
	ContainerName  *string   `gorm:"column:container_name" json:"container_name"`
	
	// Resource usage
	StorageBytes   int64     `gorm:"column:storage_bytes;default:0" json:"storage_bytes"`
	BandwidthUsage int64     `gorm:"column:bandwidth_usage;default:0" json:"bandwidth_usage"`
	
	// Player information (if available)
	PlayerCount    *int32    `gorm:"column:player_count" json:"player_count"`
	MaxPlayers     *int32    `gorm:"column:max_players" json:"max_players"`
	
	// Timestamps
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
	LastStartedAt  *time.Time `gorm:"column:last_started_at" json:"last_started_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index" json:"deleted_at"` // Soft delete
	
	// Organization and ownership
	OrganizationID string    `gorm:"column:organization_id;index" json:"organization_id"`
	CreatedBy      string    `gorm:"column:created_by;index" json:"created_by"`
}

func (GameServer) TableName() string {
	return "game_servers"
}

// BeforeCreate hook to set timestamps
func (gs *GameServer) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if gs.CreatedAt.IsZero() {
		gs.CreatedAt = now
	}
	if gs.UpdatedAt.IsZero() {
		gs.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (gs *GameServer) BeforeUpdate(tx *gorm.DB) error {
	gs.UpdatedAt = time.Now()
	return nil
}

// GameServerUsageHourly stores hourly aggregated metrics for game servers
type GameServerUsageHourly struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	GameServerID        string    `gorm:"index;not null" json:"game_server_id"`
	OrganizationID      string    `gorm:"index;not null" json:"organization_id"` // Denormalized for easier querying
	Hour                time.Time `gorm:"index" json:"hour"`                      // Truncated to hour
	AvgCPUUsage         float64   `json:"avg_cpu_usage"`
	AvgMemoryUsage      float64   `json:"avg_memory_usage"` // Average memory bytes per second for the hour (byte-seconds / 3600)
	BandwidthRxBytes    int64     `json:"bandwidth_rx_bytes"`  // Sum of incremental values
	BandwidthTxBytes    int64     `json:"bandwidth_tx_bytes"`  // Sum of incremental values
	DiskReadBytes       int64     `json:"disk_read_bytes"`     // Sum of incremental values
	DiskWriteBytes      int64     `json:"disk_write_bytes"`     // Sum of incremental values
	SampleCount         int64     `json:"sample_count"`        // Number of raw metrics aggregated
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (GameServerUsageHourly) TableName() string { return "game_server_usage_hourly" }

// VPSInstance represents a VPS (Virtual Private Server) instance in the database
type VPSInstance struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	Name           string    `gorm:"column:name" json:"name"`
	Description    *string   `gorm:"column:description" json:"description"`
	Status         int32     `gorm:"column:status;default:0" json:"status"` // VPSStatus enum
	Region         string    `gorm:"column:region" json:"region"`
	Image          int32     `gorm:"column:image" json:"image"` // VPSImage enum
	ImageID        *string   `gorm:"column:image_id" json:"image_id"` // Custom image ID
	Size           string    `gorm:"column:size" json:"size"` // Provider size ID (e.g., "cx11", "s-1vcpu-1gb")
	
	// Resource specifications
	CPUCores       int32     `gorm:"column:cpu_cores" json:"cpu_cores"`
	MemoryBytes    int64     `gorm:"column:memory_bytes" json:"memory_bytes"`
	DiskBytes      int64     `gorm:"column:disk_bytes" json:"disk_bytes"`
	
	// Network information (stored as JSON arrays)
	IPv4Addresses  string    `gorm:"column:ipv4_addresses;type:jsonb" json:"ipv4_addresses"`
	IPv6Addresses  string    `gorm:"column:ipv6_addresses;type:jsonb" json:"ipv6_addresses"`
	
	// Infrastructure information
	InstanceID *string `gorm:"column:instance_id;index" json:"instance_id"` // Internal instance ID
	NodeID     *string `gorm:"column:node_id;index" json:"node_id"`          // Docker Swarm node ID where VPS is running
	
	// SSH access
	SSHKeyID       *string   `gorm:"column:ssh_key_id" json:"ssh_key_id"`
	SSHAlias       *string   `gorm:"column:ssh_alias;index;unique" json:"ssh_alias"` // Short memorable alias for SSH (e.g., "prod-db", "web-1")
	
	// NOTE: Root password is NEVER stored in the database for security
	// Password is only returned once in CreateVPS response, then discarded
	
	// Metadata (stored as JSON object)
	Metadata       string    `gorm:"column:metadata;type:jsonb" json:"metadata"`
	
	// Timestamps
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
	LastStartedAt  *time.Time `gorm:"column:last_started_at" json:"last_started_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index" json:"deleted_at"` // Soft delete
	
	// Organization and ownership
	OrganizationID string    `gorm:"column:organization_id;index" json:"organization_id"`
	CreatedBy      string    `gorm:"column:created_by;index" json:"created_by"`
}

func (VPSInstance) TableName() string {
	return "vps_instances"
}

// BeforeCreate hook to set timestamps
func (vps *VPSInstance) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if vps.CreatedAt.IsZero() {
		vps.CreatedAt = now
	}
	if vps.UpdatedAt.IsZero() {
		vps.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (vps *VPSInstance) BeforeUpdate(tx *gorm.DB) error {
	vps.UpdatedAt = time.Now()
	return nil
}

// VPSUsageHourly stores hourly aggregated metrics for VPS instances
type VPSUsageHourly struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	VPSInstanceID       string    `gorm:"index;not null" json:"vps_instance_id"`
	OrganizationID      string    `gorm:"index;not null" json:"organization_id"` // Denormalized for easier querying
	Hour                time.Time `gorm:"index" json:"hour"`                      // Truncated to hour
	AvgCPUUsage         float64   `json:"avg_cpu_usage"`                          // Average CPU usage percentage
	AvgMemoryUsage      float64   `json:"avg_memory_usage"`                       // Average memory bytes per second for the hour (byte-seconds / 3600)
	BandwidthRxBytes    int64     `json:"bandwidth_rx_bytes"`                    // Sum of incremental values
	BandwidthTxBytes    int64     `json:"bandwidth_tx_bytes"`                    // Sum of incremental values
	DiskReadBytes       int64     `json:"disk_read_bytes"`                       // Sum of incremental values
	DiskWriteBytes      int64     `json:"disk_write_bytes"`                       // Sum of incremental values
	UptimeSeconds       int64     `json:"uptime_seconds"`                         // Total uptime in seconds for the hour
	SampleCount         int64     `json:"sample_count"`                           // Number of raw metrics aggregated
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (VPSUsageHourly) TableName() string { return "vps_usage_hourly" }

// VPSMetrics stores historical metrics for VPS instances
type VPSMetrics struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	VPSInstanceID string    `gorm:"index;not null" json:"vps_instance_id"`
	InstanceID    string    `gorm:"index" json:"instance_id"` // Internal instance ID
	NodeID        string    `gorm:"index" json:"node_id"`      // Docker Swarm node ID
	CPUUsage      float64   `json:"cpu_usage"`              // CPU usage percentage (0-100)
	MemoryUsed    int64     `json:"memory_used"`            // Memory used in bytes
	MemoryTotal   int64     `json:"memory_total"`           // Total memory in bytes
	DiskUsed      int64     `json:"disk_used"`              // Disk used in bytes
	DiskTotal     int64     `json:"disk_total"`              // Total disk in bytes
	NetworkRxBytes int64    `json:"network_rx_bytes"`       // Network received bytes
	NetworkTxBytes int64    `json:"network_tx_bytes"`       // Network transmitted bytes
	DiskReadIOPS  float64   `json:"disk_read_iops"`        // Disk read IOPS
	DiskWriteIOPS float64   `json:"disk_write_iops"`        // Disk write IOPS
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
}

func (VPSMetrics) TableName() string { return "vps_metrics" }

// GameServerMetrics stores historical metrics for game servers
type GameServerMetrics struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	GameServerID  string    `gorm:"index;not null" json:"game_server_id"`
	ContainerID   string    `gorm:"index" json:"container_id"`
	NodeID        string    `gorm:"index" json:"node_id"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   int64     `json:"memory_usage"`
	NetworkRxBytes int64    `json:"network_rx_bytes"`
	NetworkTxBytes int64    `json:"network_tx_bytes"`
	DiskReadBytes int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64    `json:"disk_write_bytes"`
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
}

func (GameServerMetrics) TableName() string { return "game_server_metrics" }

// SupportTicket represents a support ticket in the database
type SupportTicket struct {
	ID           string     `gorm:"primaryKey;column:id" json:"id"`
	Subject      string     `gorm:"column:subject;not null" json:"subject"`
	Description  string     `gorm:"column:description;type:text;not null" json:"description"`
	Status       int32      `gorm:"column:status;default:1" json:"status"` // SupportTicketStatus enum (1=OPEN)
	Priority     int32      `gorm:"column:priority;default:2" json:"priority"` // SupportTicketPriority enum (2=MEDIUM)
	Category     int32      `gorm:"column:category;default:0" json:"category"` // SupportTicketCategory enum
	CreatedBy    string     `gorm:"column:created_by;index;not null" json:"created_by"` // User ID who created the ticket
	AssignedTo   *string    `gorm:"column:assigned_to;index" json:"assigned_to"` // User ID of assignee (superadmin)
	OrganizationID *string  `gorm:"column:organization_id;index" json:"organization_id"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`
	ResolvedAt   *time.Time `gorm:"column:resolved_at" json:"resolved_at"`
}

func (SupportTicket) TableName() string {
	return "support_tickets"
}

// BeforeCreate hook to set timestamps
func (st *SupportTicket) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if st.CreatedAt.IsZero() {
		st.CreatedAt = now
	}
	if st.UpdatedAt.IsZero() {
		st.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (st *SupportTicket) BeforeUpdate(tx *gorm.DB) error {
	st.UpdatedAt = time.Now()
	return nil
}

// TicketComment represents a comment/reply on a support ticket
type TicketComment struct {
	ID        string    `gorm:"primaryKey;column:id" json:"id"`
	TicketID  string    `gorm:"column:ticket_id;index;not null" json:"ticket_id"`
	Content   string    `gorm:"column:content;type:text;not null" json:"content"`
	CreatedBy string    `gorm:"column:created_by;index;not null" json:"created_by"` // User ID who created the comment
	Internal  bool      `gorm:"column:internal;default:false" json:"internal"` // Internal comment (not visible to user)
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (TicketComment) TableName() string {
	return "ticket_comments"
}

// BeforeCreate hook to set timestamps
func (tc *TicketComment) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if tc.CreatedAt.IsZero() {
		tc.CreatedAt = now
	}
	if tc.UpdatedAt.IsZero() {
		tc.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (tc *TicketComment) BeforeUpdate(tx *gorm.DB) error {
	tc.UpdatedAt = time.Now()
	return nil
}

// AuditLog represents an audit log entry for tracking all actions
type AuditLog struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	UserID         string    `gorm:"column:user_id;index;not null" json:"user_id"`         // User who performed the action
	OrganizationID *string   `gorm:"column:organization_id;index" json:"organization_id"` // Organization context (nullable for system actions)
	Action         string    `gorm:"column:action;index;not null" json:"action"`          // RPC method name (e.g., "CreateDeployment")
	Service        string    `gorm:"column:service;index;not null" json:"service"`        // Service name (e.g., "DeploymentService")
	ResourceType   *string   `gorm:"column:resource_type;index" json:"resource_type"`     // Type of resource affected (e.g., "deployment", "organization")
	ResourceID     *string   `gorm:"column:resource_id;index" json:"resource_id"`          // ID of the affected resource
	IPAddress      string    `gorm:"column:ip_address" json:"ip_address"`                 // Client IP address
	UserAgent      string    `gorm:"column:user_agent" json:"user_agent"`                 // User agent string
	RequestData    string    `gorm:"column:request_data;type:jsonb" json:"request_data"` // Request payload (sanitized)
	ResponseStatus int32     `gorm:"column:response_status" json:"response_status"`       // HTTP/Connect status code
	ErrorMessage   *string   `gorm:"column:error_message;type:text" json:"error_message"` // Error message if action failed
	DurationMs     int64     `gorm:"column:duration_ms" json:"duration_ms"`              // Request duration in milliseconds
	CreatedAt      time.Time `gorm:"column:created_at;index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// SSHKey represents an SSH public key for VPS access
// If VPSID is null, the key is organization-wide and applies to all VPS instances in the organization
// If VPSID is set, the key is specific to that VPS instance
type SSHKey struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	OrganizationID string     `gorm:"column:organization_id;index;not null" json:"organization_id"`
	VPSID          *string    `gorm:"column:vps_id;index" json:"vps_id"` // If null, key is org-wide; if set, key is VPS-specific
	Name           string     `gorm:"column:name;not null" json:"name"` // User-friendly name
	PublicKey      string     `gorm:"column:public_key;type:text;not null" json:"public_key"` // SSH public key content
	Fingerprint    string     `gorm:"column:fingerprint;index" json:"fingerprint"`            // SSH key fingerprint
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (SSHKey) TableName() string {
	return "ssh_keys"
}

// VPSTerminalKey represents an SSH key pair for web terminal access to a VPS
// The private key is stored encrypted (or should be in production)
// The public key is added to the VPS via cloud-init
type VPSTerminalKey struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	VPSID          string    `gorm:"column:vps_id;index;not null;unique" json:"vps_id"` // One key per VPS
	OrganizationID string    `gorm:"column:organization_id;index;not null" json:"organization_id"`
	PublicKey      string    `gorm:"column:public_key;type:text;not null" json:"public_key"` // SSH public key
	PrivateKey     string    `gorm:"column:private_key;type:text;not null" json:"-"`         // SSH private key (not returned in JSON)
	Fingerprint    string    `gorm:"column:fingerprint;index" json:"fingerprint"`            // SSH key fingerprint
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (VPSTerminalKey) TableName() string {
	return "vps_terminal_keys"
}

// VPSBastionKey represents an SSH key pair for SSH bastion host connections to a VPS
// The private key is stored encrypted (or should be in production)
// The public key is added to the VPS via cloud-init
type VPSBastionKey struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	VPSID          string    `gorm:"column:vps_id;index;not null;unique" json:"vps_id"` // One key per VPS
	OrganizationID string    `gorm:"column:organization_id;index;not null" json:"organization_id"`
	PublicKey      string    `gorm:"column:public_key;type:text;not null" json:"public_key"` // SSH public key
	PrivateKey     string    `gorm:"column:private_key;type:text;not null" json:"-"`         // SSH private key (not returned in JSON)
	Fingerprint    string    `gorm:"column:fingerprint;index" json:"fingerprint"`            // SSH key fingerprint
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (VPSBastionKey) TableName() string {
	return "vps_bastion_keys"
}
