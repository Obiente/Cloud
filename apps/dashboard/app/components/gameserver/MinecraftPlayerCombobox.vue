<template>
  <Field.Root
    :invalid="!!error"
    :required="required"
    :disabled="disabled"
    class="oui-field space-y-1 w-full"
  >
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </Field.Label>

    <div class="relative w-full">
      <Field.Input
        :model-value="modelValue"
        @update:model-value="handleInput"
        :placeholder="placeholder"
        :disabled="disabled"
        :class="[inputClasses, avatarUrl ? 'pl-10' : '']"
        @blur="handleBlur"
      />
      
      <!-- Player avatar/head display -->
      <div
        v-if="avatarUrl"
        class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none"
      >
        <img
          :src="avatarUrl"
          :alt="modelValue || 'Player'"
          class="h-6 w-6 rounded"
          @error="handleImageError"
        />
      </div>
      
      <!-- Clear button -->
      <button
        v-if="showClear && modelValue"
        type="button"
        @click="handleClear"
        class="absolute inset-y-0 right-0 flex items-center pr-3 text-text-secondary hover:text-primary transition-colors"
      >
        <XMarkIcon class="h-4 w-4" />
      </button>
    </div>

    <Field.ErrorText v-if="error" class="text-sm text-danger">
      {{ error }}
    </Field.ErrorText>
    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">
      {{ helperText }}
    </Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Field } from "@ark-ui/vue/field";
import { XMarkIcon } from "@heroicons/vue/24/outline";
import { useMinecraftPlayer } from "~/composables/useMinecraftPlayer";
import { useDebounceFn } from "@vueuse/core";

interface Props {
  modelValue?: string;
  placeholder?: string;
  label?: string;
  helperText?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  showClear?: boolean;
  size?: "sm" | "md" | "lg";
}

const props = withDefaults(defineProps<Props>(), {
  showClear: true,
  size: "md",
  required: false,
  disabled: false,
  placeholder: "Enter Minecraft username...",
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const { getPlayerData } = useMinecraftPlayer();
const avatarUrl = ref<string | undefined>(undefined);
const isLoading = ref(false);

// Look up player when username changes (debounced)
const lookupPlayer = useDebounceFn(async (username: string) => {
  if (!username || username.trim().length === 0) {
    avatarUrl.value = undefined;
    return;
  }

  isLoading.value = true;
  try {
    const player = await getPlayerData(username.trim());
    if (player && player.avatarUrl) {
      avatarUrl.value = player.avatarUrl;
    } else {
      avatarUrl.value = undefined;
    }
  } catch (error) {
    console.error("[MinecraftPlayerInput] Failed to load player:", error);
    avatarUrl.value = undefined;
  } finally {
    isLoading.value = false;
  }
}, 800);

// Watch for changes to modelValue and look up player
watch(() => props.modelValue, (newValue) => {
  if (newValue && newValue.trim().length > 0) {
    lookupPlayer(newValue);
  } else {
    avatarUrl.value = undefined;
  }
}, { immediate: true });

const handleInput = (value: string) => {
  emit("update:modelValue", value);
  // Clear avatar while typing
  avatarUrl.value = undefined;
};

const handleBlur = () => {
  // Look up player when user finishes typing
  if (props.modelValue && props.modelValue.trim().length > 0) {
    lookupPlayer(props.modelValue);
  }
};

const handleClear = () => {
  emit("update:modelValue", "");
  avatarUrl.value = undefined;
};

const handleImageError = (event: Event) => {
  const img = event.target as HTMLImageElement;
  img.style.display = "none";
};

const inputClasses = computed(() => [
  "oui-input",
  `oui-input-${props.size}`,
  props.error ? "oui-input-error" : "oui-input-base",
]);
</script>

