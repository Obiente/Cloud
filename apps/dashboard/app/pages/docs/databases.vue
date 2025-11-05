<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Database Management
      </OuiText>
      <OuiText color="secondary" size="lg">
        Create and manage databases for your applications
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Database Management</OuiText>
        <OuiText size="sm" color="secondary">
          Managed databases are currently in development
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Managed databases will provide fully managed database instances with automatic backups, 
            scaling, and maintenance. Databases will be optimized for performance and reliability.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-warning/10 border border-warning/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon class="h-5 w-5 text-warning shrink-0 mt-0.5" />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  Coming Soon
                </OuiText>
                <OuiText size="sm" color="secondary">
                  Managed databases are currently in development. Check back soon for updates on 
                  when this feature will be available.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <OuiText>
            When available, managed databases will support:
          </OuiText>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">PostgreSQL</OuiText>
                <OuiText size="sm" color="secondary">
                  Fully managed PostgreSQL instances with version selection and automatic updates.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">MySQL</OuiText>
                <OuiText size="sm" color="secondary">
                  Managed MySQL instances with replication and backup support.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">MongoDB</OuiText>
                <OuiText size="sm" color="secondary">
                  Managed MongoDB instances for document-based storage needs.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold" color="primary">Redis</OuiText>
                <OuiText size="sm" color="secondary">
                  Managed Redis instances for caching and session storage.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20 mt-4">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Planned Features
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">• Automatic backups with configurable retention</OuiText>
                <OuiText size="sm" color="secondary">• Connection pooling and optimization</OuiText>
                <OuiText size="sm" color="secondary">• Monitoring and metrics</OuiText>
                <OuiText size="sm" color="secondary">• Point-in-time recovery</OuiText>
                <OuiText size="sm" color="secondary">• Maintenance windows</OuiText>
                <OuiText size="sm" color="secondary">• Scaling based on usage</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { CircleStackIcon, InformationCircleIcon } from "@heroicons/vue/24/outline";

definePageMeta({
  layout: "docs",
});
</script>
