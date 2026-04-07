<template>
  <div>
    <div class="flex items-center gap-1.5 flex-wrap">
      <NuxtLink
        v-if="organizationId"
        :to="`/superadmin/organizations/${organizationId}`"
        class="font-medium text-text-primary hover:text-primary transition-colors"
        @click.stop
      >
        {{ displayName }}
      </NuxtLink>
      <span v-else class="font-medium text-text-primary">{{ displayName }}</span>
      <span
        v-if="isPersonal"
        class="inline-flex items-center rounded px-1 py-0.5 text-[10px] font-medium bg-primary/10 text-primary leading-none"
      >Personal</span>
    </div>
    <div v-if="organizationId" class="text-xs font-mono text-text-tertiary mt-0.5">
      {{ organizationId }}
    </div>
    <div v-if="ownerName || ownerId" class="text-xs text-text-muted mt-0.5 flex items-center gap-1">
      <span class="text-text-tertiary">Owner:</span>
      <NuxtLink
        v-if="ownerId"
        :to="`/superadmin/users/${ownerId}`"
        class="text-primary hover:underline truncate max-w-[160px]"
        :title="ownerName || ownerId"
        @click.stop
      >{{ ownerName || ownerId }}</NuxtLink>
      <span v-else class="truncate max-w-[160px]" :title="ownerName">{{ ownerName }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  organizationName?: string;
  organizationId?: string;
  ownerName?: string;
  ownerId?: string;
  /** org plan — used to detect personal orgs ("personal") */
  plan?: string;
  /** org slug — used to detect personal orgs ("personal-*") */
  slug?: string;
}>();

const isPersonal = computed(
  () =>
    props.plan === "personal" ||
    (props.slug != null && props.slug.startsWith("personal-")) ||
    props.organizationName === "Personal"
);

const displayName = computed(() => {
  if (isPersonal.value && props.ownerName) {
    return `${props.organizationName || "Personal"} (${props.ownerName})`;
  }
  return props.organizationName || props.organizationId || "—";
});
</script>

