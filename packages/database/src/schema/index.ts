import {
  pgTable,
  uuid,
  varchar,
  text,
  timestamp,
  boolean,
  jsonb,
  integer,
  decimal,
  pgEnum,
} from "drizzle-orm/pg-core";
import { relations } from "drizzle-orm";

// Enums
export const userRoleEnum = pgEnum("user_role", [
  "OWNER",
  "ADMIN",
  "DEVELOPER",
  "VIEWER",
]);
export const deploymentStatusEnum = pgEnum("deployment_status", [
  "PENDING",
  "BUILDING",
  "RUNNING",
  "STOPPED",
  "FAILED",
]);
export const vpsStatusEnum = pgEnum("vps_status", [
  "CREATING",
  "RUNNING",
  "STOPPED",
  "REBOOTING",
  "TERMINATED",
]);
export const databaseStatusEnum = pgEnum("database_status", [
  "CREATING",
  "AVAILABLE",
  "BACKING_UP",
  "MAINTENANCE",
  "FAILED",
]);
export const billingStatusEnum = pgEnum("billing_status", [
  "ACTIVE",
  "PAST_DUE",
  "SUSPENDED",
  "CANCELLED",
]);
export const subscriptionStatusEnum = pgEnum("subscription_status", [
  "ACTIVE",
  "PAST_DUE",
  "CANCELLED",
  "UNPAID",
]);

// Organizations table
export const organizations = pgTable("organizations", {
  id: uuid("id").defaultRandom().primaryKey(),
  name: varchar("name", { length: 255 }).notNull(),
  slug: varchar("slug", { length: 100 }).notNull().unique(),
  description: text("description"),
  settings: jsonb("settings").default({}),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Users table
export const users = pgTable("users", {
  id: uuid("id").defaultRandom().primaryKey(),
  externalId: varchar("external_id", { length: 255 }).notNull().unique(), // Zitadel user ID
  email: varchar("email", { length: 255 }).notNull().unique(),
  name: varchar("name", { length: 255 }).notNull(),
  avatar: varchar("avatar", { length: 500 }),
  preferences: jsonb("preferences").default({}),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Organization memberships table
export const organizationMemberships = pgTable("organization_memberships", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  userId: uuid("user_id")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  role: userRoleEnum("role").notNull(),
  joinedAt: timestamp("joined_at").defaultNow().notNull(),
  invitedBy: uuid("invited_by").references(() => users.id),
});

// Deployments table
export const deployments = pgTable("deployments", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 255 }).notNull(),
  domain: varchar("domain", { length: 255 }).notNull(),
  repositoryUrl: varchar("repository_url", { length: 500 }),
  buildCommand: varchar("build_command", { length: 255 }),
  outputDirectory: varchar("output_directory", { length: 255 }),
  environmentVariables: jsonb("environment_variables").default({}),
  status: deploymentStatusEnum("status").notNull().default("PENDING"),
  lastDeployedAt: timestamp("last_deployed_at"),
  createdBy: uuid("created_by")
    .notNull()
    .references(() => users.id),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// VPS Instances table
export const vpsInstances = pgTable("vps_instances", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 255 }).notNull(),
  size: varchar("size", { length: 50 }).notNull(), // e.g., 'small', 'medium', 'large'
  region: varchar("region", { length: 50 }).notNull(),
  image: varchar("image", { length: 100 }).notNull(), // OS image
  ipAddress: varchar("ip_address", { length: 45 }),
  status: vpsStatusEnum("status").notNull().default("CREATING"),
  sshKeys: jsonb("ssh_keys").default([]),
  tags: jsonb("tags").default({}),
  createdBy: uuid("created_by")
    .notNull()
    .references(() => users.id),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Database Instances table
export const databaseInstances = pgTable("database_instances", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 255 }).notNull(),
  type: varchar("type", { length: 50 }).notNull(), // 'postgresql', 'mysql', 'redis'
  version: varchar("version", { length: 20 }).notNull(),
  size: varchar("size", { length: 50 }).notNull(),
  region: varchar("region", { length: 50 }).notNull(),
  connectionString: varchar("connection_string", { length: 500 }),
  status: databaseStatusEnum("status").notNull().default("CREATING"),
  backupRetentionDays: integer("backup_retention_days").default(7),
  maintenanceWindow: varchar("maintenance_window", { length: 100 }),
  tags: jsonb("tags").default({}),
  createdBy: uuid("created_by")
    .notNull()
    .references(() => users.id),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Billing Accounts table
