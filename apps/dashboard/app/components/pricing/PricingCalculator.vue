<template>
  <OuiContainer size="7xl" py="6xl" class="px-3 md:px-4">
    <OuiStack gap="xl" align="center">
      <!-- Header Section -->
      <OuiStack gap="md" align="center" class="text-center" style="max-width: 56rem;">
        <OuiBox class="mb-2">
          <OuiBadge 
            variant="secondary" 
            class="px-4 py-1.5 bg-accent-primary/10 border border-accent-primary/20 text-accent-primary"
          >
            <OuiText size="sm" weight="semibold" class="tracking-wide">
              PRICING CALCULATOR
            </OuiText>
          </OuiBadge>
        </OuiBox>
        <OuiText
          as="h2"
          size="3xl"
          weight="bold"
          color="primary"
          class="md:text-5xl"
        >
          Estimate Your Monthly Costs
        </OuiText>
        <OuiText size="lg" color="secondary" class="md:text-xl" style="max-width: 42rem;">
          Calculate your monthly costs based on actual usage. Adjust the sliders below to see how much you'll pay. 
          <OuiText as="span" weight="semibold" color="primary">Pay only for what you use</OuiText> - no fixed plans or hidden fees.
        </OuiText>
      </OuiStack>

      <!-- Top Section: Info Banner and Scenario Selector -->
      <OuiGrid :cols="{ sm: 1, lg: 2 }" gap="lg" style="max-width: 72rem; width: 100%;">
        <!-- Pricing Info Banner -->
        <OuiCard variant="default" class="border-accent-primary/20 bg-accent-primary/5">
          <OuiCardBody>
            <OuiFlex gap="md" align="start">
              <OuiBox class="mt-0.5 shrink-0">
                <InformationCircleIcon
                  class="h-5 w-5 text-accent-primary"
                />
              </OuiBox>
              <OuiStack gap="xs">
                <OuiText size="sm" weight="semibold" color="primary">
                  Future Pricing Updates
                </OuiText>
                <OuiText size="sm" color="secondary">
                  As we grow and achieve better economies of scale, we plan to reduce pricing for storage and other resources.
                  We're committed to passing cost savings along to our customers.
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Scenario Selector -->
        <OuiCard variant="default">
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiFlex align="center" gap="sm">
                <RocketLaunchIcon class="h-5 w-5 text-accent-primary shrink-0" />
                <OuiText size="lg" weight="semibold" color="primary">
                  Quick Start Scenarios
                </OuiText>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Choose a preset scenario or select "Custom" to configure your own resources.
              </OuiText>
              <OuiSegmentGroup
                v-model="selectedScenario"
                :options="scenarioOptions"
                class="w-full"
              />
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Calculator -->
      <OuiCard variant="default" class="shadow-lg" style="max-width: 72rem; width: 100%;">
        <OuiCardBody class="p-6 md:p-8">
          <OuiGrid :cols="{ sm: 1, md: 2 }" gap="lg">
            <!-- Memory Slider -->
            <OuiCard variant="raised" class="border-border-muted">
              <OuiCardBody class="p-5">
                <OuiStack gap="md">
                  <OuiFlex justify="between" align="center">
                    <OuiFlex align="center" gap="sm">
                      <CircleStackIcon class="h-5 w-5 text-accent-primary" />
                      <OuiText size="lg" weight="semibold" color="primary">
                        Memory (RAM)
                      </OuiText>
                    </OuiFlex>
                    <OuiBadge variant="primary" class="px-3 py-1">
                      <OuiText size="lg" weight="bold" color="primary">
                        {{ formatMemory(memoryGB ?? 0) }}
                      </OuiText>
                    </OuiBadge>
                  </OuiFlex>
                  <OuiSlider
                    v-model="memorySliderValue"
                    :min="0.25"
                    :max="32"
                    :step="0.25"
                  />
                  <OuiFlex justify="between" class="text-xs text-secondary">
                    <span>512 MB</span>
                    <span>32 GB</span>
                  </OuiFlex>
                  <OuiBox class="mt-2 p-3 rounded-lg bg-accent-primary/10 border border-accent-primary/20">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">24/7 monthly cost:</OuiText>
                      <OuiText size="sm" weight="bold" color="primary">
                        {{ formatCurrency(memoryCostMonthly) }}
                      </OuiText>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- vCPU Slider -->
            <OuiCard variant="raised" class="border-border-muted">
              <OuiCardBody class="p-5">
                <OuiStack gap="md">
                  <OuiFlex justify="between" align="center">
                    <OuiFlex align="center" gap="sm">
                      <ServerIcon class="h-5 w-5 text-accent-secondary" />
                      <OuiText size="lg" weight="semibold" color="primary">
                        vCPU Cores
                      </OuiText>
                    </OuiFlex>
                    <OuiBadge variant="secondary" class="px-3 py-1">
                      <OuiText size="lg" weight="bold" color="primary">
                        {{ (cpuCores ?? 0).toFixed(2) }} cores
                      </OuiText>
                    </OuiBadge>
                  </OuiFlex>
                  <OuiSlider
                    v-model="cpuSliderValue"
                    :min="0.25"
                    :max="8"
                    :step="0.25"
                  />
                  <OuiFlex justify="between" class="text-xs text-secondary">
                    <span>0.25 cores</span>
                    <span>8 cores</span>
                  </OuiFlex>
                  <OuiBox class="mt-2 p-3 rounded-lg bg-accent-secondary/10 border border-accent-secondary/20">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">24/7 monthly cost:</OuiText>
                      <OuiText size="sm" weight="bold" color="primary">
                        {{ formatCurrency(cpuCostMonthly) }}
                      </OuiText>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Bandwidth Slider -->
            <OuiCard variant="raised" class="border-border-muted">
              <OuiCardBody class="p-5">
                <OuiStack gap="md">
                  <OuiFlex justify="between" align="center">
                    <OuiFlex align="center" gap="sm">
                      <GlobeAltIcon class="h-5 w-5 text-accent-info" />
                      <OuiText size="lg" weight="semibold" color="primary">
                        Bandwidth
                      </OuiText>
                    </OuiFlex>
                    <OuiBadge variant="secondary" class="px-3 py-1">
                      <OuiText size="lg" weight="bold" color="primary">
                        {{ formatBandwidth(bandwidthGB ?? 0) }}
                      </OuiText>
                    </OuiBadge>
                  </OuiFlex>
                  <OuiSlider
                    v-model="bandwidthSliderValue"
                    :min="1"
                    :max="1000"
                    :step="1"
                  />
                  <OuiFlex justify="between" class="text-xs text-secondary">
                    <span>1 GB</span>
                    <span>1 TB</span>
                  </OuiFlex>
                  <OuiBox class="mt-2 p-3 rounded-lg bg-accent-info/10 border border-accent-info/20">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Monthly cost:</OuiText>
                      <OuiText size="sm" weight="bold" color="primary">
                        {{ formatCurrency(bandwidthCostMonthly) }}
                      </OuiText>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Storage Slider -->
            <OuiCard variant="raised" class="border-border-muted">
              <OuiCardBody class="p-5">
                <OuiStack gap="md">
                  <OuiFlex justify="between" align="center">
                    <OuiFlex align="center" gap="sm">
                      <ArchiveBoxIcon class="h-5 w-5 text-accent-warning" />
                      <OuiText size="lg" weight="semibold" color="primary">
                        Storage
                      </OuiText>
                    </OuiFlex>
                    <OuiBadge variant="secondary" class="px-3 py-1">
                      <OuiText size="lg" weight="bold" color="primary">
                        {{ formatStorage(storageGB ?? 0) }}
                      </OuiText>
                    </OuiBadge>
                  </OuiFlex>
                  <OuiSlider
                    v-model="storageSliderValue"
                    :min="1"
                    :max="500"
                    :step="1"
                  />
                  <OuiFlex justify="between" class="text-xs text-secondary">
                    <span>1 GB</span>
                    <span>500 GB</span>
                  </OuiFlex>
                  <OuiBox class="mt-2 p-3 rounded-lg bg-accent-warning/10 border border-accent-warning/20">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Monthly cost:</OuiText>
                      <OuiText size="sm" weight="bold" color="primary">
                        {{ formatCurrency(storageCostMonthly) }}
                      </OuiText>
                    </OuiFlex>
                    <OuiText size="xs" color="secondary" class="mt-1 opacity-75">
                      Higher pricing reflects limited capacity
                    </OuiText>
                  </OuiBox>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiGrid>

          <!-- Cost Summary -->
          <OuiCard variant="raised" class="mt-6 border-2 border-accent-primary/30 shadow-xl">
            <OuiCardBody class="p-6 md:p-8">
              <OuiStack gap="lg" align="center">
                <OuiStack gap="md" align="center" class="w-full">
                  <OuiText
                    size="sm"
                    weight="semibold"
                    color="secondary"
                    transform="uppercase"
                    class="tracking-wider"
                  >
                    Estimated Monthly Cost
                  </OuiText>
                  
                  <OuiGrid :cols="{ sm: 1, md: 2 }" gap="lg" class="w-full">
                    <OuiCard variant="raised" class="border-2 border-accent-primary">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">
                            24/7 Maximum
                          </OuiText>
                          <OuiText size="3xl" weight="bold" color="primary">
                            {{ formatCurrency(totalMonthlyCost) }}
                          </OuiText>
                          <OuiText size="xs" color="secondary">
                            Continuous uptime
                          </OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                    
                    <OuiCard variant="raised" class="border-2 border-accent-success">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">
                            Realistic Cost
                          </OuiText>
                          <OuiText size="3xl" weight="bold" color="success">
                            {{ formatCurrency(realisticMonthlyCost) }}
                          </OuiText>
                          <OuiText size="xs" color="secondary">
                            {{ currentScenarioDescription }}
                          </OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                  </OuiGrid>
                  
                  <!-- Uptime Slider -->
                  <OuiCard variant="raised" class="border-border-muted w-full">
                    <OuiCardBody class="p-5">
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="center">
                          <OuiText size="sm" weight="semibold" color="primary">
                            Adjust Expected Uptime
                          </OuiText>
                          <OuiBadge variant="secondary" class="px-3 py-1">
                            <OuiText size="sm" weight="bold" color="primary">
                              {{ (uptimeSliderValue[0] ?? 95).toFixed(0) }}%
                            </OuiText>
                          </OuiBadge>
                        </OuiFlex>
                        <OuiSlider
                          v-model="uptimeSliderValue"
                          :min="10"
                          :max="100"
                          :step="5"
                        />
                        <OuiFlex justify="between" class="text-xs text-secondary">
                          <span>10% (2.4 hrs/day)</span>
                          <span>100% (24/7)</span>
                        </OuiFlex>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                  
                  <!-- Utilization Sliders -->
                  <OuiCard variant="raised" class="border-border-muted w-full">
                    <OuiCardBody class="p-5">
                      <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" weight="semibold" color="primary">
                              Average CPU Utilization
                            </OuiText>
                            <OuiBadge variant="secondary" class="px-3 py-1">
                              <OuiText size="sm" weight="bold" color="primary">
                                {{ (cpuUtilizationSliderValue[0] ?? 40).toFixed(0) }}%
                              </OuiText>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiSlider
                            v-model="cpuUtilizationSliderValue"
                            :min="10"
                            :max="100"
                            :step="5"
                          />
                          <OuiFlex justify="between" class="text-xs text-secondary">
                            <span>10% (idle)</span>
                            <span>100% (constant load)</span>
                          </OuiFlex>
                          <OuiText size="xs" color="secondary" class="opacity-75">
                            Most apps average 20-50% CPU; game servers can spike higher during play.
                          </OuiText>
                        </OuiStack>

                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" weight="semibold" color="primary">
                              Average Memory Working Set
                            </OuiText>
                            <OuiBadge variant="secondary" class="px-3 py-1">
                              <OuiText size="sm" weight="bold" color="primary">
                                {{ (memoryUtilizationSliderValue[0] ?? 75).toFixed(0) }}%
                              </OuiText>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiSlider
                            v-model="memoryUtilizationSliderValue"
                            :min="30"
                            :max="100"
                            :step="5"
                          />
                          <OuiFlex justify="between" class="text-xs text-secondary">
                            <span>30% (mostly idle)</span>
                            <span>100% (full allocation)</span>
                          </OuiFlex>
                          <OuiText size="xs" color="secondary" class="opacity-75">
                            Many apps keep 60-85% of RAM active; adjust for your workload.
                          </OuiText>
                        </OuiStack>
                      </OuiGrid>

                      <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md" class="mt-4">
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" weight="semibold" color="primary">
                              Average Bandwidth Usage
                            </OuiText>
                            <OuiBadge variant="secondary" class="px-3 py-1">
                              <OuiText size="sm" weight="bold" color="primary">
                                {{ (bandwidthUtilizationSliderValue[0] ?? 50).toFixed(0) }}%
                              </OuiText>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiSlider
                            v-model="bandwidthUtilizationSliderValue"
                            :min="10"
                            :max="100"
                            :step="5"
                          />
                          <OuiFlex justify="between" class="text-xs text-secondary">
                            <span>10% (light traffic)</span>
                            <span>100% (constant heavy traffic)</span>
                          </OuiFlex>
                        </OuiStack>

                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" weight="semibold" color="primary">
                              Average Storage Used
                            </OuiText>
                            <OuiBadge variant="secondary" class="px-3 py-1">
                              <OuiText size="sm" weight="bold" color="primary">
                                {{ (storageUtilizationSliderValue[0] ?? 60).toFixed(0) }}%
                              </OuiText>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiSlider
                            v-model="storageUtilizationSliderValue"
                            :min="10"
                            :max="100"
                            :step="5"
                          />
                          <OuiFlex justify="between" class="text-xs text-secondary">
                            <span>10% (minimal data)</span>
                            <span>100% (full allocation)</span>
                          </OuiFlex>
                        </OuiStack>
                      </OuiGrid>
                    </OuiCardBody>
                  </OuiCard>
                </OuiStack>
                  
                  <OuiBox class="w-full h-px bg-border-muted my-2" />
                  
                  <OuiGrid :cols="{ sm: 2, md: 4 }" gap="md" class="w-full mt-2">
                    <OuiCard variant="raised" class="border-border-muted">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <CircleStackIcon class="h-4 w-4 text-accent-primary" />
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">Memory</OuiText>
                          <OuiText size="lg" weight="bold" color="primary">{{
                            formatCurrency(realisticMemoryCostMonthly)
                          }}</OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                    <OuiCard variant="raised" class="border-border-muted">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <ServerIcon class="h-4 w-4 text-accent-secondary" />
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">vCPU</OuiText>
                          <OuiText size="lg" weight="bold" color="primary">{{
                            formatCurrency(realisticCpuCostMonthly)
                          }}</OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                    <OuiCard variant="raised" class="border-border-muted">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <GlobeAltIcon class="h-4 w-4 text-accent-info" />
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">Bandwidth</OuiText>
                          <OuiText size="lg" weight="bold" color="primary">{{
                            formatCurrency(realisticBandwidthCostMonthly)
                          }}</OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                    <OuiCard variant="raised" class="border-border-muted">
                      <OuiCardBody class="p-4">
                        <OuiStack gap="xs" align="center">
                          <ArchiveBoxIcon class="h-4 w-4 text-accent-warning" />
                          <OuiText size="xs" color="secondary" transform="uppercase" class="tracking-wide">Storage</OuiText>
                          <OuiText size="lg" weight="bold" color="primary">{{
                            formatCurrency(realisticStorageCostMonthly)
                          }}</OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                  </OuiGrid>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
        </OuiCardBody>
      </OuiCard>

      <!-- FAQ Accordion -->
      <OuiCard variant="default" class="w-full max-w-4xl">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiText size="lg" weight="semibold" color="primary">
              Frequently Asked Questions
            </OuiText>
            <OuiAccordion :items="faqItems" />
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Reduced Pricing Collapsible -->
      <OuiCard
        variant="outline"
        class="w-full max-w-4xl border-border-muted/50"
      >
        <OuiCardBody>
          <OuiCollapsible
            v-model="freePlansOpen"
            label="Reduced Pricing Available"
          >
            <OuiStack gap="md">
              <OuiText size="sm" color="secondary">
                We may offer reduced pricing for qualifying customers,
                including:
              </OuiText>
              <OuiStack gap="sm">
                <OuiFlex gap="sm" align="center">
                  <CheckIcon
                    class="h-4 w-4 text-accent-success shrink-0 opacity-60"
                  />
                  <OuiText size="sm" color="secondary" class="opacity-80">
                    <strong>Students</strong> - Educational projects and
                    coursework
                  </OuiText>
                </OuiFlex>
                <OuiFlex gap="sm" align="center">
                  <CheckIcon
                    class="h-4 w-4 text-accent-success shrink-0 opacity-60"
                  />
                  <OuiText size="sm" color="secondary" class="opacity-80">
                    <strong>Open-source projects</strong> - Non-commercial
                    open-source initiatives
                  </OuiText>
                </OuiFlex>
                <OuiFlex gap="sm" align="center">
                  <CheckIcon
                    class="h-4 w-4 text-accent-success shrink-0 opacity-60"
                  />
                  <OuiText size="sm" color="secondary" class="opacity-80">
                    <strong>Non-profits</strong> - Registered non-profit
                    organizations
                  </OuiText>
                </OuiFlex>
                <OuiFlex gap="sm" align="center">
                  <CheckIcon
                    class="h-4 w-4 text-accent-success shrink-0 opacity-60"
                  />
                  <OuiText size="sm" color="secondary" class="opacity-80">
                    <strong>Early-stage startups</strong> - Pre-revenue startups
                    and MVPs
                  </OuiText>
                </OuiFlex>
              </OuiStack>
              <OuiText
                size="sm"
                color="secondary"
                class="mt-2 opacity-70 italic"
              >
                Reduced pricing is not available through the dashboard - please
                contact us to discuss your eligibility and requirements.
              </OuiText>
            </OuiStack>
          </OuiCollapsible>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  import { computed, ref, watch, onMounted } from "vue";
  import { 
    InformationCircleIcon, 
    CheckIcon,
    RocketLaunchIcon,
    CircleStackIcon,
    GlobeAltIcon,
    ArchiveBoxIcon,
    ServerIcon
  } from "@heroicons/vue/24/outline";
  import { SuperadminService } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";

  // Component imports
  import OuiSlider from "~/components/oui/Slider.vue";
  import OuiSegmentGroup from "~/components/oui/SegmentGroup.vue";
  import OuiAccordion from "~/components/oui/Accordion.vue";
  import OuiCollapsible from "~/components/oui/Collapsible.vue";

  // Pricing state
  interface PricingInfo {
    cpuCostPerCoreSecond: number;
    memoryCostPerByteSecond: number;
    bandwidthCostPerByte: number;
    storageCostPerByteMonth: number;
  }

  const pricing = ref<PricingInfo | null>(null);
  const isLoadingPricing = ref(false);

  // Fetch pricing from API
  const client = useConnectClient(SuperadminService);
  const fetchPricing = async () => {
    try {
      isLoadingPricing.value = true;
      const response = await client.getPricing({});
      
      pricing.value = {
        cpuCostPerCoreSecond: Number(response.cpuCostPerCoreSecond) || 0.000000761,
        memoryCostPerByteSecond: Number(response.memoryCostPerByteSecond) || 0.000000000000001063,
        bandwidthCostPerByte: Number(response.bandwidthCostPerByte) || 0.000000000009313,
        storageCostPerByteMonth: Number(response.storageCostPerByteMonth) || 0.000000000186264,
      };
    } catch (error) {
      console.error("Failed to fetch pricing:", error);
      // Use defaults if API fails
      pricing.value = {
        cpuCostPerCoreSecond: 0.000000761,
        memoryCostPerByteSecond: 0.000000000000001063,
        bandwidthCostPerByte: 0.000000000009313,
        storageCostPerByteMonth: 0.000000000186264,
      };
    } finally {
      isLoadingPricing.value = false;
    }
  };

  onMounted(() => {
    fetchPricing();
  });

  // Scenarios
  const scenarios = {
    small: {
      memory: 0.5,
      cpu: 0.5,
      bandwidth: 10,
      storage: 5,
      uptimeMultiplier: 0.8,
      cpuUtilization: 0.3,
      memoryUtilization: 0.7,
      bandwidthUtilization: 0.4,
      storageUtilization: 0.5,
      description: "~19 hours/day",
    },
    medium: {
      memory: 2,
      cpu: 1,
      bandwidth: 50,
      storage: 20,
      uptimeMultiplier: 0.95,
      cpuUtilization: 0.4,
      memoryUtilization: 0.75,
      bandwidthUtilization: 0.5,
      storageUtilization: 0.6,
      description: "~23 hours/day",
    },
    large: {
      memory: 4,
      cpu: 2,
      bandwidth: 100,
      storage: 50,
      uptimeMultiplier: 0.99,
      cpuUtilization: 0.5,
      memoryUtilization: 0.8,
      bandwidthUtilization: 0.6,
      storageUtilization: 0.75,
      description: "~24 hours/day",
    },
    smallgame: {
      memory: 2,
      cpu: 1,
      bandwidth: 50,
      storage: 30,
      uptimeMultiplier: 0.5,
      cpuUtilization: 0.6,
      memoryUtilization: 0.85,
      bandwidthUtilization: 0.7,
      storageUtilization: 0.7,
      description: "~12 hours/day",
    },
    biggame: {
      memory: 4,
      cpu: 2,
      bandwidth: 100,
      storage: 75,
      uptimeMultiplier: 0.67,
      cpuUtilization: 0.7,
      memoryUtilization: 0.9,
      bandwidthUtilization: 0.8,
      storageUtilization: 0.85,
      description: "~16 hours/day",
    },
  };

  const scenarioOptions = [
    { label: "Small App", value: "small" },
    { label: "Medium App", value: "medium" },
    { label: "Large App", value: "large" },
    { label: "Small Game Server", value: "smallgame" },
    { label: "Big Game Server", value: "biggame" },
  ];

  const selectedScenario = ref("medium");

  // Slider values
  const memorySliderValue = ref([2]);
  const cpuSliderValue = ref([1]);
  const bandwidthSliderValue = ref([50]);
  const storageSliderValue = ref([25]);
  const uptimeSliderValue = ref([95]); // Default to 95% uptime
  const cpuUtilizationSliderValue = ref([40]); // Default to 40% CPU usage
  const memoryUtilizationSliderValue = ref([75]); // Default to 75% memory usage of allocated when running
  const bandwidthUtilizationSliderValue = ref([50]); // Default to 50% of provisioned bandwidth use
  const storageUtilizationSliderValue = ref([60]); // Default to 60% of provisioned storage used

  // Watch scenario changes
  watch(selectedScenario, (newScenario) => {
    if (scenarios[newScenario as keyof typeof scenarios]) {
      const scenario = scenarios[newScenario as keyof typeof scenarios];
      memorySliderValue.value = [scenario.memory];
      cpuSliderValue.value = [scenario.cpu];
      bandwidthSliderValue.value = [scenario.bandwidth];
      storageSliderValue.value = [scenario.storage];
      uptimeSliderValue.value = [scenario.uptimeMultiplier * 100];
      cpuUtilizationSliderValue.value = [scenario.cpuUtilization * 100];
      memoryUtilizationSliderValue.value = [scenario.memoryUtilization * 100];
      bandwidthUtilizationSliderValue.value = [scenario.bandwidthUtilization * 100];
      storageUtilizationSliderValue.value = [scenario.storageUtilization * 100];
    }
  });

  // Computed values
  const memoryGB = computed(() => memorySliderValue.value[0] ?? 0);
  const cpuCores = computed(() => cpuSliderValue.value[0] ?? 0);
  const bandwidthGB = computed(() => bandwidthSliderValue.value[0] ?? 0);
  const storageGB = computed(() => storageSliderValue.value[0] ?? 0);

  // Constants
  const HOURS_PER_MONTH = 730; // Average hours per month
  const GB_TO_BYTES = 1073741824;

  // Calculate costs
  const memoryCostMonthly = computed(() => {
    if (!pricing.value || !Number.isFinite(pricing.value.memoryCostPerByteSecond)) return 0;
    const mem = memoryGB.value ?? 0;
    if (!Number.isFinite(mem)) return 0;
    const secondsPerMonth = HOURS_PER_MONTH * 3600;
    const byteSecondsPerMonth = mem * GB_TO_BYTES * secondsPerMonth;
    const cost = byteSecondsPerMonth * pricing.value.memoryCostPerByteSecond;
    return Number.isFinite(cost) ? cost : 0;
  });

  const cpuCostMonthly = computed(() => {
    if (!pricing.value || !Number.isFinite(pricing.value.cpuCostPerCoreSecond)) return 0;
    const cpu = cpuCores.value ?? 0;
    if (!Number.isFinite(cpu)) return 0;
    const cost = cpu * HOURS_PER_MONTH * 3600 * pricing.value.cpuCostPerCoreSecond;
    return Number.isFinite(cost) ? cost : 0;
  });

  const bandwidthCostMonthly = computed(() => {
    if (!pricing.value || !Number.isFinite(pricing.value.bandwidthCostPerByte)) return 0;
    const bw = bandwidthGB.value ?? 0;
    if (!Number.isFinite(bw)) return 0;
    const cost = bw * GB_TO_BYTES * pricing.value.bandwidthCostPerByte;
    return Number.isFinite(cost) ? cost : 0;
  });

  const storageCostMonthly = computed(() => {
    if (!pricing.value || !Number.isFinite(pricing.value.storageCostPerByteMonth)) return 0;
    const storage = storageGB.value ?? 0;
    if (!Number.isFinite(storage)) return 0;
    const cost = storage * GB_TO_BYTES * pricing.value.storageCostPerByteMonth;
    return Number.isFinite(cost) ? cost : 0;
  });

  const totalMonthlyCost = computed(() => {
    const total = memoryCostMonthly.value + cpuCostMonthly.value + bandwidthCostMonthly.value + storageCostMonthly.value;
    return Number.isFinite(total) ? total : 0;
  });

  // Get current scenario uptime multiplier from slider
  const currentUptimeMultiplier = computed(() => {
    const uptimePercent = uptimeSliderValue.value[0] ?? 95;
    return uptimePercent / 100;
  });

  const currentScenarioDescription = computed(() => {
    const uptimePercent = uptimeSliderValue.value[0] ?? 95;
    const hoursPerDay = (uptimePercent / 100) * 24;
    return `~${hoursPerDay.toFixed(0)} hours/day`;
  });

  // Per-resource realistic costs based on uptime and utilization
  const realisticCpuCostMonthly = computed(() => {
    const baseCpu = cpuCostMonthly.value;
    const uptimeMultiplier = currentUptimeMultiplier.value;
    const utilization = (cpuUtilizationSliderValue.value[0] ?? 40) / 100;
    const adjusted = baseCpu * uptimeMultiplier * utilization;
    return Number.isFinite(adjusted) ? adjusted : 0;
  });

  const realisticMemoryCostMonthly = computed(() => {
    const baseMemory = memoryCostMonthly.value;
    const uptimeMultiplier = currentUptimeMultiplier.value;
    const utilization = (memoryUtilizationSliderValue.value[0] ?? 75) / 100;
    const adjusted = baseMemory * uptimeMultiplier * utilization;
    return Number.isFinite(adjusted) ? adjusted : 0;
  });

  const realisticBandwidthCostMonthly = computed(() => {
    const baseBandwidth = bandwidthCostMonthly.value;
    const utilization = (bandwidthUtilizationSliderValue.value[0] ?? 50) / 100;
    const adjusted = baseBandwidth * utilization;
    return Number.isFinite(adjusted) ? adjusted : 0;
  });

  const realisticStorageCostMonthly = computed(() => {
    const baseStorage = storageCostMonthly.value;
    const utilization = (storageUtilizationSliderValue.value[0] ?? 60) / 100;
    const adjusted = baseStorage * utilization;
    return Number.isFinite(adjusted) ? adjusted : 0;
  });

  // Calculate realistic cost based on usage patterns
  const realisticMonthlyCost = computed(() => {
    const total =
      realisticCpuCostMonthly.value +
      realisticMemoryCostMonthly.value +
      realisticBandwidthCostMonthly.value +
      realisticStorageCostMonthly.value;
    return Number.isFinite(total) ? total : 0;
  });

  // Formatting functions
  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
  };

  const formatMemory = (gb: number) => {
    if (gb < 1) {
      return `${(gb * 1024).toFixed(0)} MB`;
    }
    return `${gb.toFixed(2)} GB`;
  };

  const formatBandwidth = (gb: number) => {
    if (gb >= 1000) {
      return `${(gb / 1000).toFixed(1)} TB`;
    }
    return `${gb.toFixed(0)} GB`;
  };

  const formatStorage = (gb: number) => {
    if (gb >= 1000) {
      return `${(gb / 1000).toFixed(1)} TB`;
    }
    return `${gb.toFixed(0)} GB`;
  };

  // FAQ items
  const faqItems = [
    {
      value: "minimum",
      label: "Is there a minimum charge or setup fee?",
      content:
        "No. We only charge for actual resource usage with no minimums or setup fees.",
    },
    {
      value: "vps-comparison",
      label: "How does this compare to traditional VPS pricing?",
      content:
        "Our pricing is competitive with VPS providers like DigitalOcean and Linode. For example, 1GB RAM + 1 vCPU running 24/7 costs $5/month, similar to entry-level VPS plans. The advantage is you only pay for what you use - if your app runs part-time, you pay less.",
    },
    {
      value: "part-time",
      label: "What if my deployment only runs part-time?",
      content:
        "You only pay for actual runtime. If your deployment runs 12 hours/day instead of 24/7, you'll pay approximately half the monthly cost. This is a key advantage over traditional VPS where you pay for the full month regardless of usage.",
    },
    {
      value: "pricing-changes",
      label: "Will pricing change over time?",
      content:
        "Yes, we plan to reduce pricing for storage and other resources as we grow and achieve better economies of scale. We're committed to passing cost savings along to our customers.",
    },
  ];

  const freePlansOpen = ref(false);
</script>
