<template>
  <OuiStack gap="xl">
    <!-- Page Header -->
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Documentation
      </OuiText>
      <OuiText color="secondary" size="lg">
        Learn how to use Obiente Cloud Dashboard to manage your infrastructure, deployments, and resources.
      </OuiText>
    </OuiStack>

    <!-- Quick Start -->
    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Getting Started</OuiText>
        <OuiText size="sm" color="secondary">
          New to Obiente Cloud? Start here to get up and running quickly.
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Obiente Cloud Dashboard is your central hub for managing cloud infrastructure, 
            deployments, game servers, VPS instances, and databases. This guide will help 
            you navigate and utilize all the features available.
          </OuiText>
          <NuxtLink to="/docs/getting-started">
            <OuiButton color="primary" class="gap-2">
              <BookOpenIcon class="h-4 w-4" />
              Read Getting Started Guide
            </OuiButton>
          </NuxtLink>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Key Concepts -->
    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Key Concepts</OuiText>
        <OuiText size="sm" color="secondary">
          Understanding the fundamental concepts of Obiente Cloud
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/organizations')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <BuildingOfficeIcon class="h-5 w-5 text-primary" />
                <OuiText size="sm" weight="semibold" color="primary">Organizations</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Organizations are workspaces that group your resources, deployments, and team members. 
                You can switch between organizations using the switcher in the sidebar.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/deployments')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-5 w-5 text-success" />
                <OuiText size="sm" weight="semibold" color="primary">Deployments</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Deployments are your applications and services running on Obiente Cloud. 
                Each deployment can be configured with different environments (production, staging, development).
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/gameservers')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <CubeIcon class="h-5 w-5 text-warning" />
                <OuiText size="sm" weight="semibold" color="primary">Game Servers</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Dedicated infrastructure for hosting multiplayer game sessions. 
                Configure server types, player limits, and auto-scaling rules.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/vps')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <ServerIcon class="h-5 w-5 text-warning" />
                <OuiText size="sm" weight="semibold" color="primary">VPS Instances</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Virtual Private Servers give you full control over a dedicated virtual machine. 
                Choose from various instance sizes and operating systems.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/databases')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <CircleStackIcon class="h-5 w-5 text-secondary" />
                <OuiText size="sm" weight="semibold" color="primary">Databases</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Provision managed databases for PostgreSQL, MySQL, MongoDB, and Redis. 
                Databases are automatically backed up and scaled based on your needs.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/billing')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <CreditCardIcon class="h-5 w-5 text-secondary" />
                <OuiText size="sm" weight="semibold" color="primary">Billing</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Track your usage and costs in real-time. Monitor resource consumption, 
                set up billing alerts, and manage payment methods.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/permissions')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <ShieldCheckIcon class="h-5 w-5 text-primary" />
                <OuiText size="sm" weight="semibold" color="primary">Permissions & Access</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Understand how permissions work, manage roles and access control. 
                Learn about system roles, custom roles, and role bindings.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer" @click="navigateTo('/docs/self-hosting')">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="sm">
                <ServerIcon class="h-5 w-5 text-primary" />
                <OuiText size="sm" weight="semibold" color="primary">Self-Hosting</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Deploy and manage your own Obiente Cloud instance with DNS delegation support. 
                Maintain full control over your infrastructure while using centralized DNS.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <!-- Popular Guides -->
    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Popular Guides</OuiText>
        <OuiText size="sm" color="secondary">
          Most commonly referenced documentation
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <NuxtLink to="/docs/getting-started" class="docs-link-item">
            <OuiFlex align="center" justify="between" gap="md">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-4 w-4 text-primary" />
                <OuiText weight="medium">Getting Started</OuiText>
              </OuiFlex>
              <ArrowRightIcon class="h-4 w-4 text-secondary" />
            </OuiFlex>
          </NuxtLink>
          <NuxtLink to="/docs/deployments" class="docs-link-item">
            <OuiFlex align="center" justify="between" gap="md">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-4 w-4 text-success" />
                <OuiText weight="medium">Deploying Applications</OuiText>
              </OuiFlex>
              <ArrowRightIcon class="h-4 w-4 text-secondary" />
            </OuiFlex>
          </NuxtLink>
          <NuxtLink to="/docs/troubleshooting" class="docs-link-item">
            <OuiFlex align="center" justify="between" gap="md">
              <OuiFlex align="center" gap="sm">
                <QuestionMarkCircleIcon class="h-4 w-4 text-warning" />
                <OuiText weight="medium">Troubleshooting</OuiText>
              </OuiFlex>
              <ArrowRightIcon class="h-4 w-4 text-secondary" />
            </OuiFlex>
          </NuxtLink>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  BookOpenIcon,
  RocketLaunchIcon,
  ServerIcon,
  CircleStackIcon,
  CreditCardIcon,
  BuildingOfficeIcon,
  CubeIcon,
  ShieldCheckIcon,
  QuestionMarkCircleIcon,
  ArrowRightIcon,
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
