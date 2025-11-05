<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Troubleshooting & FAQ
      </OuiText>
      <OuiText color="secondary" size="lg">
        Common issues and solutions
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Troubleshooting & FAQ</OuiText>
        <OuiText size="sm" color="secondary">
          Common issues and solutions
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiAccordion
          :items="faqItems"
          multiple
        >
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
      <p class="text-sm text-secondary mb-3">Common reasons for deployment failures:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Build errors:</strong> Errors in your code, missing dependencies, or invalid build commands</li>
        <li><strong>Repository issues:</strong> Invalid repository URL, missing branch, or authentication problems</li>
        <li><strong>Build strategy mismatch:</strong> Selected build strategy doesn't match your codebase (e.g., using Dockerfile strategy without a Dockerfile)</li>
        <li><strong>Resource limits:</strong> Insufficient resources (CPU, memory) or quota limits exceeded</li>
        <li><strong>Invalid configuration:</strong> Missing or incorrect environment variables, invalid Docker Compose configuration</li>
        <li><strong>Network issues:</strong> Problems cloning repository or pulling Docker images</li>
      </ul>
      <p class="text-sm text-secondary mt-3"><strong>Solution:</strong> Check the build logs in the deployment details page (Build Logs tab) for specific error messages. The logs show each step of the build process and where it failed.</p>
    `,
  },
  {
    value: "deployment-status",
    label: "What do deployment statuses mean?",
    content: `
      <p class="text-sm text-secondary mb-3">Deployment statuses indicate the current state:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>CREATED:</strong> Deployment created but not yet built</li>
        <li><strong>BUILDING:</strong> Build is currently in progress</li>
        <li><strong>DEPLOYING:</strong> Build complete, containers are being deployed</li>
        <li><strong>RUNNING:</strong> Deployment is live and running</li>
        <li><strong>STOPPED:</strong> Deployment has been manually stopped</li>
        <li><strong>FAILED:</strong> Build or deployment failed - check logs for details</li>
      </ul>
      <p class="text-sm text-secondary mt-3">You can view detailed status information and container statistics in the deployment details page.</p>
    `,
  },
  {
    value: "gameserver-not-starting",
    label: "My game server won't start. What should I do?",
    content: `
      <p class="text-sm text-secondary mb-3">If your game server won't start:</p>
      <ol class="list-decimal list-inside space-y-1 text-sm text-secondary">
        <li>Check the server status - it may be in STARTING, STOPPING, or FAILED state</li>
        <li>View the logs tab to see error messages or startup issues</li>
        <li>Check resource configuration - ensure memory and CPU are sufficient for your game type</li>
        <li>Verify environment variables and configuration files are correct</li>
        <li>Try accessing the terminal - if the server is stopped, type "start" in the terminal</li>
        <li>Check if the Docker image is compatible with your game type</li>
      </ol>
      <p class="text-sm text-secondary mt-3"><strong>Terminal Access:</strong> If the server is stopped, you can access the terminal and type "start" to start it. This is useful for debugging startup issues.</p>
    `,
  },
  {
    value: "high-costs",
    label: "Why are my costs higher than expected?",
    content: `
      <p class="text-sm text-secondary mb-3">Costs may be higher due to:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>High CPU usage:</strong> CPU is billed based on actual utilization percentage × cores × time</li>
        <li><strong>Memory usage:</strong> Memory is billed based on actual memory consumption, not allocated memory</li>
        <li><strong>Long runtime:</strong> Resources running 24/7 will cost more than part-time usage</li>
        <li><strong>Multiple resources:</strong> Running multiple deployments or game servers simultaneously</li>
        <li><strong>Storage:</strong> Storage is billed monthly regardless of usage ($0.20/GB-month)</li>
        <li><strong>Bandwidth:</strong> Large data transfers (especially for game servers with many players)</li>
      </ul>
      <p class="text-sm text-secondary mt-3"><strong>Solution:</strong> Review the Resource Usage section on the dashboard to identify high-usage resources. Check individual deployment/game server metrics to see which resources are consuming the most CPU, memory, or bandwidth. Consider stopping unused resources or optimizing resource allocations.</p>
    `,
  },
  {
    value: "resource-quotas",
    label: "What are resource quotas and how do they work?",
    content: `
      <p class="text-sm text-secondary mb-3">Resource quotas limit the amount of resources you can use per month:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>CPU Core-Seconds Monthly:</strong> Maximum CPU core-seconds per month (calculated from CPU utilization × cores × seconds)</li>
        <li><strong>Memory Byte-Seconds Monthly:</strong> Maximum memory byte-seconds per month</li>
        <li><strong>Bandwidth Bytes Monthly:</strong> Maximum bandwidth bytes per month (inbound + outbound)</li>
        <li><strong>Storage Bytes:</strong> Maximum storage bytes (not time-based, billed monthly)</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Quotas are set per organization. If you reach a quota limit, new resources may not be able to start or you may need to stop existing resources. Contact support or your organization admin to increase quotas if needed.</p>
    `,
  },
  {
    value: "build-strategy",
    label: "Which build strategy should I use?",
    content: `
      <p class="text-sm text-secondary mb-3">Build strategies are typically auto-detected, but here's when to use each:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Railpack/Nixpacks:</strong> Use for most applications - automatically detects your language/framework and builds accordingly</li>
        <li><strong>Dockerfile:</strong> Use when you have a Dockerfile in your repository</li>
        <li><strong>Docker Compose:</strong> Use for multi-container applications (Plain Compose for YAML stored in database, Compose Repo for YAML from repository)</li>
        <li><strong>Static Site:</strong> Use for static websites (HTML, CSS, JS) with optional nginx configuration</li>
      </ul>
      <p class="text-sm text-secondary mt-3">If auto-detection doesn't work correctly, you can manually select a build strategy in the deployment settings.</p>
    `,
  },
  {
    value: "environment-variables",
    label: "How do environment variables work?",
    content: `
      <p class="text-sm text-secondary mb-3">Environment variables are configured per deployment:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Set environment variables in the deployment settings (Env tab)</li>
        <li>Variables are available to your application at runtime</li>
        <li>Variables can be different per environment (production, staging, development)</li>
        <li>Changes to environment variables may require a restart or redeploy to take effect</li>
        <li>For Docker Compose deployments, environment variables can be set per service</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Environment variables are stored securely and can be updated without redeploying your application (though restart may be required).</p>
    `,
  },
  {
    value: "logs-not-showing",
    label: "Why aren't my logs showing?",
    content: `
      <p class="text-sm text-secondary mb-3">Logs may not appear if:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>The deployment or game server is not running (logs only stream for running resources)</li>
        <li>The container hasn't started yet (wait for BUILDING/DEPLOYING to complete)</li>
        <li>Logs are being buffered (logs appear in real-time but may have a small delay)</li>
        <li>Your application isn't outputting to stdout/stderr (ensure logs go to standard output)</li>
      </ul>
      <p class="text-sm text-secondary mt-3"><strong>Solution:</strong> Ensure your resource is in RUNNING status. Logs stream in real-time and support ANSI color codes. If logs still don't appear, check that your application is configured to output logs to stdout/stderr.</p>
    `,
  },
  {
    value: "permissions-denied",
    label: "I'm getting permission denied errors. What does this mean?",
    content: `
      <p class="text-sm text-secondary mb-3">Permission denied errors occur when you don't have the required permissions:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Organization-level permissions:</strong> You need permissions like "deployments.create" or "gameservers.manage" at the organization level</li>
        <li><strong>Resource-level permissions:</strong> Some permissions may be scoped to specific resources</li>
        <li><strong>Role-based access:</strong> Your role (Owner, Admin, Member, Viewer) determines what you can do</li>
        <li><strong>Custom roles:</strong> Custom roles may have specific permission sets</li>
      </ul>
      <p class="text-sm text-secondary mt-3"><strong>Solution:</strong> Contact your organization admin to assign the appropriate permissions or role. Common permissions include: deployments.view, deployments.create, deployments.manage, gameservers.view, gameservers.create, gameservers.manage.</p>
    `,
  },
  {
    value: "contact-support",
    label: "How do I contact support?",
    content: `
      <p class="text-sm text-secondary mb-3">Get help through:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Support Section:</strong> Navigate to Support in the sidebar to create support tickets</li>
        <li><strong>Documentation:</strong> This documentation covers most common questions</li>
        <li><strong>Dashboard Issues:</strong> Check the dashboard for error messages and status indicators</li>
        <li><strong>Logs:</strong> Review build logs and application logs for detailed error information</li>
      </ul>
      <p class="text-sm text-secondary mt-3">For urgent issues or questions not covered in documentation, create a support ticket. Support tickets are typically responded to within 24 hours.</p>
    `,
  },
];
</script>
