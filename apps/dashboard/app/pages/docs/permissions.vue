<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Permissions & Access Control
      </OuiText>
      <OuiText color="secondary" size="lg">
        Understand how permissions work and how to manage access in Obiente Cloud
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="secondary">
          Obiente Cloud uses a flexible role-based access control (RBAC) system
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            The permissions system allows fine-grained control over what users can do in your organization.
            Permissions can be granted through system roles, custom roles, and role bindings.
          </OuiText>

          <OuiGrid cols="1" cols-md="3" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">System Roles</OuiText>
                <OuiText size="sm" color="secondary">
                  Predefined roles (Owner, Admin, Member, Viewer, None) with hardcoded permissions
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Custom Roles</OuiText>
                <OuiText size="sm" color="secondary">
                  Organization-specific roles with configurable permissions
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Role Bindings</OuiText>
                <OuiText size="sm" color="secondary">
                  Assign roles to users, optionally scoped to specific resources
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">System Roles</OuiText>
        <OuiText size="sm" color="secondary">
          Predefined roles available to all organizations
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            System roles are defined in code and cannot be modified or deleted. They provide common
            access patterns that work for most organizations.
          </OuiText>

          <OuiAccordion :items="systemRoleItems" multiple>
            <template #trigger="{ item }">
              <OuiFlex align="center" gap="sm">
                <component v-if="item.icon" :is="item.icon" class="h-4 w-4" />
                <span>{{ item.label }}</span>
              </OuiFlex>
            </template>
            <template #content="{ item }">
              <div v-html="item.content"></div>
            </template>
          </OuiAccordion>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Custom Roles</OuiText>
        <OuiText size="sm" color="secondary">
          Create organization-specific roles with custom permissions
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Custom roles allow you to create roles with specific permissions tailored to your organization's needs.
            You can create roles like "Deployment Manager", "Production Admin", or "Read-Only Analyst".
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Creating Custom Roles</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">1. Navigate to <strong>Admin > Roles</strong></OuiText>
                <OuiText size="sm" color="secondary">2. Click <strong>Create Role</strong></OuiText>
                <OuiText size="sm" color="secondary">3. Enter role name and description</OuiText>
                <OuiText size="sm" color="secondary">4. Select permissions from the permission tree</OuiText>
                <OuiText size="sm" color="secondary">5. Save the role</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Permission Selection</OuiText>
              <OuiText size="sm" color="secondary">
                When creating a custom role, you can select specific permissions or use wildcards:
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • <strong>Specific permissions</strong>: <code class="text-xs">deployment.create</code>, <code class="text-xs">gameservers.read</code>
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Wildcard permissions</strong>: <code class="text-xs">deployment.*</code> (grants all deployment permissions)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Resource wildcards</strong>: <code class="text-xs">*</code> (all permissions, superadmin only)
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-warning/10 border border-warning/20">
            <OuiText size="sm" color="secondary">
              <strong>Note:</strong> You need <code class="text-xs">admin.roles.create</code> or <code class="text-xs">admin.roles.*</code> permission
              to create custom roles. System roles cannot be modified or deleted.
            </OuiText>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Role Bindings</OuiText>
        <OuiText size="sm" color="secondary">
          Assign roles to users with optional resource scoping
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Role bindings connect users to roles, optionally scoped to specific resources. This allows
            fine-grained access control where a user might have different permissions for different resources.
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Organization-Wide</OuiText>
                <OuiText size="sm" color="secondary">
                  A binding without resource scoping grants permissions organization-wide. The user
                  has the role's permissions for all resources in the organization.
                </OuiText>
                <OuiText size="xs" color="secondary" class="mt-2 italic">
                  Example: User has "Deployment Manager" role → can manage all deployments
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Resource-Scoped</OuiText>
                <OuiText size="sm" color="secondary">
                  A binding with resource scoping limits permissions to specific resources, resource types,
                  or environments. The user only has permissions for matching resources.
                </OuiText>
                <OuiText size="xs" color="secondary" class="mt-2 italic">
                  Example: User has "Production Manager" role binding scoped to "production" environment → has additional production-specific permissions
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Creating Role Bindings</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">1. Navigate to <strong>Admin > Bindings</strong></OuiText>
                <OuiText size="sm" color="secondary">2. Select a member and role</OuiText>
                <OuiText size="sm" color="secondary">3. (Optional) Select resource type and specific resource</OuiText>
                <OuiText size="sm" color="secondary">4. Click <strong>Bind</strong></OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Supported Resource Types</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">• <strong>Deployment</strong>: Scope to specific deployments</OuiText>
                <OuiText size="sm" color="secondary">• <strong>Environment</strong>: Scope to specific environments (applies to deployments)</OuiText>
                <OuiText size="sm" color="secondary">• <strong>VPS</strong>: Scope to specific VPS instances</OuiText>
                <OuiText size="sm" color="secondary">• <strong>Game Server</strong>: Scope to specific game servers</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Permission Format</OuiText>
        <OuiText size="sm" color="secondary">
          Understanding permission strings
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Permissions follow the format: <code class="text-sm">resource.action</code>
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Resource Types</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">• <code>deployment</code> - Deployments</OuiText>
                  <OuiText size="xs" color="secondary">• <code>gameservers</code> - Game servers</OuiText>
                  <OuiText size="xs" color="secondary">• <code>vps</code> - VPS instances</OuiText>
                  <OuiText size="xs" color="secondary">• <code>organization</code> - Organization settings</OuiText>
                  <OuiText size="xs" color="secondary">• <code>admin</code> - Admin operations</OuiText>
                  <OuiText size="xs" color="secondary">• <code>superadmin</code> - Superadmin operations</OuiText>
                </OuiStack>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Actions</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">• <code>read</code> - View/list resources</OuiText>
                  <OuiText size="xs" color="secondary">• <code>create</code> - Create new resources</OuiText>
                  <OuiText size="xs" color="secondary">• <code>update</code> - Modify resources</OuiText>
                  <OuiText size="xs" color="secondary">• <code>delete</code> - Delete resources</OuiText>
                  <OuiText size="xs" color="secondary">• <code>start</code>, <code>stop</code>, <code>restart</code> - Control resources</OuiText>
                  <OuiText size="xs" color="secondary">• <code>*</code> - Wildcard (all actions)</OuiText>
                </OuiStack>
              </OuiStack>
            </OuiBox>
          </OuiGrid>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Examples</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • <code class="text-xs">deployment.read</code> - View deployments
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <code class="text-xs">deployment.create</code> - Create deployments
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <code class="text-xs">deployment.*</code> - All deployment permissions
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <code class="text-xs">admin.roles.read</code> - View roles
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <code class="text-xs">admin.bindings.create</code> - Create role bindings
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Wildcard Permissions</OuiText>
        <OuiText size="sm" color="secondary">
          Using wildcards to grant all permissions for a resource type
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Wildcard permissions use <code>*</code> to grant all permissions for a resource type.
            This simplifies permission management for roles that need full access.
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Resource Wildcards</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">
                    • <code>deployment.*</code> - All deployment permissions
                  </OuiText>
                  <OuiText size="xs" color="secondary">
                    • <code>gameservers.*</code> - All game server permissions
                  </OuiText>
                  <OuiText size="xs" color="secondary">
                    • <code>vps.*</code> - All VPS permissions
                  </OuiText>
                  <OuiText size="xs" color="secondary">
                    • <code>admin.*</code> - All admin permissions
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Global Wildcard</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">
                    • <code>*</code> - All permissions (superadmin only)
                  </OuiText>
                </OuiStack>
                <OuiText size="xs" color="secondary" class="mt-2">
                  The global wildcard grants access to everything and is only available to superadmins.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>

          <OuiBox p="md" rounded="lg" class="bg-warning/10 border border-warning/20">
            <OuiText size="sm" color="secondary">
              <strong>Best Practice:</strong> Use wildcards sparingly. Prefer specific permissions
              unless you truly need all actions for a resource type. This follows the principle of
              least privilege and makes it easier to audit and understand permissions.
            </OuiText>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">How Permissions Work</OuiText>
        <OuiText size="sm" color="secondary">
          Understanding when and how your permissions are evaluated
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            When you try to perform an action (like creating a deployment or viewing a game server),
            the system checks your permissions in this order:
          </OuiText>

          <OuiStack gap="sm" class="pl-4">
            <OuiText size="sm" color="secondary">
              <strong>1. Your Assigned Role</strong> - The system first checks the role you were assigned
              when you joined the organization (Owner, Admin, Member, Viewer, or a custom role).
            </OuiText>
            <OuiText size="sm" color="secondary" class="pl-4">
              • System roles (Owner, Admin, Member, Viewer, None) have predefined permissions
            </OuiText>
            <OuiText size="sm" color="secondary" class="pl-4">
              • Custom roles have the specific permissions that were selected when the role was created
            </OuiText>
            <OuiText size="sm" color="secondary">
              <strong>2. Your Role Bindings</strong> - The system then checks any additional role bindings
              that were created for you.
            </OuiText>
            <OuiText size="sm" color="secondary" class="pl-4">
              • These can grant you additional permissions beyond your assigned role
            </OuiText>
            <OuiText size="sm" color="secondary" class="pl-4">
              • These additional permissions can be scoped to specific resources (like only certain deployments
              or environments)
            </OuiText>
            <OuiText size="sm" color="secondary" class="pl-4">
              • Note: Role bindings add permissions, they don't remove or limit your base role permissions
            </OuiText>
            <OuiText size="sm" color="secondary">
              <strong>3. Permission Matching</strong> - The system checks if you have the exact permission
              needed, or if you have a wildcard permission that covers it (like <code class="text-xs">deployment.*</code>).
            </OuiText>
            <OuiText size="sm" color="secondary">
              <strong>4. Access Decision</strong> - Based on the checks above, the system either allows
              or denies your action.
            </OuiText>
          </OuiStack>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Understanding Permission Priority</OuiText>
              <OuiText size="sm" color="secondary">
                Your assigned role is checked first. If it grants the permission you need, you'll have access
                organization-wide. Role bindings are then checked, which can add additional permissions that
                are scoped to specific resources. For example, if you're assigned the "Member" role (which
                includes <code class="text-xs">deployment.read</code>, <code class="text-xs">deployment.create</code>, etc. organization-wide)
                but have a role binding that grants you "Deployment Manager" permissions (including
                <code class="text-xs">deployment.delete</code>) only for production deployments, you'll have:
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • Your Member role permissions (read, create, update, etc.) everywhere in the organization
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Additional permissions from the role binding (like delete) only in production
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Example: Creating a Deployment</OuiText>
              <OuiText size="sm" color="secondary">
                When you try to create a deployment, the system checks:
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Does your assigned role (e.g., "Member") include <code class="text-xs">deployment.create</code>? ✓ Yes
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Are there any role bindings that grant or restrict this permission? Checked
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. If you have the permission, access is granted and you can create the deployment
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Troubleshooting</OuiText>
        <OuiText size="sm" color="secondary">
          Common permission issues and solutions
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiAccordion :items="troubleshootingItems" multiple>
          <template #trigger="{ item }">
            <OuiFlex align="center" gap="sm">
              <component v-if="item.icon" :is="item.icon" class="h-4 w-4" />
              <span>{{ item.label }}</span>
            </OuiFlex>
          </template>
          <template #content="{ item }">
            <div v-html="item.content"></div>
          </template>
        </OuiAccordion>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  ShieldCheckIcon,
  UserGroupIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  QuestionMarkCircleIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const systemRoleItems = [
  {
    value: "owner",
    label: "Owner",
    icon: ShieldCheckIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3"><strong>Full access to everything</strong></p>
      <p class="text-sm text-secondary mb-2">Permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><code class="text-xs">deployment.*</code> - All deployment permissions</li>
        <li><code class="text-xs">gameservers.*</code> - All game server permissions</li>
        <li><code class="text-xs">vps.*</code> - All VPS permissions</li>
        <li><code class="text-xs">organization.*</code> - All organization permissions</li>
        <li><code class="text-xs">admin.*</code> - All admin permissions</li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Capabilities:</strong> Can manage all resources, billing, delete organization, assign any role.
        There must always be at least one owner in an organization.
      </p>
    `,
  },
  {
    value: "admin",
    label: "Admin",
    icon: ShieldCheckIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3"><strong>Full resource management, limited organization control</strong></p>
      <p class="text-sm text-secondary mb-2">Permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><code class="text-xs">deployment.*</code> - All deployment permissions</li>
        <li><code class="text-xs">gameservers.*</code> - All game server permissions</li>
        <li><code class="text-xs">vps.*</code> - All VPS permissions</li>
        <li><code class="text-xs">organization.read</code>, <code class="text-xs">organization.update</code> - Limited org access</li>
        <li><code class="text-xs">organization.members.*</code> - Full member management</li>
        <li><code class="text-xs">admin.*</code> - All admin permissions</li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Capabilities:</strong> Can manage all resources, manage members, but cannot delete organization.
      </p>
    `,
  },
  {
    value: "member",
    label: "Member",
    icon: UserGroupIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3"><strong>Create and manage resources, read-only organization access</strong></p>
      <p class="text-sm text-secondary mb-2">Permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><strong>Deployments:</strong> <code class="text-xs">read</code>, <code class="text-xs">create</code>, <code class="text-xs">update</code>, <code class="text-xs">start</code>, <code class="text-xs">stop</code>, <code class="text-xs">restart</code>, <code class="text-xs">scale</code>, <code class="text-xs">logs</code> (no delete)</li>
        <li><strong>Game Servers:</strong> <code class="text-xs">read</code>, <code class="text-xs">create</code>, <code class="text-xs">update</code>, <code class="text-xs">start</code>, <code class="text-xs">stop</code>, <code class="text-xs">restart</code> (no delete)</li>
        <li><strong>VPS:</strong> <code class="text-xs">read</code>, <code class="text-xs">create</code>, <code class="text-xs">update</code>, <code class="text-xs">start</code>, <code class="text-xs">stop</code>, <code class="text-xs">reboot</code> (no delete or manage)</li>
        <li><strong>Organization:</strong> <code class="text-xs">read</code>, <code class="text-xs">members.read</code></li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Capabilities:</strong> Can create and manage resources but cannot delete them or manage organization settings.
      </p>
    `,
  },
  {
    value: "viewer",
    label: "Viewer",
    icon: UserGroupIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3"><strong>Read-only access</strong></p>
      <p class="text-sm text-secondary mb-2">Permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><code class="text-xs">deployment.read</code>, <code class="text-xs">deployment.logs</code></li>
        <li><code class="text-xs">gameservers.read</code></li>
        <li><code class="text-xs">vps.read</code></li>
        <li><code class="text-xs">organization.read</code>, <code class="text-xs">organization.members.read</code></li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Capabilities:</strong> Can view resources, metrics, and logs but cannot create, update, or delete anything.
      </p>
    `,
  },
  {
    value: "none",
    label: "None",
    icon: ShieldCheckIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3"><strong>No permissions - must use role bindings</strong></p>
      <p class="text-sm text-secondary mb-2">Permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li>No permissions</li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Capabilities:</strong> Users with this role must have permissions granted via role bindings.
        This is useful for fine-grained permission control where you want to explicitly grant permissions
        through bindings rather than relying on a default role.
      </p>
    `,
  },
];

const troubleshootingItems = [
  {
    value: "permission-denied",
    label: "I'm getting permission denied errors",
    icon: ExclamationTriangleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">If you're getting permission denied errors, check:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary pl-4">
        <li><strong>Your role:</strong> Verify your role in the organization (Owner, Admin, Member, Viewer, or custom)</li>
        <li><strong>Role permissions:</strong> If using a custom role, verify it has the required permissions</li>
        <li><strong>Role bindings:</strong> Verify you have role bindings with the required permissions</li>
        <li><strong>Resource scoping:</strong> If permissions are scoped to specific resources, ensure you're accessing the correct resource</li>
        <li><strong>Admin permissions:</strong> Some actions require <code class="text-xs">admin.*</code> permissions (e.g., creating roles, bindings)</li>
      </ol>
      <p class="text-sm text-secondary mt-3">
        <strong>Solution:</strong> Contact your organization admin to assign the appropriate permissions or role.
        Common permissions include: <code class="text-xs">deployment.read</code>, <code class="text-xs">deployment.create</code>,
        <code class="text-xs">gameservers.read</code>, <code class="text-xs">admin.roles.read</code>.
      </p>
    `,
  },
  {
    value: "cant-create",
    label: "I can't create resources",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">To create resources, you need the appropriate <code class="text-xs">create</code> permission:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><strong>Deployments:</strong> <code class="text-xs">deployment.create</code> or <code class="text-xs">deployment.*</code></li>
        <li><strong>Game Servers:</strong> <code class="text-xs">gameservers.create</code> or <code class="text-xs">gameservers.*</code></li>
        <li><strong>VPS:</strong> <code class="text-xs">vps.create</code> or <code class="text-xs">vps.*</code></li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Solution:</strong> Ask your organization admin to grant you the <code class="text-xs">create</code> permission
        for the resource type you need, or assign you a role that includes it (e.g., Member, Admin, or a custom role).
      </p>
    `,
  },
  {
    value: "cant-see-resources",
    label: "I can't see other users' resources",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">To see all resources in an organization, you need:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><strong>Read permission:</strong> <code class="text-xs">resource.read</code> or <code class="text-xs">resource.*</code></li>
        <li><strong>Organization-wide access:</strong> The permission from your assigned role grants org-wide access</li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Note:</strong> Your assigned role permissions are always organization-wide. Role bindings can add
        additional scoped permissions, but they don't replace your base role permissions. If you only have
        scoped permissions from role bindings (and no base role permissions), you'll only see those specific resources.
      </p>
      <p class="text-sm text-secondary mt-2">
        <strong>Solution:</strong> Ask your organization admin to grant you organization-wide <code class="text-xs">read</code>
        permissions, or assign you a role with broader access (e.g., Member, Admin, or Owner).
      </p>
    `,
  },
  {
    value: "cant-manage-roles",
    label: "I can't manage roles or bindings",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">To manage roles and bindings, you need admin permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary pl-4">
        <li><strong>View roles:</strong> <code class="text-xs">admin.roles.read</code> or <code class="text-xs">admin.roles.*</code></li>
        <li><strong>Create/update roles:</strong> <code class="text-xs">admin.roles.create</code>, <code class="text-xs">admin.roles.update</code>, or <code class="text-xs">admin.roles.*</code></li>
        <li><strong>View bindings:</strong> <code class="text-xs">admin.bindings.read</code> or <code class="text-xs">admin.bindings.*</code></li>
        <li><strong>Create bindings:</strong> <code class="text-xs">admin.bindings.create</code> or <code class="text-xs">admin.bindings.*</code></li>
        <li><strong>All admin operations:</strong> <code class="text-xs">admin.*</code></li>
      </ul>
      <p class="text-sm text-secondary mt-3">
        <strong>Solution:</strong> Ask your organization admin to grant you <code class="text-xs">admin.*</code> permissions,
        or assign you the Admin or Owner role, which includes these permissions.
      </p>
    `,
  },
];
</script>

