<template>
  <div class="h-full overflow-y-auto p-6">
    <div class="max-w-4xl mx-auto">
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <div>
              <OuiText as="h2" size="lg" weight="semibold">
                Server Properties
              </OuiText>
              <OuiText size="sm" color="secondary" class="mt-1">
                Configure your Minecraft server settings
              </OuiText>
            </div>
            <OuiButton
              variant="solid"
              size="sm"
              :loading="isSaving"
              @click="handleSave"
            >
              <CheckCircleIcon class="h-4 w-4 mr-2" />
              Save Changes
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <!-- Render properties by category -->
            <div
              v-for="category in categories"
              :key="category.id"
              class="border-b border-border-default pb-6 last:border-b-0 last:pb-0"
            >
              <OuiText size="sm" weight="semibold" class="mb-4 block">
                {{ category.label }}
              </OuiText>
              <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                <template
                  v-for="prop in category.properties"
                  :key="prop.key"
                >
                  <!-- Conditional properties (e.g., rcon.port only shows when enable-rcon is true) -->
                  <template
                    v-if="!prop.dependsOn || checkDependency(prop.dependsOn)"
                  >
                    <!-- String Input -->
                    <OuiInput
                      v-if="prop.type === 'string'"
                      v-model="properties[prop.camelKey]"
                      :label="prop.label"
                      :helper-text="prop.hint"
                      :placeholder="String(prop.default || '')"
                      :type="prop.inputType || 'text'"
                    />

                    <!-- Number Input -->
                    <OuiInput
                      v-else-if="prop.type === 'number'"
                      v-model="properties[prop.camelKey]"
                      :label="prop.label"
                      :helper-text="prop.hint"
                      type="number"
                      :min="prop.validation?.min"
                      :max="prop.validation?.max"
                      :placeholder="String(prop.default || '')"
                    />

                    <!-- Boolean Switch -->
                    <OuiSwitch
                      v-else-if="prop.type === 'boolean'"
                      v-model="properties[prop.camelKey]"
                      :label="prop.label"
                      :helper-text="prop.hint"
                    />

                    <!-- Select Dropdown -->
                    <OuiSelect
                      v-else-if="prop.type === 'select'"
                      v-model="properties[prop.camelKey]"
                      :label="prop.label"
                      :helper-text="prop.hint"
                      :items="prop.options"
                    />

                    <!-- Password Input -->
                    <OuiInput
                      v-else-if="prop.type === 'password'"
                      v-model="properties[prop.camelKey]"
                      :label="prop.label"
                      :helper-text="prop.hint"
                      type="password"
                      :placeholder="String(prop.default || '')"
                    />
                  </template>
                </template>
              </OuiGrid>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { CheckCircleIcon } from "@heroicons/vue/24/outline";
import propertiesSchema from "~/data/minecraft-server-properties-schema.json";

interface Props {
  fileContent: string;
  isSaving?: boolean;
  serverVersion?: string; // Minecraft server version for filtering properties
}

interface Emits {
  (e: "save", content: string): void;
}

const props = withDefaults(defineProps<Props>(), {
  isSaving: false,
  serverVersion: undefined,
});

const emit = defineEmits<Emits>();

// Parse server.properties format
function parseProperties(content: string): Record<string, string> {
  const props: Record<string, string> = {};
  const lines = content.split("\n");

  for (const line of lines) {
    const trimmed = line.trim();
    // Skip comments and empty lines
    if (!trimmed || trimmed.startsWith("#")) continue;

    const match = trimmed.match(/^([^=]+)=(.*)$/);
    if (match && match[1] && match[2] !== undefined) {
      const key = match[1].trim();
      const value = match[2].trim();
      props[key] = value;
    }
  }

  return props;
}

// Convert properties back to server.properties format
function formatProperties(props: Record<string, any>): string {
  const lines: string[] = [];
  lines.push("#Minecraft server properties");
  lines.push(`#${new Date().toISOString()}`);
  lines.push("");

  // Get all properties from schema
  const allProps = propertiesSchema.properties.map((p) => ({
    ...p,
    camelKey: p.key.replace(/-([a-z])/g, (g) => g[1]?.toUpperCase() || ""),
  }));

  // Sort by category and key
  const sortedProps = allProps.sort((a, b) => {
    const categoryOrder = Object.keys(propertiesSchema.categories);
    const aCat = categoryOrder.indexOf(a.category);
    const bCat = categoryOrder.indexOf(b.category);
    if (aCat !== bCat) return aCat - bCat;
    return a.key.localeCompare(b.key);
  });

  // Write properties (excluding restricted properties managed by the platform)
  for (const prop of sortedProps) {
    // Skip restricted properties that are managed by the platform
    if (RESTRICTED_PROPERTIES.includes(prop.key)) {
      continue;
    }
    
    const value = props[prop.camelKey];
    if (value !== undefined && value !== null && value !== "") {
      const formattedValue =
        typeof value === "boolean" ? (value ? "true" : "false") : String(value);
      lines.push(`${prop.key}=${formattedValue}`);
    }
  }

  return lines.join("\n");
}