export const billingAccounts = pgTable("billing_accounts", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  stripeCustomerId: varchar("stripe_customer_id", { length: 255 }).unique(),
  status: billingStatusEnum("status").notNull().default("ACTIVE"),
  billingEmail: varchar("billing_email", { length: 255 }),
  companyName: varchar("company_name", { length: 255 }),
  taxId: varchar("tax_id", { length: 100 }),
  address: jsonb("address"),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Subscriptions table
export const subscriptions = pgTable("subscriptions", {
  id: uuid("id").defaultRandom().primaryKey(),
  billingAccountId: uuid("billing_account_id")
    .notNull()
    .references(() => billingAccounts.id, { onDelete: "cascade" }),
  stripeSubscriptionId: varchar("stripe_subscription_id", {
    length: 255,
  }).unique(),
  status: subscriptionStatusEnum("status").notNull(),
  planName: varchar("plan_name", { length: 100 }).notNull(),
  planPrice: decimal("plan_price", { precision: 10, scale: 2 }).notNull(),
  billingInterval: varchar("billing_interval", { length: 20 }).notNull(), // 'month', 'year'
  currentPeriodStart: timestamp("current_period_start").notNull(),
  currentPeriodEnd: timestamp("current_period_end").notNull(),
  cancelledAt: timestamp("cancelled_at"),
  createdAt: timestamp("created_at").defaultNow().notNull(),
  updatedAt: timestamp("updated_at").defaultNow().notNull(),
});

// Usage Records table
export const usageRecords = pgTable("usage_records", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  resourceType: varchar("resource_type", { length: 50 }).notNull(), // 'deployment', 'vps', 'database'
  resourceId: uuid("resource_id").notNull(),
  metricName: varchar("metric_name", { length: 100 }).notNull(), // 'bandwidth', 'storage', 'compute_hours'
  quantity: decimal("quantity", { precision: 15, scale: 6 }).notNull(),
  unit: varchar("unit", { length: 20 }).notNull(), // 'GB', 'hours', 'requests'
  cost: decimal("cost", { precision: 10, scale: 6 }).notNull(),
  periodStart: timestamp("period_start").notNull(),
  periodEnd: timestamp("period_end").notNull(),
  createdAt: timestamp("created_at").defaultNow().notNull(),
});

// Relations
export const organizationsRelations = relations(organizations, ({ many }) => ({
  memberships: many(organizationMemberships),
  deployments: many(deployments),
  vpsInstances: many(vpsInstances),
  databaseInstances: many(databaseInstances),
  billingAccount: many(billingAccounts),
  usageRecords: many(usageRecords),
}));

export const usersRelations = relations(users, ({ many }) => ({
  memberships: many(organizationMemberships),
  createdDeployments: many(deployments),
  createdVpsInstances: many(vpsInstances),
  createdDatabaseInstances: many(databaseInstances),
  invitedMemberships: many(organizationMemberships, {
    relationName: "invitedBy",
  }),
}));

export const organizationMembershipsRelations = relations(
  organizationMemberships,
  ({ one }) => ({
    organization: one(organizations, {
      fields: [organizationMemberships.organizationId],
      references: [organizations.id],
    }),
    user: one(users, {
      fields: [organizationMemberships.userId],
      references: [users.id],
    }),
    invitedBy: one(users, {
      fields: [organizationMemberships.invitedBy],
      references: [users.id],
      relationName: "invitedBy",
    }),
  })
);

export const deploymentsRelations = relations(deployments, ({ one }) => ({
  organization: one(organizations, {
    fields: [deployments.organizationId],
    references: [organizations.id],
  }),
  createdBy: one(users, {
    fields: [deployments.createdBy],
    references: [users.id],
  }),
}));

export const vpsInstancesRelations = relations(vpsInstances, ({ one }) => ({
  organization: one(organizations, {
    fields: [vpsInstances.organizationId],
    references: [organizations.id],
  }),
  createdBy: one(users, {
    fields: [vpsInstances.createdBy],
    references: [users.id],
  }),
}));

export const databaseInstancesRelations = relations(
  databaseInstances,
  ({ one }) => ({
    organization: one(organizations, {
      fields: [databaseInstances.organizationId],
      references: [organizations.id],
    }),
    createdBy: one(users, {
      fields: [databaseInstances.createdBy],
      references: [users.id],
    }),
  })
);

export const billingAccountsRelations = relations(
  billingAccounts,
  ({ one, many }) => ({
    organization: one(organizations, {
      fields: [billingAccounts.organizationId],
      references: [organizations.id],
    }),
    subscriptions: many(subscriptions),
  })
);

export const subscriptionsRelations = relations(subscriptions, ({ one }) => ({
  billingAccount: one(billingAccounts, {
    fields: [subscriptions.billingAccountId],
    references: [billingAccounts.id],
  }),
}));

export const usageRecordsRelations = relations(usageRecords, ({ one }) => ({
  organization: one(organizations, {
    fields: [usageRecords.organizationId],
    references: [organizations.id],
  }),
}));
