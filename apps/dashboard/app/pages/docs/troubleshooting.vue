<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Troubleshooting & FAQ
      </OuiText>
      <OuiText color="tertiary" size="lg">
        Common issues with deployments, GitHub connections, databases, billing,
        and day-to-day operations
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Most Common Issues</OuiText>
        <OuiText size="sm" color="tertiary">
          Start with the issue that most closely matches what you are seeing
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiAccordion :items="faqItems" multiple>
          <template #trigger="{ item }">
            <span>{{ item.label }}</span>
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
definePageMeta({
  layout: "docs",
});

const faqItems = [
  {
    value: "deployment-failed",
    label: "Why did my deployment fail?",
    content: `
      <p class="text-sm text-secondary mb-3">Most deployment failures come from one of these categories:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Build errors:</strong> missing dependencies, invalid commands, or application code issues</li>
        <li><strong>Repository issues:</strong> wrong repository URL, wrong branch, or GitHub access problems</li>
        <li><strong>Build strategy mismatch:</strong> the detected strategy does not match the repo contents</li>
        <li><strong>Runtime configuration:</strong> missing environment variables, bad ports, or broken Compose config</li>
        <li><strong>Quota or sizing problems:</strong> insufficient CPU, memory, or storage</li>
      </ul>
      <p class="text-sm text-secondary mt-3"><strong>Best first step:</strong> open the Builds tab and read the failing step, not just the final error badge.</p>
    `,
  },
  {
    value: "github-linking",
    label: "GitHub OAuth returns successfully, but no account appears",
    content: `
      <p class="text-sm text-secondary mb-3">This is usually a backend configuration issue, not a browser issue.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Verify the dashboard has a GitHub client ID and secret at runtime</li>
        <li>Verify the GitHub OAuth callback URL exactly matches <code>/api/github/callback</code> on your public dashboard host</li>
        <li>Verify <code>auth-service</code> can encrypt stored GitHub tokens</li>
        <li>Redeploy <code>dashboard</code> and <code>auth-service</code> after changing any GitHub or encryption secret</li>
      </ul>
      <p class="text-sm text-secondary mt-3">If the callback lands back on Settings but the list is still empty, inspect the auth-service logs.</p>
    `,
  },
  {
    value: "compose-enotfound-cache",
    label:
      "My Compose app logs ENOTFOUND for cache, redis, or another internal service",
    content: `
      <p class="text-sm text-secondary mb-3">That hostname should usually match a Compose service name from the same stack.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>If your Compose file defines <code>cache:</code>, use <code>redis://cache:6379</code></li>
        <li>If the service is actually named <code>redis:</code>, use <code>redis://redis:6379</code></li>
        <li>Do not use managed Obiente database hostnames as if they were internal Compose service names</li>
        <li><code>depends_on</code> helps ordering, but it does not guarantee readiness</li>
      </ul>
    `,
  },
  {
    value: "database-hostname",
    label: "My managed database hostname does not resolve",
    content: `
      <p class="text-sm text-secondary mb-3">If an app cannot resolve <code>db-xxxxxxxxxxxxxxxx.my.obiente.cloud</code>, the issue is usually DNS or platform health.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Make sure the database is running</li>
        <li>Verify Obiente DNS is healthy</li>
        <li>For self-hosted setups, verify delegation for <code>my.obiente.cloud</code></li>
        <li>Test name resolution from inside the failing container</li>
      </ul>
      <p class="text-sm text-secondary mt-3">If direct queries to your DNS node work but public <code>dig</code> does not, the issue is delegation or recursive DNS, not the app itself.</p>
    `,
  },
  {
    value: "database-password-reset",
    label: "I reset a database password, but the app still cannot connect",
    content: `
      <p class="text-sm text-secondary mb-3">A password reset only changes the database credential. It does not update your applications automatically.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Save the new password immediately because it is shown once</li>
        <li>Update the deployment env vars, Compose secret, or external client config that uses the old password</li>
        <li>Restart or redeploy the consuming application</li>
      </ul>
      <p class="text-sm text-secondary mt-3">PostgreSQL, MySQL, and MariaDB resets are the most reliable reset workflows today.</p>
    `,
  },
  {
    value: "logs-not-showing",
    label: "Why are logs missing or delayed?",
    content: `
      <p class="text-sm text-secondary mb-3">Log issues usually come from one of these cases:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>The resource is not yet running</li>
        <li>The app writes logs somewhere other than stdout or stderr</li>
        <li>You are looking at a build log when you actually need runtime logs, or vice versa</li>
        <li>The resource recently moved across instances and you need to reopen the stream</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Use Builds for build-time failures and Logs for runtime failures.</p>
    `,
  },
  {
    value: "high-costs",
    label: "Why are my costs higher than expected?",
    content: `
      <p class="text-sm text-secondary mb-3">Higher costs are usually explained by sustained runtime or unexpected resource usage.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>CPU:</strong> billed from actual utilization over time</li>
        <li><strong>Memory:</strong> billed from actual memory usage</li>
        <li><strong>Storage:</strong> billed monthly, even when the app is idle</li>
        <li><strong>Bandwidth:</strong> billed from transferred data</li>
        <li><strong>Multiple resources:</strong> costs add up across deployments, databases, game servers, and VPS instances</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Check the Billing page for invoice history, estimated monthly cost, and per-resource breakdowns.</p>
    `,
  },
  {
    value: "permissions-denied",
    label: "What does permission denied mean in the dashboard?",
    content: `
      <p class="text-sm text-secondary mb-3">Permission errors mean your role does not include the action you attempted.</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Organization-level permissions control most create, edit, and delete actions</li>
        <li>Owners and admins can typically manage more billing and resource operations</li>
        <li>Some admin and superadmin pages require dedicated access even if you can read the main resource pages</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Ask an organization admin to review your role, role bindings, or custom permissions.</p>
    `,
  },
];
</script>
