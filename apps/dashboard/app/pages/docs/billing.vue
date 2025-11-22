<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Billing & Usage
      </OuiText>
      <OuiText color="secondary" size="lg">
        Understand your costs and usage metrics with pay-as-you-go pricing
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Pricing Model</OuiText>
        <OuiText size="sm" color="secondary">
          Pay-as-you-go pricing based on actual resource usage
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Obiente Cloud uses pay-as-you-go pricing where you only pay for actual resource usage. 
            This is perfect for game servers, VPS instances, and variable workloads where resources 
            may be idle or offline. Unlike traditional hosting providers that charge full price 
            regardless of usage, you only pay when resources are actually running.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Why Pay-as-You-Go?
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • <strong>Game servers:</strong> Low costs when idle or offline. Traditional hosting charges $20-40/month even when empty.
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>VPS instances:</strong> Pay for actual CPU and memory usage, not idle time.
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Development environments:</strong> Stop paying for resources that sit idle overnight or on weekends.
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Variable workloads:</strong> Scale costs automatically with demand - no over-provisioning required.
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiAccordion
            :items="billingAccordionItems"
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
        <OuiText as="h2" class="oui-card-title">Resource Limitations & Plans</OuiText>
        <OuiText size="sm" color="secondary">
          Understanding resource restrictions and automatic plan upgrades
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Organizations may have resource limitations based on their assigned plan. These limitations help ensure 
            fair usage and prevent abuse. Your plan determines the maximum resources you can use:
          </OuiText>

          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">Resource Limits Include:</OuiText>
            <OuiStack gap="xs" class="pl-4">
              <OuiText size="sm" color="secondary">
                • <strong>CPU Cores:</strong> Maximum number of CPU cores you can allocate across all deployments
              </OuiText>
              <OuiText size="sm" color="secondary">
                • <strong>Memory:</strong> Maximum memory (RAM) you can allocate across all deployments
              </OuiText>
              <OuiText size="sm" color="secondary">
                • <strong>Deployments:</strong> Maximum number of concurrent deployments
              </OuiText>
              <OuiText size="sm" color="secondary">
                • <strong>Bandwidth:</strong> Maximum monthly bandwidth (inbound + outbound) in bytes
              </OuiText>
              <OuiText size="sm" color="secondary">
                • <strong>Storage:</strong> Maximum storage capacity in bytes
              </OuiText>
            </OuiStack>
          </OuiStack>

          <OuiBox p="md" rounded="lg" class="bg-warning/10 border border-warning/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="warning">
                Why Do I Have Resource Restrictions?
              </OuiText>
              <OuiText size="sm" color="secondary">
                Resource limitations are in place to:
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • Prevent abuse and ensure fair usage across all users
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Provide a safety check for new accounts before granting unlimited resources
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Allow automatic upgrades based on payment history
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                How to Upgrade Your Plan
              </OuiText>
              <OuiText size="sm" color="secondary">
                Plans are automatically upgraded when you meet certain criteria:
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • <strong>Adding Credits:</strong> When you add credits to your account, you may automatically 
                  be upgraded to a higher tier plan if you meet the minimum payment threshold
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Making Payments:</strong> When you pay for usage (either through credits or direct 
                  billing), your total payment history is tracked. Once you reach a plan's minimum payment 
                  threshold, you're automatically upgraded
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • <strong>Safety Check:</strong> The automatic upgrade system ensures that users who have 
                  demonstrated they actually pay for usage (by reaching minimum payment thresholds) are granted 
                  higher resource limits
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiText size="sm" color="secondary">
            <strong>Note:</strong> If you need immediate access to higher resource limits, contact support. 
            {{ isSelfHosted ? 'Superadmins' : 'The Obiente Cloud Team' }} can also manually assign plans to organizations.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Monthly Free Credits</OuiText>
        <OuiText size="sm" color="secondary">
          Some plans include monthly free credits automatically granted to your account
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Certain plans may include monthly free credits that are automatically added to your account 
            balance on the first day of each month. These credits can be used just like purchased credits 
            to pay for resource usage.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                How Monthly Free Credits Work
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • Free credits are automatically granted on the 1st of each month
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • The amount depends on your current plan (set by The Obiente Cloud Team)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Credits appear in your credit transaction history with a note indicating they're monthly free credits
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Free credits are used first before purchased credits (FIFO)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • If you upgrade or change plans, you'll receive the new plan's monthly free credits going forward
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiText size="sm" color="secondary">
            You can view your monthly free credit amount in your plan information on the Billing page. 
            Check your credit transaction history to see when free credits were granted.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Resource Pricing</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Memory (RAM)</OuiText>
              <OuiText size="xl" weight="bold" color="primary">$0.00411 per GB-hour</OuiText>
              <OuiText size="xs" color="secondary">$3.00 per GB-month (24/7 usage)</OuiText>
              <OuiText size="sm" color="secondary" class="mt-2">
                Billed per byte-second of memory usage. Only charged when deployments are running.
                Based on actual memory consumption, not allocated memory.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">vCPU</OuiText>
              <OuiText size="xl" weight="bold" color="primary">$0.00274 per vCPU-hour</OuiText>
              <OuiText size="xs" color="secondary">$2.00 per vCPU-month (24/7 usage)</OuiText>
              <OuiText size="sm" color="secondary" class="mt-2">
                Billed per core-second of vCPU usage. Calculated based on actual CPU utilization 
                percentage. If you use 50% of 1 core for 1 hour, you pay for 0.5 core-hours.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Bandwidth</OuiText>
              <OuiText size="xl" weight="bold" color="primary">$0.01 per GB</OuiText>
              <OuiText size="xs" color="secondary">Transferred data (inbound + outbound)</OuiText>
              <OuiText size="sm" color="secondary" class="mt-2">
                Billed per gigabyte of data transferred. Includes both inbound and outbound traffic. 
                Bandwidth costs are typically minimal for most applications.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Storage</OuiText>
              <OuiText size="xl" weight="bold" color="primary">$0.20 per GB-month</OuiText>
              <OuiText size="xs" color="secondary">Persistent storage</OuiText>
              <OuiText size="sm" color="secondary" class="mt-2">
                Billed per gigabyte of storage used per month. Storage includes persistent volumes, 
                database storage, and deployment storage. Billed monthly regardless of usage.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Usage Metrics</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Usage is tracked and aggregated for billing purposes. Metrics are collected in real-time 
            and aggregated hourly for efficient storage and querying.
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">CPU Core-Seconds</OuiText>
                <OuiText size="sm" color="secondary">
                  Total CPU usage calculated as (CPU percentage × cores × seconds). For example, 
                  50% usage of 2 cores for 1 hour = 1 core-hour = 3600 core-seconds.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Memory Byte-Seconds</OuiText>
                <OuiText size="sm" color="secondary">
                  Total memory usage calculated as (memory bytes × seconds). For example, 1GB 
                  running for 1 hour = 1 GB-hour = 3,600,000,000 byte-seconds.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Bandwidth Bytes</OuiText>
                <OuiText size="sm" color="secondary">
                  Total data transferred (inbound + outbound). Tracked incrementally and summed 
                  for billing. Includes all network traffic from your resources.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Storage Bytes</OuiText>
                <OuiText size="sm" color="secondary">
                  Current storage usage across all resources. Includes persistent volumes, database 
                  storage, and deployment storage. Billed monthly based on peak usage.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Quotas</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Organizations have resource quotas that limit the amount of resources you can use. 
            Quotas are set per organization and can be viewed in the Organizations section.
          </OuiText>
          <OuiStack gap="sm">
            <OuiText size="sm" color="secondary">
              • <strong>CPU Core-Seconds Monthly:</strong> Maximum CPU core-seconds per month
            </OuiText>
            <OuiText size="sm" color="secondary">
              • <strong>Memory Byte-Seconds Monthly:</strong> Maximum memory byte-seconds per month
            </OuiText>
            <OuiText size="sm" color="secondary">
              • <strong>Bandwidth Bytes Monthly:</strong> Maximum bandwidth bytes per month
            </OuiText>
            <OuiText size="sm" color="secondary">
              • <strong>Storage Bytes:</strong> Maximum storage bytes (not time-based)
            </OuiText>
          </OuiStack>
          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20 mt-2">
            <OuiText size="sm" color="secondary">
              If you need to increase your quotas, contact support or check with your organization admin.
            </OuiText>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
  import {
    CreditCardIcon,
    InformationCircleIcon,
    ChartBarIcon,
  } from "@heroicons/vue/24/outline";
  import type { Component } from "vue";
  
  definePageMeta({
    layout: "docs",
  });
  
  // Redirect if billing is disabled
  const config = useConfig();
  await config.fetchConfig();
  if (config.billingEnabled.value !== true) {
    throw createError({
      statusCode: 404,
      statusMessage: "Billing is disabled",
    });
  }
  
  const isSelfHosted = computed(() => config.selfHosted.value === true);

