<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Getting Started
      </OuiText>
      <OuiText color="secondary" size="lg">
        Welcome to Obiente Cloud - Get started with deploying and managing your applications
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Welcome to Obiente Cloud</OuiText>
        <OuiText size="sm" color="secondary">
          Get started with deploying and managing your applications
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Obiente Cloud Dashboard is your central hub for managing cloud infrastructure, 
            deployments, game servers, VPS instances, and databases. This guide will help 
            you navigate and utilize all the features available.
          </OuiText>
          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-primary shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Quick Start
                </OuiText>
                <OuiText size="sm" color="secondary">
                  When you sign up, you automatically receive a "Personal" organization. You 
                  can belong to multiple organizations and switch between them using the 
                  organization switcher in the sidebar header. Create additional organizations 
                  for teams or projects by navigating to Organizations in the sidebar.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Key Concepts</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <BuildingOfficeIcon class="h-5 w-5 text-primary" />
                <OuiText size="sm" weight="semibold" color="primary">Organizations</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Organizations are workspaces that group your resources, deployments, and team members. 
                You can switch between organizations using the switcher in the sidebar header. Each 
                organization has separate billing, quotas, and permissions.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-5 w-5 text-success" />
                <OuiText size="sm" weight="semibold" color="primary">Deployments</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Deployments are your applications and services running on Obiente Cloud. They support 
                multiple build strategies (Railpack, Nixpacks, Dockerfile, Docker Compose) and can be 
                configured with different environments (production, staging, development).
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <CubeIcon class="h-5 w-5 text-warning" />
                <OuiText size="sm" weight="semibold" color="primary">Game Servers</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Dedicated Docker containers for hosting multiplayer game sessions. Features pay-as-you-go 
                pricing where you only pay when servers are running. Supports Minecraft, Valheim, Rust, 
                CS2, and many other game types.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <CreditCardIcon class="h-5 w-5 text-secondary" />
                <OuiText size="sm" weight="semibold" color="primary">Pay-as-You-Go Pricing</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Only pay for actual resource usage. CPU core-seconds, memory byte-seconds, bandwidth, 
                and storage are tracked and billed based on real usage. Perfect for variable workloads 
                and resources that may be idle.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">First Steps</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Here's a typical workflow to get started:
          </OuiText>
          <OuiStack gap="sm">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">1. Your Personal Organization</OuiText>
                <OuiText size="sm" color="secondary">
                  When you sign up, you automatically receive a "Personal" organization. This 
                  organization is ready to use immediately - you can start creating deployments, 
                  game servers, and other resources right away. You can create additional 
                  organizations from the Organizations page if you need separate workspaces for 
                  teams or projects.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">2. Create Your First Deployment</OuiText>
                <OuiText size="sm" color="secondary">
                  Navigate to Deployments and click "New Deployment". Enter a name, select an environment, 
                  and optionally add groups/labels. You'll be redirected to configure the repository URL, 
                  build settings, and environment variables.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">3. Configure and Deploy</OuiText>
                <OuiText size="sm" color="secondary">
                  Configure your deployment settings, including repository URL, build commands, environment 
                  variables, and resource limits. Once configured, deploy and monitor the build process 
                  in real-time through the build logs.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">4. Monitor and Manage</OuiText>
                <OuiText size="sm" color="secondary">
                  Use the dashboard to monitor your deployments, view metrics, check logs, and manage 
                  resources. The dashboard shows real-time metrics, cost breakdowns, and deployment health.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Next Steps</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Now that you understand the basics, explore these guides:
          </OuiText>
          <OuiStack gap="sm">
            <NuxtLink to="/docs/deployments" class="docs-link-item">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-4 w-4 text-success" />
                <OuiText>Learn about Deployments</OuiText>
              </OuiFlex>
            </NuxtLink>
            <NuxtLink to="/docs/gameservers" class="docs-link-item">
              <OuiFlex align="center" gap="sm">
                <CubeIcon class="h-4 w-4 text-warning" />
                <OuiText>Set up Game Servers</OuiText>
              </OuiFlex>
            </NuxtLink>
            <NuxtLink to="/docs/dashboard" class="docs-link-item">
              <OuiFlex align="center" gap="sm">
                <HomeIcon class="h-4 w-4 text-primary" />
                <OuiText>Understand the Dashboard</OuiText>
              </OuiFlex>
            </NuxtLink>
            <NuxtLink to="/docs/organizations" class="docs-link-item">
              <OuiFlex align="center" gap="sm">
                <BuildingOfficeIcon class="h-4 w-4 text-primary" />
                <OuiText>Set up Organizations & Teams</OuiText>
              </OuiFlex>
            </NuxtLink>
            <NuxtLink to="/docs/billing" class="docs-link-item">
              <OuiFlex align="center" gap="sm">
                <CreditCardIcon class="h-4 w-4 text-secondary" />
                <OuiText>Understand Billing & Pricing</OuiText>
              </OuiFlex>
            </NuxtLink>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  InformationCircleIcon,
  BuildingOfficeIcon,
  RocketLaunchIcon,
  ServerIcon,
  CreditCardIcon,
  HomeIcon,
  CubeIcon,
} from "@heroicons/vue/24/outline";

definePageMeta({
  layout: "docs",
});
</script>

<style scoped>
.docs-link-item {
  display: block;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border-muted);
  text-decoration: none;
  transition: all 0.15s ease;
}

.docs-link-item:hover {
  background-color: var(--surface-muted);
  border-color: var(--primary);
}
</style>
