<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Managed Databases
      </OuiText>
      <OuiText color="tertiary" size="lg">
        Create database instances, copy connection strings, and manage
        credentials from the dashboard
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="tertiary">
          Managed databases are available directly in Obiente Cloud
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Obiente managed databases give you a provisioned database instance
            with a public connection hostname, generated credentials, and
            connection details directly in the dashboard.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-info/10 border border-info/20">
            <OuiFlex align="start" gap="md">
              <InformationCircleIcon
                class="h-5 w-5 text-info shrink-0 mt-0.5"
              />
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="primary">
                  What the dashboard currently handles well
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  Create a database, copy the connection string, inspect host
                  and port details, and reset credentials when supported by the
                  selected engine.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <OuiGrid cols="1" cols-md="2" gap="md">
            <OuiBox
              p="md"
              rounded="lg"
              class="bg-surface-muted/40 ring-1 ring-border-muted"
            >
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <CircleStackIcon class="h-5 w-5 text-primary" />
                  <OuiText size="sm" weight="semibold" color="primary"
                    >Supported Engines</OuiText
                  >
                </OuiFlex>
                <OuiText size="sm" color="tertiary">
                  PostgreSQL, MySQL, MariaDB, MongoDB, and Redis can all be
                  provisioned from the dashboard.
                </OuiText>
              </OuiStack>
            </OuiBox>

            <OuiBox
              p="md"
              rounded="lg"
              class="bg-surface-muted/40 ring-1 ring-border-muted"
            >
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <LinkIcon class="h-5 w-5 text-primary" />
                  <OuiText size="sm" weight="semibold" color="primary"
                    >Public Hostnames</OuiText
                  >
                </OuiFlex>
                <OuiText size="sm" color="tertiary">
                  Each database gets a managed hostname such as
                  <OuiCode code="db-8e98bdb7f10a47ec.my.obiente.cloud" /> that
                  can be used from apps and external clients.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Typical Workflow</OuiText>
        <OuiText size="sm" color="tertiary">
          The usual path from creation to application connection
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiText size="sm" color="tertiary">
            1. Create the database from the Databases page.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            2. Open the database and copy the host, port, username, and full
            connection string.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            3. Put those values into your deployment, Compose stack, or external
            application.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            4. Use the Connection tab again any time you need to confirm
            hostnames or ports.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            5. If the password is lost, reset it from the Connection tab and
            update the consuming application immediately.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Connection Details</OuiText>
        <OuiText size="sm" color="tertiary">
          How to use the hostnames and credentials shown in the dashboard
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            The Connection panel shows the canonical hostname, the exposed port,
            the database name, and engine-specific connection strings. For most
            users, copying the generated connection string is the safest option.
          </OuiText>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Compose and multi-service apps</OuiText
              >
              <OuiText size="sm" color="tertiary">
                If your application runs in an Obiente Compose deployment, keep
                the distinction clear: use Compose service names like
                <OuiCode code="cache" /> for containers inside the same stack,
                and use the database hostname from the Connection panel for the
                managed database itself.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiAlert color="warning">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="semibold"
                >Passwords are only shown once</OuiText
              >
              <OuiText size="sm">
                Save the initial password when the database is created, or save
                the new one immediately after a password reset.
              </OuiText>
            </OuiStack>
          </OuiAlert>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Password Resets</OuiText>
        <OuiText size="sm" color="tertiary">
          Reset behavior depends on the selected engine
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox
            p="md"
            rounded="lg"
            class="bg-success/10 border border-success/20"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Supported today</OuiText
              >
              <OuiText size="sm" color="tertiary">
                PostgreSQL, MySQL, and MariaDB password resets are applied in
                the database engine itself.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="md"
            rounded="lg"
            class="bg-warning/10 border border-warning/20"
          >
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary"
                >Use caution</OuiText
              >
              <OuiText size="sm" color="tertiary">
                MongoDB and Redis workflows may differ. Check the returned
                connection details and test before relying on a reset in
                production.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>

        <OuiText size="sm" color="tertiary">
          After a reset, update every deployment, Compose stack, secret store,
          or external client that still holds the old password.
        </OuiText>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Current Limitations</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiText size="sm" color="tertiary">
            Backup scheduling is not currently a self-serve setting in the
            database UI.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            DNS and connection issues usually come from platform DNS health or
            client-side configuration, not from the connection panel itself.
          </OuiText>
          <OuiText size="sm" color="tertiary">
            If a managed database hostname does not resolve from your app,
            verify Obiente DNS health and your container’s DNS resolver before
            changing application code.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  CircleStackIcon,
  InformationCircleIcon,
  LinkIcon,
} from "@heroicons/vue/24/outline";

definePageMeta({
  layout: "docs",
});
</script>