// Compare Minecraft versions (e.g., "1.19.2" vs "1.18")
function compareVersions(version1: string, version2: string): number {
  const v1Parts = version1.split(".").map(Number);
  const v2Parts = version2.split(".").map(Number);
  
  const maxLength = Math.max(v1Parts.length, v2Parts.length);
  
  for (let i = 0; i < maxLength; i++) {
    const v1Part = v1Parts[i] || 0;
    const v2Part = v2Parts[i] || 0;
    
    if (v1Part < v2Part) return -1;
    if (v1Part > v2Part) return 1;
  }
  
  return 0;
}

// Filter properties by version
function filterPropertiesByVersion(
  props: typeof propertiesSchema.properties,
  version?: string
) {
  if (!version) return props;

  return props.filter((prop) => {
    const minVersion = prop.versions?.min || "1.0";
    const maxVersion = prop.versions?.max || "latest";

    // Check minimum version
    if (compareVersions(version, minVersion) < 0) {
      return false;
    }

    // Check maximum version (if not "latest")
    if (maxVersion !== "latest") {
      if (compareVersions(version, maxVersion) > 0) {
        return false;
      }
    }

    return true;
  });
}

// Properties that are managed by the platform and should not be editable by users
const RESTRICTED_PROPERTIES = ["server-port", "server-ip"];

// Process schema and create categories
const processedProperties = computed(() => {
  const filtered = filterPropertiesByVersion(
    propertiesSchema.properties,
    props.serverVersion
  );

  // Filter out restricted properties that are managed by the platform
  return filtered
    .filter((prop) => !RESTRICTED_PROPERTIES.includes(prop.key))
    .map((prop) => ({
      ...prop,
      camelKey: prop.key.replace(/-([a-z])/g, (g) => g[1]?.toUpperCase() || ""),
    }));
});

const categories = computed(() => {
  const categoryMap = new Map<string, any>();

  for (const prop of processedProperties.value) {
    const categoryId = prop.category;
    if (!categoryMap.has(categoryId)) {
      const categoryInfo = propertiesSchema.categories[categoryId as keyof typeof propertiesSchema.categories];
      categoryMap.set(categoryId, {
        id: categoryId,
        label: categoryInfo?.label || categoryId,
        properties: [],
      });
    }
    categoryMap.get(categoryId)!.properties.push(prop);
  }

  return Array.from(categoryMap.values());
});

// Create reactive properties object
const properties = ref<Record<string, any>>({});

// Check if a property dependency is met
function checkDependency(dependsOn: Record<string, any>): boolean {
  for (const [key, expectedValue] of Object.entries(dependsOn)) {
    const camelKey = key.replace(/-([a-z])/g, (g) => g[1]?.toUpperCase() || "");
    const currentValue = properties.value[camelKey];
    if (currentValue !== expectedValue) {
      return false;
    }
  }
  return true;
}

// Initialize properties from file content
function loadProperties(content: string) {
  const parsed = parseProperties(content);
  const props: Record<string, any> = {};

  // Remove restricted properties from parsed content (they're managed by the platform)
  for (const restrictedKey of RESTRICTED_PROPERTIES) {
    delete parsed[restrictedKey];
  }

  // Initialize with defaults from schema first
  for (const prop of processedProperties.value) {
    const camelKey = prop.key.replace(/-([a-z])/g, (g) => g[1]?.toUpperCase() || "");
    const value = parsed[prop.key];
    
    if (value !== undefined && value !== null && value !== "") {
      // Convert based on type
      if (prop.type === "boolean") {
        props[camelKey] = value === "true";
      } else if (prop.type === "number") {
        props[camelKey] = Number(value) || 0;
      } else {
        props[camelKey] = value;
      }
    } else {
      // Use default from schema
      if (prop.type === "boolean") {
        props[camelKey] = prop.default === true || prop.default === "true";
      } else if (prop.type === "number") {
        props[camelKey] = Number(prop.default) || 0;
      } else {
        props[camelKey] = prop.default || "";
      }
    }
  }

  properties.value = props;
}

// Watch for file content changes
watch(
  () => props.fileContent,
  (newContent) => {
    loadProperties(newContent);
  },
  { immediate: true }
);

function handleSave() {
  const formatted = formatProperties(properties.value);
  emit("save", formatted);
}
</script>
