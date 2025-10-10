// Core entity types for Obiente Cloud Dashboard

export interface Organization {
  id: string;
  name: string;
  slug: string;
  domain?: string;
  plan: "starter" | "pro" | "enterprise";
  status: "active" | "suspended" | "trial";
  createdAt: Date;
  updatedAt: Date;
  billingEmail: string;
  billingAddress?: Address;
  maxDeployments: number;
  maxVpsInstances: number;
  maxTeamMembers: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
  createdAt: Date;
  updatedAt: Date;
  lastLoginAt?: Date;
  externalId: string;
  timezone?: string;
  notificationPreferences?: NotificationSettings;
}

export interface OrganizationMember {
  id: string;
  organizationId: string;
  userId: string;
  role: "owner" | "admin" | "member" | "viewer";
  status: "active" | "invited" | "suspended";
  invitedBy?: string;
  invitedAt?: Date;
  joinedAt?: Date;
  createdAt: Date;
  updatedAt: Date;
  customPermissions?: Permission[];
}

export interface Deployment {
  id: string;
  organizationId: string;
  name: string;
  domain: string;
  customDomains: string[];
  type: "static" | "nodejs" | "python" | "docker";
  repositoryUrl?: string;
  branch: string;
  buildCommand?: string;
  installCommand?: string;
  outputDirectory?: string;
  status: "building" | "ready" | "error" | "stopped";
  healthStatus: "healthy" | "degraded" | "unhealthy";
  lastDeployedAt?: Date;
  bandwidthUsage: number;
  storageUsage: number;
  createdAt: Date;
  updatedAt: Date;
  createdBy: string;
}

export interface VPSInstance {
  id: string;
  organizationId: string;
  name: string;
  plan: "small" | "medium" | "large" | "xlarge";
  cpuCores: number;
  memoryGb: number;
  diskGb: number;
  operatingSystem: string;
  region: string;
  ipAddress: string;
  privateIp?: string;
  status: "starting" | "running" | "stopped" | "error" | "terminated";
  uptimePercentage: number;
  cpuUsagePercent: number;
  memoryUsagePercent: number;
  diskUsagePercent: number;
  bandwidthUsage: number;
  createdAt: Date;
  updatedAt: Date;
  createdBy: string;
}

export interface Database {
  id: string;
  organizationId: string;
  deploymentId?: string;
  name: string;
  type: "postgresql" | "mysql" | "redis" | "mongodb";
  version: string;
  plan: "shared" | "dedicated-small" | "dedicated-medium" | "dedicated-large";
  host: string;
  port: number;
  username: string;
  databaseName: string;
  status: "creating" | "available" | "maintenance" | "error";
  storageUsageGb: number;
  connectionCount: number;
  createdAt: Date;
  updatedAt: Date;
  createdBy: string;
}

export interface BillingAccount {
  id: string;
  organizationId: string;
  stripeCustomerId: string;
  stripeSubscriptionId?: string;
  billingEmail: string;
  billingAddress: Address;
  paymentMethodId?: string;
  currentPeriodStart: Date;
  currentPeriodEnd: Date;
  currentUsage: UsageMetrics;
  createdAt: Date;
  updatedAt: Date;
}

export interface Invoice {
  id: string;
  billingAccountId: string;
  invoiceNumber: string;
  stripeInvoiceId: string;
  amountTotal: number;
  amountPaid: number;
  currency: string;
  periodStart: Date;
  periodEnd: Date;
  status: "draft" | "open" | "paid" | "void" | "uncollectible";
  dueDate: Date;
  paidAt?: Date;
  createdAt: Date;
  updatedAt: Date;
}

export interface Permission {
  id: string;
  organizationMemberId: string;
  resourceType: "deployment" | "vps" | "database" | "billing";
  resourceId: string;
  permissions: string[];
  createdAt: Date;
  updatedAt: Date;
}

export interface AuditLog {
  id: string;
  organizationId: string;
  userId?: string;
  action: string;
  resourceType: string;
  resourceId: string;
  ipAddress: string;
  userAgent: string;
  metadata: Record<string, any>;
  createdAt: Date;
}

// Supporting types

export interface Address {
  line1: string;
  line2?: string;
  city: string;
  state?: string;
  postalCode: string;
  country: string;
}

export interface UsageMetrics {
  deploymentsCount: number;
  vpsInstancesCount: number;
  databasesCount: number;
  bandwidthGb: number;
  storageGb: number;
  computeHours: number;
}

export interface NotificationSettings {
  emailNotifications: boolean;
  deploymentAlerts: boolean;
  billingAlerts: boolean;
  securityAlerts: boolean;
  marketingEmails: boolean;
}

// API response types

export interface ApiResponse<T = any> {
  data?: T;
  error?: ApiError;
  meta?: ApiMeta;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

export interface ApiMeta {
  pagination?: Pagination;
  total?: number;
}

export interface Pagination {
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
}

// Authentication types

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
  organizations: OrganizationMembership[];
}

export interface OrganizationMembership {
  organizationId: string;
  organizationName: string;
  organizationSlug: string;
  role: OrganizationMember["role"];
  status: OrganizationMember["status"];
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

// Utility types

export type ResourceType =
  | "organization"
  | "deployment"
  | "vps"
  | "database"
  | "billing";
export type PermissionAction =
  | "read"
  | "write"
  | "delete"
  | "deploy"
  | "manage";
export type OrganizationRole = OrganizationMember["role"];
export type DeploymentStatus = Deployment["status"];
export type VPSStatus = VPSInstance["status"];
export type DatabaseStatus = Database["status"];

// Form types

export interface CreateOrganizationForm {
  name: string;
  slug: string;
  plan: Organization["plan"];
}

export interface CreateDeploymentForm {
  name: string;
  type: Deployment["type"];
  repositoryUrl?: string;
  branch: string;
  buildCommand?: string;
  installCommand?: string;
  outputDirectory?: string;
}

export interface CreateVPSForm {
  name: string;
  plan: VPSInstance["plan"];
  operatingSystem: string;
  region: string;
}

export interface CreateDatabaseForm {
  name: string;
  type: Database["type"];
  plan: Database["plan"];
  deploymentId?: string;
}

export interface InviteMemberForm {
  email: string;
  role: OrganizationMember["role"];
}
