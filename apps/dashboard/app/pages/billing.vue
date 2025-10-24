<template>
  <div class="min-h-screen bg-background text-white">
    <!-- Navigation -->
    <nav class="border-b border-surface-muted/20">
      <OuiContainer py="lg" size="7xl">
        <OuiFlex justify="between" align="center">
          <!-- Logo -->
          <OuiFlex align="center" gap="md">
            <div
              class="h-8 w-8 rounded bg-linear-to-br from-purple-500 to-cyan-500"
            />
            <OuiText size="xl" class="font-bold text-white">
              Obiente Cloud
            </OuiText>
          </OuiFlex>

          <!-- Navigation Links -->
          <OuiFlex align="center" gap="lg" class="hidden md:flex">
            <OuiButton variant="ghost" size="sm" as="nuxt-link" to="/">
              Home
            </OuiButton>
            <OuiButton variant="ghost" size="sm" as="nuxt-link" to="/dashboard">
              Dashboard
            </OuiButton>
            <OuiButton variant="ghost" size="sm" as="nuxt-link" to="/billing">
              Billing
            </OuiButton>
            <OuiButton variant="ghost" size="sm"> Support </OuiButton>
          </OuiFlex>

          <!-- User Actions -->
          <OuiFlex align="center" gap="md">
            <OuiButton variant="ghost" size="sm"> Account </OuiButton>
            <OuiButton variant="solid" size="sm"> Sign Out </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiContainer>
    </nav>

    <!-- Page Header -->
    <section class="border-b border-surface-muted/20">
      <OuiContainer py="6xl" size="7xl">
        <OuiStack gap="lg">
          <OuiText size="6xl" class="font-bold text-white">
            Billing & Usage
          </OuiText>
          <OuiText size="lg" class="text-muted max-w-2xl">
            Monitor your usage, manage payment methods, and view billing history
            for your Obiente Cloud services.
          </OuiText>
        </OuiStack>
      </OuiContainer>
    </section>

    <!-- Current Usage -->
    <section class="border-b border-surface-muted/20">
      <OuiContainer py="6xl" size="7xl">
        <OuiStack gap="2xl">
          <OuiText size="3xl" class="font-bold text-white">
            Current Usage
          </OuiText>

          <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="lg">
            <!-- vCPU Usage -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="md">
                <OuiFlex justify="between" align="start">
                  <OuiStack gap="xs">
                    <OuiText size="sm" class="text-muted">vCPU Hours</OuiText>
                    <OuiText size="2xl" class="font-bold text-white">
                      127.5
                    </OuiText>
                  </OuiStack>
                  <OuiBadge variant="success">Active</OuiBadge>
                </OuiFlex>
                <div class="w-full bg-surface-muted rounded-full h-2">
                  <div
                    class="bg-linear-to-r from-purple-500 to-cyan-500 h-2 rounded-full"
                    style="width: 68%"
                  ></div>
                </div>
                <OuiText size="sm" class="text-muted">
                  68% of monthly allocation (200 hours)
                </OuiText>
              </OuiStack>
            </OuiCard>

            <!-- Memory Usage -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="md">
                <OuiFlex justify="between" align="start">
                  <OuiStack gap="xs">
                    <OuiText size="sm" class="text-muted">Memory (GB)</OuiText>
                    <OuiText size="2xl" class="font-bold text-white">
                      512.3
                    </OuiText>
                  </OuiStack>
                  <OuiBadge variant="warning">High</OuiBadge>
                </OuiFlex>
                <div class="w-full bg-surface-muted rounded-full h-2">
                  <div
                    class="bg-linear-to-r from-purple-500 to-cyan-500 h-2 rounded-full"
                    style="width: 82%"
                  ></div>
                </div>
                <OuiText size="sm" class="text-muted">
                  82% of monthly allocation (1024 GB)
                </OuiText>
              </OuiStack>
            </OuiCard>

            <!-- Storage Usage -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="md">
                <OuiFlex justify="between" align="start">
                  <OuiStack gap="xs">
                    <OuiText size="sm" class="text-muted">Storage (GB)</OuiText>
                    <OuiText size="2xl" class="font-bold text-white">
                      85.7
                    </OuiText>
                  </OuiStack>
                  <OuiBadge variant="secondary">Normal</OuiBadge>
                </OuiFlex>
                <div class="w-full bg-surface-muted rounded-full h-2">
                  <div
                    class="bg-linear-to-r from-purple-500 to-cyan-500 h-2 rounded-full"
                    style="width: 34%"
                  ></div>
                </div>
                <OuiText size="sm" class="text-muted">
                  34% of monthly allocation (250 GB)
                </OuiText>
              </OuiStack>
            </OuiCard>
          </OuiGrid>

          <!-- Current Month Summary -->
          <OuiCard variant="outline" class="p-6">
            <OuiStack gap="lg">
              <OuiText size="xl" class="font-semibold text-white">
                Current Month Estimate
              </OuiText>
              <OuiFlex justify="between" align="center">
                <OuiStack gap="xs">
                  <OuiText size="sm" class="text-muted">January 2024</OuiText>
                  <OuiText size="3xl" class="font-bold text-white">
                    $23.47
                  </OuiText>
                  <OuiText size="sm" class="text-muted">
                    Based on current usage patterns
                  </OuiText>
                </OuiStack>
                <OuiButton variant="outline" size="sm">
                  View Details
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </OuiCard>
        </OuiStack>
      </OuiContainer>
    </section>

    <!-- Payment Information -->
    <section class="border-b border-surface-muted/20">
      <OuiContainer py="6xl" size="7xl">
        <OuiStack gap="2xl">
          <OuiFlex justify="between" align="center">
            <OuiText size="3xl" class="font-bold text-white">
              Payment Methods
            </OuiText>
            <OuiButton variant="solid" size="sm">
              <PlusIcon class="h-4 w-4 mr-2" />
              Add Payment Method
            </OuiButton>
          </OuiFlex>

          <OuiGrid cols="1" cols-lg="2" gap="lg">
            <!-- Primary Payment Method -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="lg">
                <OuiFlex justify="between" align="start">
                  <OuiStack gap="sm">
                    <OuiFlex align="center" gap="sm">
                      <CreditCardIcon class="h-6 w-6 text-purple-400" />
                      <OuiText size="lg" class="font-semibold text-white">
                        •••• •••• •••• 4242
                      </OuiText>
                      <OuiBadge variant="success">Primary</OuiBadge>
                    </OuiFlex>
                    <OuiText size="sm" class="text-muted">
                      Visa • Expires 12/26
                    </OuiText>
                  </OuiStack>
                  <OuiButton variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </OuiButton>
                </OuiFlex>

                <OuiFlex gap="sm">
                  <OuiButton variant="outline" size="sm"> Update </OuiButton>
                  <OuiButton variant="ghost" size="sm"> Remove </OuiButton>
                </OuiFlex>
              </OuiStack>
            </OuiCard>

            <!-- Backup Payment Method -->
            <OuiCard variant="outline" class="p-6 border-dashed">
              <OuiStack gap="lg" class="text-center">
                <div class="mx-auto">
                  <CreditCardIcon class="h-12 w-12 text-muted mx-auto mb-4" />
                  <OuiText size="lg" class="font-semibold text-white">
                    Add Backup Payment
                  </OuiText>
                  <OuiText size="sm" class="text-muted mt-2">
                    Ensure uninterrupted service with a backup payment method
                  </OuiText>
                </div>
                <OuiButton variant="outline" size="sm" class="self-center">
                  Add Card
                </OuiButton>
              </OuiStack>
            </OuiCard>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>
    </section>

    <!-- Billing History -->
    <section class="border-b border-surface-muted/20">
      <OuiContainer py="6xl" size="7xl">
        <OuiStack gap="2xl">
          <OuiFlex justify="between" align="center">
            <OuiText size="3xl" class="font-bold text-white">
              Billing History
            </OuiText>
            <OuiButton variant="outline" size="sm">
              <ArrowDownTrayIcon class="h-4 w-4 mr-2" />
              Download All
            </OuiButton>
          </OuiFlex>

          <OuiCard variant="raised" class="overflow-hidden">
            <!-- Table Header -->
            <div class="p-6 border-b border-surface-muted/20">
              <OuiGrid cols="5" gap="md" class="text-sm font-medium text-muted">
                <OuiText>Invoice</OuiText>
                <OuiText>Date</OuiText>
                <OuiText>Amount</OuiText>
                <OuiText>Status</OuiText>
                <OuiText>Actions</OuiText>
              </OuiGrid>
            </div>

            <!-- Table Rows -->
            <div class="divide-y divide-surface-muted/20">
              <div class="p-6">
                <OuiGrid cols="5" gap="md" class="items-center">
                  <OuiText size="sm" class="font-medium text-white">
                    #INV-2024-001
                  </OuiText>
                  <OuiText size="sm" class="text-muted"> Jan 1, 2024 </OuiText>
                  <OuiText size="sm" class="font-medium text-white">
                    $28.50
                  </OuiText>
                  <OuiBadge variant="success">Paid</OuiBadge>
                  <OuiFlex gap="sm">
                    <OuiButton variant="ghost" size="xs"> View </OuiButton>
                    <OuiButton variant="ghost" size="xs"> Download </OuiButton>
                  </OuiFlex>
                </OuiGrid>
              </div>

              <div class="p-6">
                <OuiGrid cols="5" gap="md" class="items-center">
                  <OuiText size="sm" class="font-medium text-white">
                    #INV-2023-012
                  </OuiText>
                  <OuiText size="sm" class="text-muted"> Dec 1, 2023 </OuiText>
                  <OuiText size="sm" class="font-medium text-white">
                    $31.75
                  </OuiText>
                  <OuiBadge variant="success">Paid</OuiBadge>
                  <OuiFlex gap="sm">
                    <OuiButton variant="ghost" size="xs"> View </OuiButton>
                    <OuiButton variant="ghost" size="xs"> Download </OuiButton>
                  </OuiFlex>
                </OuiGrid>
              </div>

              <div class="p-6">
                <OuiGrid cols="5" gap="md" class="items-center">
                  <OuiText size="sm" class="font-medium text-white">
                    #INV-2023-011
                  </OuiText>
                  <OuiText size="sm" class="text-muted"> Nov 1, 2023 </OuiText>
                  <OuiText size="sm" class="font-medium text-white">
                    $24.20
                  </OuiText>
                  <OuiBadge variant="success">Paid</OuiBadge>
                  <OuiFlex gap="sm">
                    <OuiButton variant="ghost" size="xs"> View </OuiButton>
                    <OuiButton variant="ghost" size="xs"> Download </OuiButton>
                  </OuiFlex>
                </OuiGrid>
              </div>
            </div>
          </OuiCard>
        </OuiStack>
      </OuiContainer>
    </section>

    <!-- Account Settings -->
    <section>
      <OuiContainer py="6xl" size="7xl">
        <OuiStack gap="2xl">
          <OuiText size="3xl" class="font-bold text-white">
            Account Settings
          </OuiText>

          <OuiGrid cols="1" cols-lg="2" gap="lg">
            <!-- Billing Preferences -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="lg">
                <OuiText size="xl" class="font-semibold text-white">
                  Billing Preferences
                </OuiText>

                <OuiStack gap="md">
                  <OuiFlex
                    justify="between"
                    align="center"
                    class="py-3 border-b border-surface-muted/20"
                  >
                    <OuiStack gap="xs">
                      <OuiText size="sm" class="font-medium text-white">
                        Email Notifications
                      </OuiText>
                      <OuiText size="xs" class="text-muted">
                        Receive billing updates via email
                      </OuiText>
                    </OuiStack>
                    <div
                      class="relative inline-block w-10 h-6 bg-green-500 rounded-full"
                    >
                      <div
                        class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transform translate-x-4 transition-transform"
                      ></div>
                    </div>
                  </OuiFlex>

                  <OuiFlex
                    justify="between"
                    align="center"
                    class="py-3 border-b border-surface-muted/20"
                  >
                    <OuiStack gap="xs">
                      <OuiText size="sm" class="font-medium text-white">
                        Usage Alerts
                      </OuiText>
                      <OuiText size="xs" class="text-muted">
                        Alert when approaching limits
                      </OuiText>
                    </OuiStack>
                    <div
                      class="relative inline-block w-10 h-6 bg-green-500 rounded-full"
                    >
                      <div
                        class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transform translate-x-4 transition-transform"
                      ></div>
                    </div>
                  </OuiFlex>

                  <OuiFlex justify="between" align="center" class="py-3">
                    <OuiStack gap="xs">
                      <OuiText size="sm" class="font-medium text-white">
                        Auto-Pay
                      </OuiText>
                      <OuiText size="xs" class="text-muted">
                        Automatically pay invoices
                      </OuiText>
                    </OuiStack>
                    <div
                      class="relative inline-block w-10 h-6 bg-surface-muted rounded-full"
                    >
                      <div
                        class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform"
                      ></div>
                    </div>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCard>

            <!-- Billing Information -->
            <OuiCard variant="raised" class="p-6">
              <OuiStack gap="lg">
                <OuiText size="xl" class="font-semibold text-white">
                  Billing Information
                </OuiText>

                <OuiStack gap="md">
                  <div>
                    <OuiText size="sm" class="text-muted mb-1">
                      Billing Email
                    </OuiText>
                    <OuiText size="sm" class="text-white">
                      billing@example.com
                    </OuiText>
                  </div>

                  <div>
                    <OuiText size="sm" class="text-muted mb-1">
                      Company Name
                    </OuiText>
                    <OuiText size="sm" class="text-white">
                      Acme Corporation
                    </OuiText>
                  </div>

                  <div>
                    <OuiText size="sm" class="text-muted mb-1">
                      Tax ID
                    </OuiText>
                    <OuiText size="sm" class="text-white"> 12-3456789 </OuiText>
                  </div>

                  <OuiButton variant="outline" size="sm" class="self-start">
                    Update Information
                  </OuiButton>
                </OuiStack>
              </OuiStack>
            </OuiCard>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>
    </section>

    <!-- Footer -->
    <AppFooter />
  </div>
</template>

<script setup lang="ts">
  import {
    PlusIcon,
    CreditCardIcon,
    PencilIcon,
    ArrowDownTrayIcon,
  } from "@heroicons/vue/24/outline";

  // Page meta
  definePageMeta({
    layout: false,
  });

  // SEO
  useHead({
    title: "Billing & Usage - Obiente Cloud",
    meta: [
      {
        name: "description",
        content:
          "Monitor your usage, manage payment methods, and view billing history for your Obiente Cloud services.",
      },
    ],
  });
</script>
