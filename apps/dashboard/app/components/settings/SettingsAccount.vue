<template>
  <OuiStack gap="lg">
      <!-- Profile Information -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiText size="sm" weight="semibold">Profile Information</OuiText>

            <!-- Loading State -->
            <OuiText v-if="!auth.user" size="sm" color="tertiary">
              Loading account information...
            </OuiText>

            <!-- Editable Form -->
            <OuiStack v-else gap="md">
              <OuiInput
                v-model="formData.preferred_username"
                label="Username"
                placeholder="Enter username"
                :error="errors.preferred_username"
              />

              <OuiFlex gap="sm">
                <OuiInput
                  v-model="formData.given_name"
                  label="First Name"
                  placeholder="Enter first name"
                  :error="errors.given_name"
                  class="flex-1"
                />
                <OuiInput
                  v-model="formData.family_name"
                  label="Last Name"
                  placeholder="Enter last name"
                  :error="errors.family_name"
                  class="flex-1"
                />
              </OuiFlex>

              <OuiInput
                v-model="formData.name"
                label="Display Name"
                placeholder="Enter display name"
                :error="errors.name"
              />

              <OuiInput
                v-model="formData.locale"
                label="Locale"
                placeholder="en"
                helper-text="Language preference (e.g., en, de, fr)"
                :error="errors.locale"
              />

              <!-- Read-only fields -->
              <OuiStack gap="none" class="divide-y divide-border-default pt-4 border-t border-border-muted">
                <OuiFlex align="center" justify="between" gap="sm" class="py-2">
                  <OuiText size="xs" color="tertiary">Email</OuiText>
                  <OuiFlex align="center" gap="sm">
                    <OuiText size="sm" weight="medium">{{ auth.user.email || "—" }}</OuiText>
                    <OuiBadge
                      v-if="auth.user.email_verified !== undefined"
                      :variant="auth.user.email_verified ? 'success' : 'warning'"
                      size="xs"
                    >
                      {{ auth.user.email_verified ? "Verified" : "Not verified" }}
                    </OuiBadge>
                  </OuiFlex>
                </OuiFlex>
                <OuiFlex align="center" justify="between" gap="sm" class="py-2">
                  <OuiText size="xs" color="tertiary">User ID</OuiText>
                  <OuiText size="xs" color="tertiary" class="font-mono">{{ auth.user.sub }}</OuiText>
                </OuiFlex>
              </OuiStack>

              <!-- Action Buttons -->
              <OuiFlex gap="sm" align="center" class="pt-4 border-t border-border-muted">
                <OuiButton
                  variant="solid"
                  @click="saveProfile"
                  :disabled="isSaving || !hasChanges"
                >
                  <CheckIcon v-if="!isSaving" class="h-4 w-4 mr-2" />
                  <div v-else class="h-4 w-4 mr-2 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  {{ isSaving ? "Saving..." : "Save Changes" }}
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  @click="resetForm"
                  :disabled="isSaving || !hasChanges"
                >
                  Cancel
                </OuiButton>
                <OuiText v-if="saveError" size="xs" color="danger" class="ml-auto">
                  {{ saveError }}
                </OuiText>
              </OuiFlex>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Security -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiFlex v-if="auth.user" align="center" justify="between" gap="sm">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="semibold">Security</OuiText>
              <OuiText size="xs" color="tertiary">Password, two-factor authentication, and sessions</OuiText>
            </OuiStack>
            <OuiButton
              variant="outline"
              size="sm"
              @click="openManagementConsole"
              :disabled="!managementUrl"
            >
              <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
              Manage
            </OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ArrowTopRightOnSquareIcon, CheckIcon } from "@heroicons/vue/24/outline";
import type { UserSession } from "@obiente/types";

const auth = useAuth();
const config = useRuntimeConfig();

// Form state
const formData = ref({
  preferred_username: "",
  given_name: "",
  family_name: "",
  name: "",
  locale: "",
});

const errors = ref<Record<string, string>>({});
const isSaving = ref(false);
const saveError = ref("");

