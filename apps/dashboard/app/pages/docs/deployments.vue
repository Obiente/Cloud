<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Managing Deployments
      </OuiText>
      <OuiText color="tertiary" size="lg">
        Deploy applications from repositories, Dockerfiles, images, and Compose
        stacks
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="tertiary">
          Deployments let you run applications, services, and static sites on
          Obiente Cloud
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Deployments are the main application unit in Obiente Cloud. You can
            build from GitHub, deploy directly from Docker-oriented sources, and
            manage routing, logs, metrics, files, services, and environment
            variables from one place.
          </OuiText>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-success/10 border border-success/20"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Recommended deployment flow
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="tertiary">
                  1. Create or select the deployment from the Deployments page.
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  2. Connect GitHub first if you want repository browsing and
                  auto-deploys.
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  3. Pick the repository, branch, or container source.
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  4. Review the detected build strategy and adjust it only if
                  detection is wrong.
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  5. Configure env vars, resource sizing, and routing.
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  6. Start the deployment and use Builds, Logs, and Metrics to
                  validate it.
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Compose-specific note
              </OuiText>
              <OuiText size="sm" color="tertiary">
                In Docker Compose deployments, internal services should talk to
                each other by Compose service name. External managed resources
                such as Obiente databases should use the managed hostname shown
                in their connection details.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiAccordion :items="deploymentAccordionItems" multiple>
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
        <OuiText as="h2" class="oui-card-title">Deployment Features</OuiText>
        <OuiText size="sm" color="tertiary">
          Comprehensive tools for managing your deployments
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Build History</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Track every build with detailed logs, build status, durations,
                and deployment outcomes.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Real-time Metrics</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Monitor CPU, memory, network, and disk usage in real-time. View
                historical metrics and track resource consumption over time.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Logs & Terminal</OuiText
              >
              <OuiText size="sm" color="tertiary">
                View application logs with real-time streaming and open an
                interactive terminal for live debugging.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >File Management</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Browse, upload, edit, and delete files directly through the
                dashboard. Edit configuration files and manage application
                assets without SSH access.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Environment Variables</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Manage environment variables securely and keep repository,
                runtime, and secret configuration in one place.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Docker Compose Support</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Deploy multi-container applications using Docker Compose, with
                service-level logs and routing support.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Custom Domains & Routing</OuiText
              >
              <OuiText size="sm" color="tertiary">
                Configure default `my.obiente.cloud` hostnames, add custom
                domains, and manage path-based routing.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Service Management</OuiText
              >
              <OuiText size="sm" color="tertiary">
                For Docker Compose deployments, manage individual services, view
                service logs, restart services independently, and monitor
                service health.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Deployment Statuses</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-secondary" />
              <OuiText size="sm" weight="medium" color="primary"
                >CREATED</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Deployment created but not yet built</OuiText
              >
            </OuiFlex>
          </OuiBox>
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-warning" />
              <OuiText size="sm" weight="medium" color="primary"
                >BUILDING</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Build is in progress</OuiText
              >
            </OuiFlex>
          </OuiBox>
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-warning" />
              <OuiText size="sm" weight="medium" color="primary"
                >DEPLOYING</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Build complete, deploying containers</OuiText
              >
            </OuiFlex>
          </OuiBox>
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-success" />
              <OuiText size="sm" weight="medium" color="primary"
                >RUNNING</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Deployment is live and running</OuiText
              >
            </OuiFlex>
          </OuiBox>
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-secondary" />
              <OuiText size="sm" weight="medium" color="primary"
                >STOPPED</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Deployment has been stopped</OuiText
              >
            </OuiFlex>
          </OuiBox>
          <OuiBox
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-danger" />
              <OuiText size="sm" weight="medium" color="primary"
                >FAILED</OuiText
              >
              <OuiText size="sm" color="tertiary"
                >• Build or deployment failed</OuiText
              >
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  RocketLaunchIcon,
  Cog6ToothIcon,
  ChartBarIcon,
  CodeBracketIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const deploymentAccordionItems = [
  {
    value: "deployment-types",
    label: "Deployment Types",
    icon: RocketLaunchIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Obiente Cloud supports multiple deployment types that are auto-detected from your code:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Docker:</strong> Deploy from Docker images or Dockerfiles</li>
        <li><strong>Static:</strong> Host static websites (HTML, CSS, JS)</li>
        <li><strong>Node.js:</strong> Deploy Node.js applications</li>
        <li><strong>Go:</strong> Deploy Go applications</li>
        <li><strong>Python:</strong> Deploy Python applications</li>
        <li><strong>Ruby:</strong> Deploy Ruby/Rails applications</li>
        <li><strong>Rust:</strong> Deploy Rust applications</li>
        <li><strong>Java:</strong> Deploy Java applications</li>
        <li><strong>PHP:</strong> Deploy PHP applications</li>
        <li><strong>Generic:</strong> Used when the runtime type cannot be detected</li>
      </ul>
      <p class="text-sm text-secondary mt-3">The deployment type is automatically detected based on your repository contents when you configure the repository URL.</p>
    `,
  },
  {
    value: "build-strategies",
    label: "Build Strategies",
    icon: Cog6ToothIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Different build strategies are available depending on your deployment type:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Railpack:</strong> Nixpacks variant optimized for Rails and other frameworks. Can build any language including Rails applications.</li>
        <li><strong>Nixpacks:</strong> Buildpacks that can automatically detect and build your application from various languages and frameworks.</li>
        <li><strong>Dockerfile:</strong> Build from a Dockerfile in your repository. You can specify a custom Dockerfile path.</li>
        <li><strong>Plain Compose:</strong> Deploy using Docker Compose YAML stored directly in the database (no repository needed).</li>
        <li><strong>Compose from Repository:</strong> Clone your repository and use a Docker Compose file from the repo.</li>
        <li><strong>Static Site:</strong> Host static websites with optional nginx configuration.</li>
      </ul>
      <p class="text-sm text-secondary mt-3">The build strategy is typically auto-detected based on your repository, but you can also configure it manually.</p>
    `,
  },
  {
    value: "environments",
    label: "Environments",
    icon: CodeBracketIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Deployments can be configured for different environments:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Production:</strong> Live, production deployments with production domains</li>
        <li><strong>Staging:</strong> Pre-production testing environment for QA and testing</li>
        <li><strong>Development:</strong> Development and testing environments</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Each environment can have different resource limits, environment variables, and configurations. You can filter deployments by environment in the deployments list.</p>
    `,
  },
  {
    value: "groups-labels",
    label: "Groups & Labels",
    icon: Cog6ToothIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Groups and labels help organize your deployments:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Add multiple groups/labels when creating a deployment (e.g., "frontend", "backend", "api")</li>
        <li>Filter deployments by group in the deployments list</li>
        <li>Groups are useful for organizing deployments by project, team, or service type</li>
        <li>Groups are stored as an array and can be updated after deployment creation</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Groups help you manage large numbers of deployments by categorizing them logically.</p>
    `,
  },
  {
    value: "deployment-actions",
    label: "Deployment Actions",
    icon: RocketLaunchIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Available actions for deployments:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Start:</strong> Start a stopped deployment</li>
        <li><strong>Stop:</strong> Stop a running deployment</li>
        <li><strong>Restart:</strong> Restart containers without rebuilding (faster than redeploy)</li>
        <li><strong>Redeploy:</strong> Trigger a new build and deployment</li>
        <li><strong>Delete:</strong> Permanently delete a deployment and its resources</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Actions are available from the deployment details page. Restart is useful when you only need to restart containers without rebuilding.</p>
    `,
  },
  {
    value: "monitoring",
    label: "Monitoring & Logs",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Each deployment provides comprehensive monitoring:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Real-time Metrics:</strong> CPU usage percentage, memory bytes, network RX/TX bytes, disk read/write bytes</li>
        <li><strong>Request Metrics:</strong> Request count and error count (if tracked via middleware)</li>
        <li><strong>Logs:</strong> Application logs with ANSI color support and real-time streaming</li>
        <li><strong>Build Logs:</strong> Detailed build logs showing each step of the build process</li>
        <li><strong>Historical Metrics:</strong> View metrics over time with hourly aggregation</li>
        <li><strong>Usage Tracking:</strong> Track CPU core-seconds, memory byte-seconds, bandwidth, and storage for billing</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Metrics are streamed in real-time and stored for historical analysis. Usage metrics are used for billing calculations.</p>
    `,
  },
];
</script>
