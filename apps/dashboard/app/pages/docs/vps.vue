<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        VPS Instances
      </OuiText>
      <OuiText color="secondary" size="lg">
        Provision and manage virtual private servers
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">VPS Instances</OuiText>
        <OuiText size="sm" color="secondary">
          Virtual Private Servers (VPS) are currently in development
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            VPS Instances will provide dedicated virtual machines with full control over the operating 
            system and software stack. VPS instances will feature pay-as-you-go pricing where you only 
            pay for actual CPU and memory usage, not idle time.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-warning/10 border border-warning/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-warning shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Coming Soon
                </OuiText>
                <OuiText size="sm" color="secondary">
                  VPS instances are currently in development. Check back soon for updates on when 
                  this feature will be available.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <OuiText>
            When available, VPS instances will support:
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Instance Sizes</OuiText>
                <OuiText size="sm" color="secondary">
                  Choose from various instance sizes (small, medium, large, xlarge) with different 
                  CPU, RAM, and storage configurations.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Operating Systems</OuiText>
                <OuiText size="sm" color="secondary">
                  Select from various Linux distributions (Ubuntu, Debian, etc.) and versions.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Pay-as-You-Go</OuiText>
                <OuiText size="sm" color="secondary">
                  Pay for actual CPU and memory usage, not idle time. More cost-effective than 
                  traditional VPS providers that charge full price regardless of utilization.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Full Control</OuiText>
                <OuiText size="sm" color="secondary">
                  Full root access, SSH access, web console, and the ability to install any software 
                  or configure the system as needed.
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
  ServerIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";

definePageMeta({
  layout: "docs",
});
</script>