// Initialize form data from user
const initializeForm = () => {
  if (auth.user) {
    formData.value = {
      preferred_username: auth.user.preferred_username || "",
      given_name: auth.user.given_name || "",
      family_name: auth.user.family_name || "",
      name: auth.user.name || "",
      locale: auth.user.locale || "",
    };
    errors.value = {};
    saveError.value = "";
  }
};

// Watch for user changes
watch(() => auth.user, () => {
  initializeForm();
}, { immediate: true });

// Check if form has changes
const hasChanges = computed(() => {
  if (!auth.user) return false;
  
  return (
    formData.value.preferred_username !== (auth.user.preferred_username || "") ||
    formData.value.given_name !== (auth.user.given_name || "") ||
    formData.value.family_name !== (auth.user.family_name || "") ||
    formData.value.name !== (auth.user.name || "") ||
    formData.value.locale !== (auth.user.locale || "")
  );
});

// Reset form to original values
const resetForm = () => {
  initializeForm();
};

// Save profile
const saveProfile = async () => {
  if (!auth.user || !hasChanges.value) return;

  errors.value = {};
  saveError.value = "";
  isSaving.value = true;

  try {
    const { useConnectClient } = await import("~/lib/connect-client");
    const { AuthService, UpdateUserProfileRequestSchema } = await import("@obiente/proto");
    const { create } = await import("@bufbuild/protobuf");

    const client = useConnectClient(AuthService);

    const request = create(UpdateUserProfileRequestSchema, {
      preferredUsername: formData.value.preferred_username?.trim() || undefined,
      givenName: formData.value.given_name?.trim() || undefined,
      familyName: formData.value.family_name?.trim() || undefined,
      name: formData.value.name?.trim() || undefined,
      locale: formData.value.locale?.trim() || undefined,
    });

    const response = await client.updateUserProfile(request);

    // Update auth user state directly with the response data
    // This avoids refetching which might return cached data
    if (response.user && auth.session) {
      // Map the API response user to the User type format
      const updatedUser = {
        sub: response.user.id || auth.user.sub,
        name: response.user.name || auth.user.name,
        given_name: response.user.givenName || auth.user.given_name || "",
        family_name: response.user.familyName || auth.user.family_name || "",
        locale: response.user.locale || auth.user.locale || "",
        updated_at: response.user.updatedAt 
          ? (typeof response.user.updatedAt === 'object' && 'seconds' in response.user.updatedAt
              ? Number(response.user.updatedAt.seconds)
              : Math.floor(new Date(response.user.updatedAt as string | number | Date).getTime() / 1000))
          : auth.user.updated_at,
        preferred_username: response.user.preferredUsername || auth.user.preferred_username || "",
        email: response.user.email || auth.user.email,
        email_verified: response.user.emailVerified ?? auth.user.email_verified,
      };
      
      // Update session state directly
      const sessionState = useState<UserSession | null>("obiente-session", () => null);
      if (sessionState.value) {
        sessionState.value = {
          ...sessionState.value,
          user: updatedUser,
        };
      }
    }

    // Show success feedback
    const { showAlert } = useDialog();
    await showAlert({
      title: "Profile Updated",
      message: "Your profile has been successfully updated.",
    });
  } catch (err: unknown) {
    console.error("Failed to update profile:", err);
    saveError.value = (err as Error).message || "Failed to update profile. Please try again.";
    
    // Handle validation errors
    if ((err as any).code === "invalid_argument" || (err as Error).message?.includes("validation")) {
      // Could parse error details here if available
    }
  } finally {
    isSaving.value = false;
  }
};

// Construct Zitadel management console URL
const managementUrl = computed(() => {
  if (!auth.user) return null;
  
  const zitadelBase = config.public.oidcBase || "https://auth.obiente.cloud";
  const baseUrl = zitadelBase.replace(/\/$/, "");
  // Use self-service UI endpoint - doesn't require user ID in URL
  return `${baseUrl}/ui/console/users/me`;
});

const openManagementConsole = () => {
  if (managementUrl.value) {
    window.open(managementUrl.value, "_blank", "noopener,noreferrer");
  }
};
</script>
