<template>
  <OuiContainer size="7xl" py="6xl">
    <OuiStack gap="2xl" align="center">
      <!-- Header -->
      <OuiStack gap="lg" align="center" class="text-center max-w-3xl">
        <OuiText
          as="h2"
          size="3xl"
          weight="bold"
          color="primary"
          class="md:text-5xl"
        >
          Pricing Calculator
        </OuiText>
        <OuiText size="lg" color="secondary" class="md:text-xl">
          Calculate your monthly costs based on actual usage. Pay only for what
          you use.
        </OuiText>
      </OuiStack>

      <!-- Pricing Info Banner -->
      <OuiCard variant="default" class="w-full max-w-4xl">
        <OuiCardBody>
          <OuiFlex gap="md" align="center">
            <InformationCircleIcon
              class="h-5 w-5 text-accent-primary shrink-0"
            />
            <OuiText size="sm" color="secondary">
              <strong>Note:</strong> As we grow and achieve better economies of
              scale, we plan to reduce pricing for storage and other resources.
              We're committed to passing cost savings along to our customers.
            </OuiText>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Scenario Selector -->
      <OuiCard variant="default" class="w-full max-w-4xl">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiText size="lg" weight="semibold" color="primary">
              Quick Start Scenarios
            </OuiText>
            <OuiSegmentGroup
              v-model="selectedScenario"
              :options="scenarioOptions"
              class="w-full"
            />
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Calculator -->
      <OuiCard variant="default" class="w-full max-w-4xl">
        <OuiCardBody>
          <OuiStack gap="xl">
            <!-- Memory Slider -->
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiText size="lg" weight="medium" color="primary">
                  Memory (RAM)
                </OuiText>
                <OuiText size="lg" weight="bold" color="accent">
                  {{ formatMemory(memoryGB ?? 0) }}
                </OuiText>
              </OuiFlex>
              <OuiSlider
                v-model="memorySliderValue"
                :min="0.25"
                :max="32"
                :step="0.25"
              />
              <OuiFlex justify="between" class="text-sm text-secondary">
                <span>512 MB</span>
                <span>32 GB</span>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Running 24/7 for a month:
                {{ formatCurrency(memoryCostMonthly) }}
              </OuiText>
            </OuiStack>

            <!-- vCPU Slider -->
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiText size="lg" weight="medium" color="primary">
                  vCPU Cores
                </OuiText>
                <OuiText size="lg" weight="bold" color="accent">
                  {{ (cpuCores ?? 0).toFixed(2) }} cores
                </OuiText>
              </OuiFlex>
              <OuiSlider
                v-model="cpuSliderValue"
                :min="0.25"
                :max="8"
                :step="0.25"
              />
              <OuiFlex justify="between" class="text-sm text-secondary">
                <span>0.25 cores</span>
                <span>8 cores</span>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Running 24/7 for a month: {{ formatCurrency(cpuCostMonthly) }}
              </OuiText>
            </OuiStack>

            <!-- Bandwidth Slider -->
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiText size="lg" weight="medium" color="primary">
                  Bandwidth
                </OuiText>
                <OuiText size="lg" weight="bold" color="accent">
                  {{ formatBandwidth(bandwidthGB ?? 0) }}
                </OuiText>
              </OuiFlex>
              <OuiSlider
                v-model="bandwidthSliderValue"
                :min="1"
                :max="1000"
                :step="1"
              />
              <OuiFlex justify="between" class="text-sm text-secondary">
                <span>1 GB</span>
                <span>1 TB</span>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Per month: {{ formatCurrency(bandwidthCostMonthly) }}
              </OuiText>
            </OuiStack>

            <!-- Storage Slider -->
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiText size="lg" weight="medium" color="primary">
                  Storage
                </OuiText>
                <OuiText size="lg" weight="bold" color="accent">
                  {{ formatStorage(storageGB ?? 0) }}
                </OuiText>
              </OuiFlex>
              <OuiSlider
                v-model="storageSliderValue"
                :min="1"
                :max="500"
                :step="1"
              />
              <OuiFlex justify="between" class="text-sm text-secondary">
                <span>1 GB</span>
                <span>500 GB</span>
              </OuiFlex>
              <OuiText size="sm" color="secondary">
                Per month: {{ formatCurrency(storageCostMonthly) }}
                <span class="text-xs opacity-75 ml-1">
                  (Higher pricing reflects limited capacity)
                </span>
              </OuiText>
            </OuiStack>

            <!-- Cost Summary -->
            <OuiCard variant="default" class="mt-4">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText
                    size="lg"
                    weight="semibold"
                    color="primary"
                    class="text-center"
                  >
                    Estimated Monthly Cost
                  </OuiText>
                  <OuiText
                    size="3xl"
                    weight="bold"
                    color="primary"
                    class="text-center"
                  >
                    {{ formatCurrency(totalMonthlyCost) }}
                  </OuiText>
                  <OuiGrid cols="2" cols-md="4" gap="md" class="mt-4">
                    <OuiStack gap="xs" align="center">
                      <OuiText size="sm" color="secondary">Memory</OuiText>
                      <OuiText size="md" weight="semibold">{{
                        formatCurrency(memoryCostMonthly)
                      }}</OuiText>
                    </OuiStack>
                    <OuiStack gap="xs" align="center">
                      <OuiText size="sm" color="secondary">vCPU</OuiText>
                      <OuiText size="md" weight="semibold">{{
                        formatCurrency(cpuCostMonthly)
                      }}</OuiText>
                    </OuiStack>
                    <OuiStack gap="xs" align="center">
                      <OuiText size="sm" color="secondary">Bandwidth</OuiText>
                      <OuiText size="md" weight="semibold">{{
                        formatCurrency(bandwidthCostMonthly)
                      }}</OuiText>
                    </OuiStack>
                    <OuiStack gap="xs" align="center">
                      <OuiText size="sm" color="secondary">Storage</OuiText>
                      <OuiText size="md" weight="semibold">{{
                        formatCurrency(storageCostMonthly)
                      }}</OuiText>
                    </OuiStack>
                  </OuiGrid>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>
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
  import { InformationCircleIcon, CheckIcon } from "@heroicons/vue/24/outline";
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
        cpuCostPerCoreSecond: response.cpuCostPerCoreSecond,
        memoryCostPerByteSecond: response.memoryCostPerByteSecond,
        bandwidthCostPerByte: response.bandwidthCostPerByte,
        storageCostPerByteMonth: response.storageCostPerByteMonth,
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
    small: { memory: 0.5, cpu: 0.25, bandwidth: 10, storage: 5 },
    medium: { memory: 2, cpu: 1, bandwidth: 50, storage: 25 },
    large: { memory: 8, cpu: 2, bandwidth: 200, storage: 100 },
  };

  const scenarioOptions = [
    { label: "Small App", value: "small" },
    { label: "Medium App", value: "medium" },
    { label: "Large App", value: "large" },
    { label: "Custom", value: "custom" },
  ];

  const selectedScenario = ref("medium");

  // Slider values
  const memorySliderValue = ref([2]);
  const cpuSliderValue = ref([1]);
  const bandwidthSliderValue = ref([50]);
  const storageSliderValue = ref([25]);

  // Watch scenario changes
  watch(selectedScenario, (newScenario) => {
    if (
      newScenario !== "custom" &&
      scenarios[newScenario as keyof typeof scenarios]
    ) {
      const scenario = scenarios[newScenario as keyof typeof scenarios];
      memorySliderValue.value = [scenario.memory];
      cpuSliderValue.value = [scenario.cpu];
      bandwidthSliderValue.value = [scenario.bandwidth];
      storageSliderValue.value = [scenario.storage];
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
    if (!pricing.value) return 0;
    const mem = memoryGB.value ?? 0;
    const gbHourByteSeconds = mem * GB_TO_BYTES * 3600;
    return (
      gbHourByteSeconds *
      HOURS_PER_MONTH *
      pricing.value.memoryCostPerByteSecond
    );
  });

  const cpuCostMonthly = computed(() => {
    if (!pricing.value) return 0;
    const cpu = cpuCores.value ?? 0;
    return cpu * HOURS_PER_MONTH * 3600 * pricing.value.cpuCostPerCoreSecond;
  });

  const bandwidthCostMonthly = computed(() => {
    if (!pricing.value) return 0;
    const bw = bandwidthGB.value ?? 0;
    return bw * GB_TO_BYTES * pricing.value.bandwidthCostPerByte;
  });

  const storageCostMonthly = computed(() => {
    if (!pricing.value) return 0;
    const storage = storageGB.value ?? 0;
    return storage * GB_TO_BYTES * pricing.value.storageCostPerByteMonth;
  });

  const totalMonthlyCost = computed(() => {
    return (
      memoryCostMonthly.value +
      cpuCostMonthly.value +
      bandwidthCostMonthly.value +
      storageCostMonthly.value
    );
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
