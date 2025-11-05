<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Self-Hosting Obiente Cloud
      </OuiText>
      <OuiText color="secondary" size="lg">
        Deploy and manage your own Obiente Cloud instance with DNS delegation support
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="secondary">
          Self-host Obiente Cloud on your own infrastructure while using the main DNS service
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Self-hosting allows you to deploy Obiente Cloud on your own infrastructure while 
            leveraging the production DNS service for unified domain resolution. This enables 
            you to maintain full control over your deployments and data while benefiting from 
            centralized DNS management.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Key Benefits
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  • Full control over your infrastructure and data
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Use <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">my.obiente.cloud</code> DNS without exposing port 53
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Automatic DNS record synchronization
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Secure API key authentication for DNS delegation
                </OuiText>
                <OuiText size="sm" color="secondary">
                  • Simple $2/month subscription for DNS delegation
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Prerequisites</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Before self-hosting Obiente Cloud, ensure you have:
          </OuiText>
          <OuiStack gap="xs" class="pl-4">
            <OuiText size="sm" color="secondary">
              • Docker and Docker Compose installed
            </OuiText>
            <OuiText size="sm" color="secondary">
              • A server with sufficient resources (minimum 2GB RAM, 2 CPU cores recommended)
            </OuiText>
            <OuiText size="sm" color="secondary">
              • Domain name configured (optional, for custom domains)
            </OuiText>
            <OuiText size="sm" color="secondary">
              • An Obiente Cloud account with an organization
            </OuiText>
            <OuiText size="sm" color="secondary">
              • DNS delegation subscription ($2/month) - Subscribe via the dashboard
            </OuiText>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Quick Start</OuiText>
        <OuiText size="sm" color="secondary">
          Get your self-hosted instance up and running in minutes
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Step 1: Subscribe to DNS Delegation
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Navigate to <NuxtLink to="/billing" class="text-primary hover:underline">Billing</NuxtLink> or <NuxtLink to="/self-host" class="text-primary hover:underline">Self-Hosted DNS</NuxtLink> in the dashboard
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Select your organization
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. Click "Subscribe ($2/month)" in the DNS Delegation section
                </OuiText>
                <OuiText size="sm" color="secondary">
                  4. Complete the Stripe checkout process
                </OuiText>
                <OuiText size="sm" color="secondary">
                  5. Your API key will be automatically created when the subscription becomes active
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Step 2: Retrieve Your API Key
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Navigate to <NuxtLink to="/self-host" class="text-primary hover:underline">Self-Hosted DNS</NuxtLink> in the dashboard
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Select your organization
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. If you don't see an API key yet, click "Create API Key"
                </OuiText>
                <OuiText size="sm" color="secondary">
                  4. Save the API key securely - it will not be shown again!
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Step 3: Configure Your Self-Hosted Instance
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Clone the Obiente Cloud repository
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Configure environment variables for DNS delegation
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. Start your services with Docker Compose
                </OuiText>
                <OuiText size="sm" color="secondary">
                  4. Verify DNS records are being pushed
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Configuration</OuiText>
        <OuiText size="sm" color="secondary">
          Configure your self-hosted instance for DNS delegation
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="lg">
          <OuiStack gap="md">
            <OuiText size="sm" weight="medium">Environment Variables</OuiText>
            <OuiBox p="md" class="bg-surface-base border border-border-muted rounded">
              <pre class="text-xs font-mono whitespace-pre-wrap"><code># Production API URL
DNS_DELEGATION_PRODUCTION_API_URL="https://api.obiente.cloud"

# API key (retrieved from dashboard)
DNS_DELEGATION_API_KEY="your-api-key-here"

# Optional: How often to push DNS records (default: 2m)
DNS_DELEGATION_PUSH_INTERVAL="2m"

