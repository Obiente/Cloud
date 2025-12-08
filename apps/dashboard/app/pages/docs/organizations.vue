<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Organizations & Teams
      </OuiText>
      <OuiText color="secondary" size="lg">
        Manage your organizations, team members, roles, and permissions
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="secondary">
          Organizations are workspaces that group resources, deployments, and team members
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Organizations are workspaces that group your resources, deployments, game servers, 
            and team members. You can belong to multiple organizations and switch between them 
            easily. Each organization has its own resources, billing, and permissions.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-info shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Personal Organization
                </OuiText>
                <OuiText size="sm" color="secondary">
                  When you sign up, you automatically receive a "Personal" organization. This 
                  is your default workspace where you can start creating resources immediately. 
                  You're the Owner of this organization.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Creating Additional Organizations
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Navigate to Organizations in the sidebar
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Click "Create Organization" or use the organization switcher in the sidebar header
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. Enter organization name and slug (URL-friendly identifier)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  4. Configure initial settings and invite team members
                </OuiText>
              </OuiStack>
              <OuiText size="sm" color="secondary" class="mt-2">
                Create additional organizations to separate projects, teams, or environments. 
                Each organization has its own resources, billing, and team members.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiAccordion
            :items="organizationAccordionItems"
            multiple
          >
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
        <OuiText as="h2" class="oui-card-title">Roles & Permissions</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Obiente Cloud uses a flexible role-based access control (RBAC) system with scoped permissions. 
            Roles can be assigned organization-wide or scoped to specific resources.
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Owner</OuiText>
                <OuiText size="sm" color="secondary">
                  Full access to all resources and settings. Can manage billing, delete the organization, 
                  and assign roles to other members.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Admin</OuiText>
                <OuiText size="sm" color="secondary">
                  Can manage all resources and projects within the organization. Can create, update, 
                  and delete deployments, game servers, and other resources. Cannot delete the organization.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Member</OuiText>
                <OuiText size="sm" color="secondary">
                  Can view and create resources. Can update resources they have access to. Cannot 
                  delete resources or manage organization settings.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Viewer</OuiText>
                <OuiText size="sm" color="secondary">
                  Read-only access. Can view resources, metrics, and logs but cannot create, update, 
                  or delete anything.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Scoped Permissions</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Permissions are scoped to resource types and actions. Common permissions include:
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Deployments</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">• deployments.read</OuiText>
                  <OuiText size="xs" color="secondary">• deployments.create</OuiText>
                  <OuiText size="xs" color="secondary">• deployments.manage</OuiText>
                </OuiStack>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Game Servers</OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="xs" color="secondary">• gameservers.read</OuiText>
                  <OuiText size="xs" color="secondary">• gameservers.create</OuiText>
                  <OuiText size="xs" color="secondary">• gameservers.manage</OuiText>
                </OuiStack>
              </OuiStack>
            </OuiBox>
          </OuiGrid>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiText size="sm" color="secondary">
              Permissions can be assigned organization-wide or scoped to specific resources. 
              For example, a user might have "deployments.read" for all deployments but 
              "deployments.manage" only for specific deployments in a project.
            </OuiText>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Organization Management</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Organizations have various settings and features:
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Resource Quotas</OuiText>
                <OuiText size="sm" color="secondary">
                  Set limits on CPU core-seconds, memory byte-seconds, bandwidth bytes, and storage 
                  bytes per month. Quotas help prevent unexpected costs.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Billing</OuiText>
                <OuiText size="sm" color="secondary">
                  Track usage and costs per organization. View detailed billing breakdowns, 
                  usage metrics, and estimated costs.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Member Management</OuiText>
                <OuiText size="sm" color="secondary">
                  Invite team members, assign roles, and manage permissions. View member activity 
                  and manage access to resources.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Custom Roles</OuiText>
                <OuiText size="sm" color="secondary">
                  Create custom roles with specific permissions. Assign roles organization-wide 
                  or scoped to specific resources using role bindings.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  BuildingOfficeIcon,
  UserGroupIcon,
  ShieldCheckIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const organizationAccordionItems = [
  {
    value: "personal-org",
    label: "Personal Organization",
    icon: BuildingOfficeIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">
        When you sign up for Obiente Cloud, you automatically receive a "Personal" organization. 
        This is your default workspace where you can immediately start creating deployments, game 
        servers, and other resources. You're automatically assigned the Owner role in this organization.
      </p>
      <p class="text-sm text-secondary mt-3">
        Your Personal organization is ready to use right away - no setup required! You can create 
        additional organizations when you need separate workspaces for teams or projects.
      </p>
    `,
  },
  {
    value: "creating-org",
    label: "Creating Additional Organizations",
    icon: BuildingOfficeIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">To create a new organization (in addition to your Personal one):</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li>Navigate to Organizations in the sidebar</li>
        <li>Click "Create Organization" or use the "+" button in the organization switcher</li>
        <li>Enter organization name (display name)</li>
        <li>Enter organization slug (URL-friendly identifier, must be unique)</li>
        <li>Configure initial settings</li>
        <li>Invite team members via email</li>
      </ol>
      <p class="text-sm text-secondary mt-3">Once created, you can switch to the new organization using the switcher in the sidebar header. Each organization maintains separate resources, billing, and team members.</p>
    `,
  },
  {
    value: "team-management",
    label: "Team Management",
    icon: UserGroupIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Manage team members and permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Invite Members:</strong> Invite team members via email address</li>
        <li><strong>Assign Roles:</strong> Assign default roles (Owner, Admin, Member, Viewer) or custom roles</li>
        <li><strong>Resource Scoping:</strong> Assign permissions scoped to specific resources or organization-wide</li>
        <li><strong>Role Bindings:</strong> Use role bindings to assign custom roles with specific permissions</li>
        <li><strong>Remove Members:</strong> Remove team members from the organization</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Roles can be assigned organization-wide or scoped to specific resources (deployments, game servers, etc.) using resource selectors.</p>
    `,
  },
  {
    value: "switching-orgs",
    label: "Switching Organizations",
    icon: BuildingOfficeIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Switch between organizations:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Use the organization switcher in the sidebar header (top of the sidebar)</li>
        <li>Select an organization from the dropdown</li>
        <li>All resources, deployments, and data are scoped to the selected organization</li>
        <li>Your permissions may differ between organizations</li>
        <li>The dashboard and all pages will show data for the selected organization</li>
      </ul>
      <p class="text-sm text-secondary mt-3">You can belong to multiple organizations and switch between them easily. Each organization maintains separate resources and billing.</p>
    `,
  },
  {
    value: "custom-roles",
    label: "Custom Roles & Bindings",
    icon: ShieldCheckIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Create custom roles and assign them via bindings:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Custom Roles:</strong> Create roles with specific permissions (e.g., "deployments.read", "gameservers.manage")</li>
        <li><strong>Role Bindings:</strong> Bind users to roles, optionally scoped to specific resources</li>
        <li><strong>Resource Selectors:</strong> Use selectors to scope roles to resources matching criteria (e.g., environment="production")</li>
        <li><strong>Scoped Permissions:</strong> Permissions can be organization-wide or limited to specific resources</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Custom roles allow fine-grained access control. For example, you can create a role that allows managing only production deployments.</p>
    `,
  },
];
</script>
