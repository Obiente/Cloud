<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        VPS Instances
      </OuiText>
      <OuiText color="secondary" size="lg">
        Provision and manage virtual private servers with full root access
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            VPS Instances provide dedicated virtual machines with full root access and complete control 
            over the operating system. VPS instances are provisioned via Proxmox and can be accessed 
            through web terminal or SSH proxy.
          </OuiText>

          <OuiText>
            <strong>Key Features:</strong>
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Instance Sizes</OuiText>
                <OuiText size="sm" color="secondary">
                  Choose from various instance sizes with different CPU, RAM, and storage configurations 
                  to match your workload requirements.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Operating Systems</OuiText>
                <OuiText size="sm" color="secondary">
                  Select from various Linux distributions including Ubuntu 22.04/24.04, Debian 12/13, 
                  Rocky Linux 9, and AlmaLinux 9.
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
                  Full root access, SSH access via proxy, web terminal, and the ability to install 
                  any software or configure the system as needed.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Creating a VPS Instance</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            To create a new VPS instance:
          </OuiText>

          <OuiStack gap="sm" class="list-decimal list-inside pl-4">
            <OuiText>
              Navigate to the <strong>VPS Instances</strong> page from the sidebar
            </OuiText>
            <OuiText>
              Click the <strong>Create VPS Instance</strong> button
            </OuiText>
            <OuiText>
              Fill in the required information:
              <ul class="list-disc list-inside ml-4 mt-2 space-y-1">
                <li><strong>Name:</strong> A descriptive name for your VPS instance</li>
                <li><strong>Description:</strong> Optional description of the instance's purpose</li>
                <li><strong>Region:</strong> Select the region where the VPS will be provisioned</li>
                <li><strong>Operating System:</strong> Choose from available Linux distributions</li>
                <li><strong>Instance Size:</strong> Select CPU, RAM, and storage configuration</li>
              </ul>
            </OuiText>
            <OuiText>
              Review the resource limits and click <strong>Create</strong>
            </OuiText>
          </OuiStack>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-info shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Resource Limits
                </OuiText>
                <OuiText size="sm" color="secondary">
                  The instance size you select sets the maximum resources (CPU, RAM, storage) your VPS 
                  can use. Billing is pay-as-you-go based on actual usage, not the maximum limits.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Accessing Your VPS</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            VPS instances can be accessed through multiple methods:
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Web Terminal</OuiText>
              <OuiText size="sm" color="secondary">
                Access your VPS directly from the browser using the built-in web terminal. This provides 
                a full terminal experience without requiring SSH client configuration.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">SSH Proxy</OuiText>
              <OuiText size="sm" color="secondary">
                Connect via SSH through a jump host proxy. This allows you to access your VPS without 
                requiring a dedicated public IP address, reducing costs. Use the connection instructions 
                provided in the VPS details page.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Managing VPS Instances</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Once your VPS is created, you can manage it through the dashboard:
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Lifecycle Operations</OuiText>
                <OuiText size="sm" color="secondary">
                  Start, stop, and reboot your VPS instances as needed. Status changes are reflected 
                  immediately in the dashboard.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Monitoring</OuiText>
                <OuiText size="sm" color="secondary">
                  View real-time metrics including CPU usage, memory consumption, disk I/O, and network 
                  traffic for your VPS instances.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Updates</OuiText>
                <OuiText size="sm" color="secondary">
                  Update VPS instance name, description, and metadata. Instance size changes require 
                  recreating the VPS.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Deletion</OuiText>
                <OuiText size="sm" color="secondary">
                  Delete VPS instances when no longer needed. Deletion is permanent and cannot be undone, 
                  so ensure you have backups of important data.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">VPS Status</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            VPS instances have the following statuses:
          </OuiText>

          <OuiStack gap="sm">
            <OuiFlex align="start" gap="md">
              <OuiBox class="w-2 h-2 rounded-full bg-warning mt-2 shrink-0" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Creating</OuiText>
                <OuiText size="sm" color="secondary">
                  The VPS is being provisioned. This process typically takes a few minutes.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <OuiFlex align="start" gap="md">
              <OuiBox class="w-2 h-2 rounded-full bg-info mt-2 shrink-0" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Starting</OuiText>
                <OuiText size="sm" color="secondary">
                  The VPS is booting up and will be available shortly.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <OuiFlex align="start" gap="md">
              <OuiBox class="w-2 h-2 rounded-full bg-success mt-2 shrink-0" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Running</OuiText>
                <OuiText size="sm" color="secondary">
                  The VPS is running and accessible. You can connect via web terminal or SSH.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <OuiFlex align="start" gap="md">
              <OuiBox class="w-2 h-2 rounded-full bg-secondary mt-2 shrink-0" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Stopped</OuiText>
                <OuiText size="sm" color="secondary">
                  The VPS is stopped and not consuming resources. You can start it when needed.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <OuiFlex align="start" gap="md">
              <OuiBox class="w-2 h-2 rounded-full bg-danger mt-2 shrink-0" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Failed</OuiText>
                <OuiText size="sm" color="secondary">
                  The VPS provisioning or operation failed. Check the error message for details.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Quotas and Limits</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            VPS instances are subject to organization quotas and limits:
          </OuiText>

          <OuiStack gap="sm" class="list-disc list-inside pl-4">
            <OuiText>
              <strong>Maximum VPS Instances:</strong> Your organization plan determines the maximum 
              number of VPS instances you can create. Contact support to increase your limit.
            </OuiText>
            <OuiText>
              <strong>Resource Limits:</strong> Each instance size has maximum CPU, RAM, and storage 
              limits. These limits cannot be exceeded, but you only pay for actual usage.
            </OuiText>
            <OuiText>
              <strong>Billing:</strong> VPS instances are billed based on actual resource usage 
              (CPU core-seconds, memory byte-seconds, bandwidth, storage) on a pay-as-you-go basis.
            </OuiText>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Best Practices</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            <strong>Security:</strong>
          </OuiText>
          <OuiStack gap="sm" class="list-disc list-inside pl-4">
            <OuiText>
              Keep your VPS instances updated with the latest security patches
            </OuiText>
            <OuiText>
              Use SSH keys instead of passwords for authentication
            </OuiText>
            <OuiText>
              Configure firewall rules to restrict access to necessary ports only
            </OuiText>
            <OuiText>
              Regularly backup important data and configurations
            </OuiText>
          </OuiStack>

          <OuiText>
            <strong>Performance:</strong>
          </OuiText>
          <OuiStack gap="sm" class="list-disc list-inside pl-4">
            <OuiText>
              Choose an instance size that matches your workload requirements
            </OuiText>
            <OuiText>
              Monitor resource usage to optimize costs and performance
            </OuiText>
            <OuiText>
              Stop VPS instances when not in use to avoid unnecessary charges
            </OuiText>
          </OuiStack>

          <OuiText>
            <strong>Cost Optimization:</strong>
          </OuiText>
          <OuiStack gap="sm" class="list-disc list-inside pl-4">
            <OuiText>
              Since billing is pay-as-you-go, stop instances during off-hours to reduce costs
            </OuiText>
            <OuiText>
              Right-size your instances based on actual usage patterns
            </OuiText>
            <OuiText>
              Use the smallest instance size that meets your performance requirements
            </OuiText>
          </OuiStack>
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