# Optional: TTL for pushed DNS records (default: 300s = 5 minutes)
DNS_DELEGATION_TTL="300s"</code></pre>
            </OuiBox>
          </OuiStack>

          <OuiStack gap="md">
            <OuiText size="sm" weight="medium">Docker Compose Example</OuiText>
            <OuiBox p="md" class="bg-surface-base border border-border-muted rounded">
              <pre class="text-xs font-mono whitespace-pre-wrap"><code>services:
  api:
    image: obiente/cloud-api:latest
    environment:
      DNS_DELEGATION_PRODUCTION_API_URL: "https://api.obiente.cloud"
      DNS_DELEGATION_API_KEY: "your-api-key-here"
      DNS_DELEGATION_PUSH_INTERVAL: "2m"
      DNS_DELEGATION_TTL: "300s"
      # ... other environment variables</code></pre>
            </OuiBox>
          </OuiStack>

          <OuiAccordion
            :items="configAccordionItems"
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
        <OuiText as="h2" class="oui-card-title">DNS Delegation</OuiText>
        <OuiText size="sm" color="secondary">
          How DNS delegation works for self-hosted instances
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            DNS delegation allows your self-hosted Obiente Cloud instance to use the main 
            <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">my.obiente.cloud</code> DNS service 
            without exposing DNS port 53 locally. This enables unified DNS resolution across all deployments.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                How It Works
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Your self-hosted API periodically pushes DNS records to the production API
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Production API stores these records with a TTL (time-to-live)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. Production DNS server queries delegated records when local lookup fails
                </OuiText>
                <OuiText size="sm" color="secondary">
                  4. Records automatically expire if not refreshed within TTL
                </OuiText>
                <OuiText size="sm" color="secondary">
                  5. DNS queries resolve correctly using the delegated records
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiStack gap="sm">
            <OuiText size="sm" weight="medium">Benefits</OuiText>
            <OuiStack gap="xs" class="pl-4">
              <OuiText size="sm" color="secondary">
                • No need to expose DNS port 53 on your host
              </OuiText>
              <OuiText size="sm" color="secondary">
                • No nameserver configuration required
              </OuiText>
              <OuiText size="sm" color="secondary">
                • Automatic record expiration prevents stale DNS entries
              </OuiText>
              <OuiText size="sm" color="secondary">
                • Secure API key authentication
              </OuiText>
              <OuiText size="sm" color="secondary">
                • Unified DNS resolution across all deployments
              </OuiText>
            </OuiStack>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Verification & Testing</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiStack gap="md">
            <OuiText size="sm" weight="medium">Verify DNS Records Are Being Pushed</OuiText>
            <OuiText size="sm" color="secondary">
              Check your API logs to verify DNS records are being pushed:
            </OuiText>
            <OuiBox p="md" class="bg-surface-base border border-border-muted rounded">
              <pre class="text-xs font-mono whitespace-pre-wrap"><code>docker logs obiente-api | grep -i "dns pusher"

# You should see logs like:
# [DNS Pusher] Successfully pushed 5 DNS records</code></pre>
            </OuiBox>
          </OuiStack>

          <OuiStack gap="md">
            <OuiText size="sm" weight="medium">Test DNS Resolution</OuiText>
            <OuiText size="sm" color="secondary">
              Verify that your deployments resolve correctly via production DNS:
            </OuiText>
            <OuiBox p="md" class="bg-surface-base border border-border-muted rounded">
              <pre class="text-xs font-mono whitespace-pre-wrap"><code># Query DNS for your deployment
dig deploy-123.my.obiente.cloud

