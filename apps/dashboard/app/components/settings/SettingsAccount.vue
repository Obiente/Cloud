<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <OuiText as="h2" size="lg" weight="semibold">Account Settings</OuiText>
      
      <!-- Account Information -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Profile Information</OuiText>
              <OuiText size="xs" color="secondary">
                Update your account profile information
              </OuiText>
            </OuiStack>

            <!-- Loading State -->
            <OuiText v-if="!auth.user" size="sm" color="secondary">
              Loading account information...
            </OuiText>

            <!-- Editable Form -->
            <OuiStack v-else gap="md">
              <OuiInput
                v-model="formData.preferred_username"
                label="Username"
                placeholder="Enter username"
                helper-text="Your preferred username"
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
                helper-text="This is how your name will appear to others"
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
              <div class="pt-2 border-t border-border-muted">
                <OuiStack gap="sm">
                  <OuiStack gap="xs">
                    <OuiText size="xs" weight="semibold" transform="uppercase" color="secondary" class="tracking-wide">
                      Email
                    </OuiText>
                    <OuiFlex align="center" gap="sm" wrap="wrap">
                      <OuiText size="sm" weight="medium">
                        {{ auth.user.email || "—" }}
                      </OuiText>
                      <OuiBox
                        v-if="auth.user.email_verified !== undefined"
                        px="xs"
                        py="xs"
                        rounded="sm"
                        :class="auth.user.email_verified 
                          ? 'bg-success/10 text-success' 
                          : 'bg-warning/10 text-warning'"
                        class="text-xs font-medium"
                      >
                        {{ auth.user.email_verified ? "Verified" : "Not verified" }}
                      </OuiBox>
                    </OuiFlex>
                    <OuiFlex v-if="auth.user" gap="sm" align="center" wrap="wrap" class="pt-1">
                      <OuiButton
                        variant="ghost"
                        size="sm"
                        @click="openManagementConsole"
                        :disabled="!managementUrl"
                      >
                        <ArrowTopRightOnSquareIcon class="h-3 w-3 mr-1" />
                        Update Email
                      </OuiButton>
                      <OuiText size="xs" color="secondary">
                        Opens in a new window
                      </OuiText>
                    </OuiFlex>
                  </OuiStack>

                  <OuiStack gap="xs">
                    <OuiText size="xs" weight="semibold" transform="uppercase" color="secondary" class="tracking-wide">
                      User ID
                    </OuiText>
                    <OuiText size="xs" color="secondary" class="font-mono break-all">
                      {{ auth.user.sub }}
                    </OuiText>
                  </OuiStack>
                </OuiStack>
              </div>

              <!-- Action Buttons -->
              <OuiFlex gap="sm" align="center" class="pt-2 border-t border-border-muted">
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

      <!-- Security Information -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Security & Authentication</OuiText>
              <OuiText size="xs" color="secondary">
                Change your password, manage two-factor authentication, and configure security settings
              </OuiText>
            </OuiStack>

            <OuiFlex v-if="auth.user" gap="sm" align="center" wrap="wrap">
              <OuiButton
                variant="solid"
                @click="openManagementConsole"
                class="w-full sm:w-auto"
                :disabled="!managementUrl"
              >
                <ArrowTopRightOnSquareIcon class="h-4 w-4 mr-2" />
                Manage Security Settings
              </OuiButton>
              <OuiText size="xs" color="secondary">
                Opens in a new window
              </OuiText>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ArrowTopRightOnSquareIcon, CheckIcon } from "@heroicons/vue/24/outline";

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

    // Update auth user state
    if (response.user) {
      await auth.fetch(); // Refresh user data
    }

    // Show success feedback
    const { showAlert } = useDialog();
    await showAlert({
      title: "Profile Updated",
      message: "Your profile has been successfully updated.",
    });
  } catch (err: any) {
    console.error("Failed to update profile:", err);
    saveError.value = err.message || "Failed to update profile. Please try again.";
    
    // Handle validation errors
    if (err.code === "invalid_argument" || err.message?.includes("validation")) {
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