const billingAccordionItems = [
  {
    value: "understanding-costs",
    label: "Understanding Costs",
    icon: CreditCardIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Costs are calculated based on actual resource usage:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>CPU:</strong> Core-seconds consumed (based on actual CPU utilization percentage)</li>
        <li><strong>Memory:</strong> Byte-seconds consumed (based on actual memory usage)</li>
        <li><strong>Bandwidth:</strong> Data transferred (inbound + outbound) in bytes</li>
        <li><strong>Storage:</strong> Storage used in bytes (billed monthly)</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Costs are billed monthly and estimated based on current usage patterns. The dashboard shows estimated monthly costs based on current usage rates.</p>
    `,
  },
  {
    value: "cost-breakdown",
    label: "Cost Breakdown",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The dashboard provides a detailed cost breakdown:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Total Estimated Cost:</strong> Sum of all resource costs for the month</li>
        <li><strong>CPU Cost:</strong> Cost from CPU core-seconds usage</li>
        <li><strong>Memory Cost:</strong> Cost from memory byte-seconds usage</li>
        <li><strong>Bandwidth Cost:</strong> Cost from data transfer</li>
        <li><strong>Storage Cost:</strong> Cost from storage usage</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Costs are shown in cents and converted to dollars for display. View detailed breakdowns on the Billing page.</p>
    `,
  },
  {
    value: "viewing-usage",
    label: "Viewing Usage",
    icon: InformationCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">View usage and costs:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li>Navigate to Billing in the sidebar</li>
        <li>Select your organization</li>
        <li>View current month usage, estimated costs, and quota limits</li>
      </ol>
      <p class="text-sm text-secondary mt-3">Usage data is updated in real-time as resources are used. Historical usage data is available for past months.</p>
    `,
  },
];
</script>
