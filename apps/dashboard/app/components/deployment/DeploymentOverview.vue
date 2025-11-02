<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <!-- Stats Grid -->
      <OuiGrid cols="1" :cols-md="2" gap="md">
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Domain</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <Icon name="uil:globe" class="h-4 w-4 text-secondary" />
            <OuiText size="sm" weight="medium">{{ deployment.domain }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Framework</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <CodeBracketIcon class="h-4 w-4 text-primary" />
            <OuiText size="sm" weight="medium">{{
              getTypeLabel((deployment as any).type)
            }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Environment</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <CpuChipIcon class="h-4 w-4 text-secondary" />
            <OuiText size="sm" weight="medium">{{
              getEnvironmentLabel(deployment.environment)
            }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Build Time</OuiText
          >
          <OuiText size="lg" weight="bold">{{ deployment.buildTime }}s</OuiText>
        </OuiBox>
      </OuiGrid>

    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { CodeBracketIcon, CpuChipIcon } from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import { DeploymentType, Environment as EnvEnum } from "@obiente/proto";

interface Props {
  deployment: Deployment;
}

const props = defineProps<Props>();

const getTypeLabel = (t: DeploymentType | number | undefined) => {
  switch (t) {
    case DeploymentType.DOCKER:
      return "Docker";
    case DeploymentType.STATIC:
      return "Static Site";
    case DeploymentType.NODE:
      return "Node.js";
    case DeploymentType.GO:
      return "Go";
    default:
      return "Custom";
  }
};

const getEnvironmentLabel = (env: string | EnvEnum | number) => {
  if (typeof env === "number") {
    switch (env) {
      case EnvEnum.PRODUCTION:
        return "Production";
      case EnvEnum.STAGING:
        return "Staging";
      case EnvEnum.DEVELOPMENT:
        return "Development";
      default:
        return "Environment";
    }
  }
  return String(env);
};
</script>
