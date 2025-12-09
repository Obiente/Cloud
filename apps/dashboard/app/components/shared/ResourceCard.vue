<template>
  <OuiCard
    :class="[
      'relative overflow-hidden cursor-pointer',
      statusMeta.cardClass,
      statusMeta.beforeGradient,
    ]"
    class="transition-all duration-500 ease-in-out"
    @click="handleClick"
  >
    <!-- Status Bar -->
    <div
      class="absolute top-0 left-0 right-0 h-1 bg-surface-muted/20 transition-colors duration-500"
    >
      <div
        :class="[
          'h-full transition-all duration-500 ease-in-out',
          statusMeta.barClass,
          isActioning && 'animate-pulse',
        ]"
        :style="{ width: '100%' }"
      />
      <!-- Action indicator overlay -->
      <div
        v-if="isActioning"
        class="absolute inset-0 h-full shimmer-animation"
        style="
          background: linear-gradient(
            90deg,
            transparent 0%,
            rgba(255, 255, 255, 0.4) 50%,
            transparent 100%
          );
          background-size: 200% 100%;
        "
      />
    </div>

    <OuiCardBody>
      <OuiStack gap="md">
        <!-- Loading Skeleton -->
        <template v-if="loading">
          <!-- Header Skeleton - Match actual structure with icon placeholder -->
          <OuiFlex justify="between" align="start">
            <OuiStack gap="xs" class="flex-1 min-w-0">
              <OuiFlex align="center" gap="sm">
                <!-- Icon placeholder - use actual icon if available, with random variation -->
                <component
                  v-if="statusMeta?.icon"
                  :is="statusMeta.icon"
                  :class="['h-5 w-5 shrink-0', statusMeta.iconClass]"
                  :style="{
                    opacity: skeletonVars.iconOpacity,
                    transform: `scale(${skeletonVars.iconScale})`,
                  }"
                />
                <slot name="icon">
                  <component
                    v-if="icon"
                    :is="icon"
                    :class="['h-5 w-5 shrink-0', iconClass]"
                    :style="{
                      opacity: skeletonVars.iconOpacity,
                      transform: `scale(${skeletonVars.iconScale})`,
                    }"
                  />
                </slot>
                <OuiSkeleton
                  v-if="!statusMeta?.icon && !icon"
                  width="1.25rem"
                  height="1.25rem"
                  variant="rectangle"
                  :rounded="true"
                />
                <OuiSkeleton
                  :width="skeletonVars.titleWidth"
                  height="1.5rem"
                  variant="text"
                />
              </OuiFlex>
              <slot name="subtitle">
                <OuiSkeleton
                  :width="skeletonVars.subtitleWidth"
                  height="1rem"
                  variant="text"
                />
              </slot>
            </OuiStack>
            <OuiFlex gap="xs">
              <slot name="actions" />
            </OuiFlex>
          </OuiFlex>

          <!-- Status Badge Skeleton - Use actual badge structure -->
          <OuiBadge variant="secondary" size="sm" class="opacity-30">
            <OuiSkeleton
              :width="randomTextWidthByType('short')"
              height="0.875rem"
              variant="text"
              class="bg-transparent"
            />
          </OuiBadge>

          <!-- Resources Skeleton - Use actual resource structure -->
          <slot name="resources">
            <OuiGrid :cols="gridCols" v-if="resources && resources.length > 0" gap="sm">
              <OuiBox
                v-for="(resource, idx) in resources"
                :key="idx"
                p="sm"
                rounded="lg"
                class="bg-surface-muted/40"
              >
                <OuiStack gap="xs" align="center">
                  <component
                    v-if="resource.icon"
                    :is="resource.icon"
                    class="h-4 w-4 text-secondary"
                    :style="{
                      opacity: skeletonVars.iconOpacity,
                      transform: `scale(${skeletonVars.iconScale})`,
                    }"
                  />
                  <OuiSkeleton
                    :width="randomTextWidthByType('label')"
                    height="0.875rem"
                    variant="text"
                  />
                </OuiStack>
              </OuiBox>
            </OuiGrid>
          </slot>

          <!-- Custom Info Section Skeleton -->
          <slot name="info" />

          <!-- Footer Skeleton - Match actual structure -->
          <OuiFlex justify="between" align="center">
            <slot name="footer-left">
              <OuiSkeleton
                :width="skeletonVars.subtitleWidth"
                height="0.875rem"
                variant="text"
              />
            </slot>
            <slot name="footer-right">
              <OuiSkeleton
                width="6rem"
                height="2rem"
                variant="rectangle"
                :rounded="true"
              />
            </slot>
          </OuiFlex>
        </template>

        <!-- Actual Content -->
        <template v-else>
          <!-- Header -->
          <OuiFlex justify="between" align="start">
            <OuiStack gap="xs" class="flex-1 min-w-0">
              <OuiFlex align="center" gap="sm">
                <component
                  v-if="statusMeta?.icon"
                  :is="statusMeta.icon"
                  :class="[
                    'h-5 w-5 shrink-0 transition-colors duration-500',
                    statusMeta.iconClass,
                  ]"
                />
                <slot name="icon">
                  <component
                    v-if="icon"
                    :is="icon"
                    :class="['h-5 w-5 shrink-0', iconClass]"
                  />
                </slot>
                <OuiText
                  as="h3"
                  size="lg"
                  weight="semibold"
                  class="truncate"
                  :title="title"
                >
                  {{ title }}
                </OuiText>
              </OuiFlex>
              <slot name="subtitle">
                <OuiText
                  v-if="subtitle"
                  size="sm"
                  color="secondary"
                  class="truncate"
                >
                  {{ subtitle }}
                </OuiText>
              </slot>
            </OuiStack>

            <OuiFlex gap="xs">
              <slot name="actions" />
            </OuiFlex>
          </OuiFlex>

          <!-- Status Badge -->
          <OuiBadge
            v-if="statusMeta"
            :variant="statusMeta.badge"
            size="sm"
            class="transition-all duration-300"
          >
            {{ statusMeta.label }}
          </OuiBadge>

          <!-- Resources / Custom Content -->
          <slot name="resources">
              <OuiGrid :cols="gridCols" v-if="resources && resources.length > 0" gap="sm">
              <OuiBox
                v-for="(resource, idx) in resources"
                :key="idx"
                p="sm"
                rounded="lg"
                class="bg-surface-muted/40"
              >
                <OuiStack gap="xs" align="center">
                  <component
                    v-if="resource.icon"
                    :is="resource.icon"
                    class="h-4 w-4 text-secondary"
                  />
                  <OuiText size="xs" weight="medium">{{
                    resource.label
                  }}</OuiText>
                </OuiStack>
              </OuiBox>
            </OuiGrid>
          </slot>

          <!-- Custom Info Section -->
          <slot name="info" />

          <!-- Footer -->
          <OuiFlex justify="between" align="center">
            <slot name="footer-left">
              <OuiText v-if="createdAt" size="xs" color="secondary">
                <OuiRelativeTime :value="createdAt" />
              </OuiText>
            </slot>
            <slot name="footer-right">
              <OuiButton
                v-if="detailUrl"
                variant="ghost"
                size="sm"
                @click.stop="navigateToDetail"
              >
                View Details
                <ArrowRightIcon class="h-4 w-4" />
              </OuiButton>
            </slot>
          </OuiFlex>
        </template>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { useRouter } from "vue-router";
  import { ArrowRightIcon } from "@heroicons/vue/24/outline";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
  import OuiSkeleton from "~/components/oui/Skeleton.vue";
  import OuiBadge from "~/components/oui/Badge.vue";
  import OuiGrid from "~/components/oui/Grid.vue";
  import OuiBox from "~/components/oui/Box.vue";
  import {
    useSkeletonVariations,
    randomTextWidthByType,
    randomIconVariation,
  } from "~/composables/useSkeletonVariations";

  interface Resource {
    icon?: any;
    label: string;
  }

  interface StatusMeta {
    badge: "success" | "danger" | "warning" | "secondary";
    label: string;
    cardClass: string;
    beforeGradient: string;
    barClass: string;
    icon?: any;
    iconClass: string;
  }

  interface Props {
    title?: string;
    subtitle?: string;
    statusMeta?: StatusMeta;
    icon?: any;
    iconClass?: string;
    resources?: Resource[];
    createdAt?: Date | string;
    detailUrl?: string;
    clickable?: boolean;
    progressValue?: number;
    isActioning?: boolean;
    loading?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    clickable: true,
    isActioning: false,
    loading: false,
    title: "",
    statusMeta: () => ({
      badge: "secondary" as const,
      label: "",
      cardClass: "",
      beforeGradient: "",
      barClass: "",
      iconClass: "",
    }),
  });

  const emit = defineEmits<{
    click: [];
  }>();

  const router = useRouter();

  // Generate random variations for skeleton (consistent per instance)
  const skeletonVars = useSkeletonVariations();

  const handleClick = () => {
    if (props.clickable) {
      if (props.detailUrl) {
        navigateToDetail();
      } else {
        emit("click");
      }
    }
  };

  const navigateToDetail = () => {
    if (props.detailUrl) {
      router.push(props.detailUrl);
    }
  };

  // Compute Grid columns in a type-safe way for OuiGrid
  const gridCols = computed(() => {
    const count = props.resources ? props.resources.length : 0;
    const sm = (count > 0 ? Math.min(4, count) : 1) as 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;
    return { base: 1 as const, sm };
  });
</script>

<style scoped>
  @keyframes shimmer {
    0% {
      background-position: -200% 0;
    }
    100% {
      background-position: 200% 0;
    }
  }

  .shimmer-animation {
    animation: shimmer 2s ease-in-out infinite;
  }
</style>