# Should return the IP from your self-hosted database</code></pre>
            </OuiBox>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Troubleshooting</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiAccordion
          :items="troubleshootingItems"
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
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Best Practices</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">API Key Security</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">• Never commit API keys to version control</OuiText>
                <OuiText size="sm" color="secondary">• Use environment variables or secrets management</OuiText>
                <OuiText size="sm" color="secondary">• Rotate API keys periodically</OuiText>
                <OuiText size="sm" color="secondary">• Revoke compromised keys immediately</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">TTL Configuration</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">• Default TTL: 300 seconds (5 minutes)</OuiText>
                <OuiText size="sm" color="secondary">• Push interval must be less than TTL</OuiText>
                <OuiText size="sm" color="secondary">• Recommended: Push interval < TTL / 2</OuiText>
                <OuiText size="sm" color="secondary">• Shorter TTL = faster expiration but more frequent pushes</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Monitoring</OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">• Monitor API logs for DNS pusher activity</OuiText>
                <OuiText size="sm" color="secondary">• Set up alerts for DNS push failures</OuiText>
                <OuiText size="sm" color="secondary">• Verify DNS resolution periodically</OuiText>
                <OuiText size="sm" color="secondary">• Keep subscription active to maintain DNS delegation</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
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
            Now that you have your self-hosted instance configured:
          </OuiText>
          <OuiStack gap="sm">
            <NuxtLink to="/docs/deployments">
              <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer">
                <OuiFlex align="center" gap="sm">
                  <RocketLaunchIcon class="h-5 w-5 text-primary" />
                  <OuiText size="sm" weight="semibold" color="primary">Learn about Deployments</OuiText>
                </OuiFlex>
              </OuiBox>
            </NuxtLink>
            <NuxtLink to="/docs/gameservers">
              <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer">
                <OuiFlex align="center" gap="sm">
                  <CubeIcon class="h-5 w-5 text-primary" />
                  <OuiText size="sm" weight="semibold" color="primary">Learn about Game Servers</OuiText>
                </OuiFlex>
              </OuiBox>
            </NuxtLink>
            <NuxtLink to="/self-host">
              <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted hover:ring-primary/50 transition-all cursor-pointer">
                <OuiFlex align="center" gap="sm">
                  <ServerIcon class="h-5 w-5 text-primary" />
                  <OuiText size="sm" weight="semibold" color="primary">Manage DNS Delegation</OuiText>
                </OuiFlex>
              </OuiBox>
            </NuxtLink>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  ServerIcon,
  Cog6ToothIcon,
  QuestionMarkCircleIcon,
  RocketLaunchIcon,
  CubeIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const configAccordionItems = [
  {
    value: "environment-variables",
    label: "Environment Variables",
    icon: Cog6ToothIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Configure DNS delegation using environment variables:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>DNS_DELEGATION_PRODUCTION_API_URL:</strong> URL of the production API (required)</li>
        <li><strong>DNS_DELEGATION_API_KEY:</strong> API key from the dashboard (required)</li>
        <li><strong>DNS_DELEGATION_PUSH_INTERVAL:</strong> How often to push DNS records (default: "2m")</li>
        <li><strong>DNS_DELEGATION_TTL:</strong> TTL for pushed DNS records (default: "300s")</li>
      </ul>
      <p class="text-sm text-secondary mt-3">All environment variables are optional except for <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_PRODUCTION_API_URL</code> and <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_API_KEY</code>.</p>
    `,
  },
  {
    value: "push-interval",
    label: "Push Interval Configuration",
    icon: Cog6ToothIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">The push interval determines how often DNS records are pushed to production:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Default:</strong> 2 minutes ("2m")</li>
        <li><strong>Format:</strong> Duration string (e.g., "30s", "1m", "5m")</li>
        <li><strong>Minimum:</strong> Should be less than TTL to prevent expiration</li>
        <li><strong>Recommended:</strong> Push interval < TTL / 2</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Shorter intervals ensure records are refreshed more frequently, while longer intervals reduce API calls but increase the risk of expiration.</p>
    `,
  },
  {
    value: "ttl-configuration",
    label: "TTL Configuration",
    icon: Cog6ToothIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">TTL (Time-To-Live) determines how long DNS records remain valid:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Default:</strong> 300 seconds (5 minutes)</li>
        <li><strong>Format:</strong> Duration string (e.g., "60s", "300s", "600s")</li>
        <li><strong>Expiration:</strong> Records expire if not refreshed within TTL</li>
        <li><strong>Cleanup:</strong> Expired records are automatically cleaned up</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Shorter TTL values provide faster expiration (useful for stopped deployments), while longer TTL values reduce the frequency of DNS queries but may leave stale records.</p>
    `,
  },
];

const troubleshootingItems = [
  {
    value: "dns-records-not-pushing",
    label: "DNS Records Not Pushing",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">If DNS records are not being pushed, check the following:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li><strong>Verify API Key:</strong> Check that <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_API_KEY</code> is set correctly</li>
        <li><strong>Check Production API URL:</strong> Verify <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_PRODUCTION_API_URL</code> is correct</li>
        <li><strong>Check API Logs:</strong> <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">docker logs obiente-api | grep -i "dns pusher"</code></li>
        <li><strong>Verify Subscription:</strong> Ensure your subscription is active in the dashboard</li>
        <li><strong>Check Network:</strong> Ensure your API can reach the production API URL</li>
      </ol>
    `,
  },
  {
    value: "dns-records-not-resolving",
    label: "DNS Records Not Resolving",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">If DNS records are not resolving:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li><strong>Check if Records Are Pushed:</strong> Verify records are being pushed successfully</li>
        <li><strong>Check Record Expiration:</strong> Ensure push interval < TTL</li>
        <li><strong>Verify Deployment Status:</strong> Ensure your deployment is running</li>
        <li><strong>Test DNS Query:</strong> Use <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">dig deploy-123.my.obiente.cloud</code></li>
        <li><strong>Check DNS Server Logs:</strong> Review production DNS server logs</li>
      </ol>
    `,
  },
  {
    value: "invalid-api-key",
    label: "Invalid API Key Error",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">If you receive "Invalid API key" errors:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li><strong>Verify API Key:</strong> Check that the API key matches the one in your dashboard</li>
        <li><strong>Check Subscription:</strong> Ensure your subscription is active</li>
        <li><strong>Check Key Status:</strong> Verify the API key hasn't been revoked</li>
        <li><strong>Create New Key:</strong> If revoked, create a new API key from the dashboard</li>
        <li><strong>Update Configuration:</strong> Update your environment variables with the new key</li>
      </ol>
    `,
  },
  {
    value: "records-expiring-too-quickly",
    label: "Records Expiring Too Quickly",
    icon: QuestionMarkCircleIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">If records expire too quickly:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li><strong>Increase TTL:</strong> Set <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_TTL="600s"</code> (10 minutes)</li>
        <li><strong>Decrease Push Interval:</strong> Set <code class="text-xs bg-surface-subtle px-1 py-0.5 rounded">DNS_DELEGATION_PUSH_INTERVAL="1m"</code></li>
        <li><strong>Ensure Push Interval < TTL:</strong> Verify the push interval is less than TTL</li>
        <li><strong>Monitor Logs:</strong> Check that DNS pusher is running regularly</li>
      </ol>
    `,
  },
];
</script>

