<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Dashboard Overview
      </OuiText>
      <OuiText color="secondary" size="lg">
        Understanding your dashboard metrics and KPIs
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Dashboard Overview</OuiText>
        <OuiText size="sm" color="secondary">
          Comprehensive overview of your cloud infrastructure, usage metrics, and resource health
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            The Dashboard provides a comprehensive overview of your cloud infrastructure, 
            including resource usage, deployment health, cost breakdown, and recent activity. 
            All metrics are scoped to your currently selected organization.
          </OuiText>

          <OuiAccordion
            :items="dashboardAccordionItems"
            multiple
            class="mt-4"
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
        <OuiText as="h2" class="oui-card-title">Real-time Updates</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            The dashboard automatically refreshes every 30 seconds to show the latest data. 
            You can also manually refresh using the refresh button in the header.
          </OuiText>
          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-info shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Data Refresh
                </OuiText>
                <OuiText size="sm" color="secondary">
                  Dashboard data refreshes automatically every 30 seconds. Metrics are collected 
                  in real-time from your running resources and aggregated for display.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  ChartBarIcon,
  ShieldCheckIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const dashboardAccordionItems = [
  {
    value: "kpi-overview",
    label: "KPI Cards",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The KPI cards at the top of the dashboard show:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Active Deployments:</strong> Total number of deployments in your organization, with a subtitle showing how many are currently running</li>
        <li><strong>VPS Instances:</strong> Number of virtual private servers provisioned (if VPS feature is enabled)</li>
        <li><strong>Game Servers:</strong> Total game server instances in your organization</li>
        <li><strong>Databases:</strong> Number of managed databases (if database feature is enabled)</li>
        <li><strong>Estimated Monthly Cost:</strong> Projected costs based on current usage patterns, calculated from CPU core-seconds, memory byte-seconds, bandwidth, and storage usage</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Click any card to navigate to the detailed view. Costs are estimated based on current usage rates projected over the month.</p>
    `,
  },
  {
    value: "resource-usage",
    label: "Resource Usage",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The Resource Usage section displays current month usage:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>CPU:</strong> Core-seconds consumed this month, formatted as core-hours for display</li>
        <li><strong>Memory:</strong> Average memory usage per hour (memory byte-seconds / 3600)</li>
        <li><strong>Bandwidth:</strong> Total data transferred (upload + download) in bytes</li>
        <li><strong>Storage:</strong> Total storage used across all resources in bytes</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Progress bars show usage relative to your quota limits. If a quota is set to 0 (unlimited), the progress bar shows 0%. Usage is tracked from deployment_usage_hourly aggregates for the current month.</p>
    `,
  },
  {
    value: "cost-breakdown",
    label: "Cost Breakdown",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The Cost Breakdown section shows estimated monthly costs:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Total Estimated:</strong> Sum of all resource costs for the month</li>
        <li><strong>CPU Cost:</strong> Calculated from CPU core-seconds usage</li>
        <li><strong>Memory Cost:</strong> Calculated from memory byte-seconds usage</li>
        <li><strong>Bandwidth Cost:</strong> Calculated from data transfer</li>
        <li><strong>Storage Cost:</strong> Calculated from storage usage</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Costs are shown in dollars and are estimated based on current usage patterns projected over the month. View detailed billing in the Organizations section.</p>
    `,
  },
  {
    value: "deployment-health",
    label: "Deployment Health",
    icon: ShieldCheckIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Monitor the health status of all your deployments:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Running:</strong> Deployments that are active and healthy (status: RUNNING)</li>
        <li><strong>Building:</strong> Deployments currently being built or updated (status: BUILDING)</li>
        <li><strong>Stopped:</strong> Deployments that have been manually stopped (status: STOPPED)</li>
        <li><strong>Errors:</strong> Deployments experiencing issues (status: FAILED)</li>
      </ul>
      <p class="text-sm text-secondary mt-3">The "Requires Attention" section highlights deployments that need action (ERROR, STOPPED, or BUILDING status). Click on any deployment to view details and take action.</p>
    `,
  },
  {
    value: "recent-deployments",
    label: "Recent Deployments",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The Recent Deployments section shows:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Up to 5 most recently deployed or updated deployments</li>
        <li>Deployment name, domain, status, and environment</li>
        <li>Deployment type (Docker, Static, Node.js, Go, Python, etc.)</li>
        <li>Relative time since last deployment or update</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Deployments are sorted by lastDeployedAt timestamp, falling back to createdAt if not deployed yet. Click any deployment to view its details page.</p>
    `,
  },
];
</script>
