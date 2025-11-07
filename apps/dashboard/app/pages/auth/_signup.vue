<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-base p-4">
    <div class="w-full max-w-md">
      <!-- Logo and Header -->
      <OuiStack gap="lg" align="center" class="mb-8">
        <ObienteLogo size="lg" class="shadow-lg" />
        <OuiStack gap="xs" align="center">
          <OuiText size="3xl" weight="bold" color="primary">Create Account</OuiText>
          <OuiText size="md" color="secondary">Sign up to get started with Obiente Cloud</OuiText>
        </OuiStack>
      </OuiStack>

      <!-- Signup Card -->
      <OuiCard variant="raised" class="shadow-xl">
        <OuiCardBody>
          <OuiStack gap="lg">
            <!-- Error Message -->
            <OuiCard
              v-if="error"
              variant="outline"
              status="danger"
              class="border-danger bg-danger/10"
            >
              <OuiCardBody>
                <OuiFlex align="center" gap="sm">
                  <ExclamationCircleIcon class="h-5 w-5 text-danger shrink-0" />
                  <OuiText size="sm" color="danger">{{ error }}</OuiText>
              </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Success Message -->
            <OuiCard
              v-if="success"
              variant="outline"
              status="success"
              class="border-success bg-success/10"
            >
              <OuiCardBody>
                <OuiFlex align="center" gap="sm">
                  <CheckCircleIcon class="h-5 w-5 text-success shrink-0" />
                  <OuiText size="sm" color="success">
                    Account created successfully! Redirecting...
                  </OuiText>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Signup Form -->
            <form @submit.prevent="handleSignup" class="space-y-4">
              <OuiStack gap="md">
                <!-- Name Input -->
                <OuiInput
                  v-model="name"
                  type="text"
                  label="Full Name"
                  placeholder="John Doe"
                  required
                  :error="errors.name"
                  :disabled="loading"
                  size="lg"
                >
                  <template #prefix>
                    <UserIcon class="h-5 w-5 text-text-secondary" />
                  </template>
                </OuiInput>

                <!-- Email Input -->
                <OuiInput
                  v-model="email"
                  type="email"
                  label="Email Address"
                  placeholder="you@example.com"
                  required
                  :error="errors.email"
                  :disabled="loading"
                  size="lg"
                >
                  <template #prefix>
                    <EnvelopeIcon class="h-5 w-5 text-text-secondary" />
                  </template>
                </OuiInput>

                <!-- Password Input -->
                <OuiInput
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  label="Password"
                  placeholder="Create a strong password"
                  required
                  :error="errors.password"
                  :disabled="loading"
                  size="lg"
                  helper-text="Must be at least 8 characters"
                >
                  <template #prefix>
                    <LockClosedIcon class="h-5 w-5 text-text-secondary" />
                  </template>
                  <template #suffix>
                    <button
                      type="button"
                      @click="showPassword = !showPassword"
                      class="text-text-secondary hover:text-primary transition-colors"
                      tabindex="-1"
                    >
                      <EyeIcon v-if="!showPassword" class="h-5 w-5" />
                      <EyeSlashIcon v-else class="h-5 w-5" />
                    </button>
                  </template>
                </OuiInput>

                <!-- Confirm Password Input -->
                <OuiInput
                  v-model="confirmPassword"
                  :type="showConfirmPassword ? 'text' : 'password'"
                  label="Confirm Password"
                  placeholder="Confirm your password"
                  required
                  :error="errors.confirmPassword"
                  :disabled="loading"
                  size="lg"
                >
                  <template #prefix>
                    <LockClosedIcon class="h-5 w-5 text-text-secondary" />
                  </template>
                  <template #suffix>
                    <button
                      type="button"
                      @click="showConfirmPassword = !showConfirmPassword"
                      class="text-text-secondary hover:text-primary transition-colors"
                      tabindex="-1"
                    >
                      <EyeIcon v-if="!showConfirmPassword" class="h-5 w-5" />
                      <EyeSlashIcon v-else class="h-5 w-5" />
                    </button>
                  </template>
                </OuiInput>

                <!-- Terms and Conditions -->
                <div>
                  <OuiCheckbox
                    v-model="acceptedTerms"
                    :disabled="loading"
                  >
                    <OuiText size="sm" color="secondary">
                      I agree to the
                      <NuxtLink
                        to="/terms"
                        class="text-primary hover:text-accent-primary transition-colors"
                        target="_blank"
                      >
                        Terms of Service
                      </NuxtLink>
                      and
                      <NuxtLink
                        to="/privacy"
                        class="text-primary hover:text-accent-primary transition-colors"
                        target="_blank"
                      >
                        Privacy Policy
                      </NuxtLink>
                    </OuiText>
                  </OuiCheckbox>
                  <OuiText v-if="errors.terms" size="sm" color="danger" class="mt-1">
                    {{ errors.terms }}
                  </OuiText>
                </div>

                <!-- Submit Button -->
                <OuiButton
                  type="submit"
                  size="lg"
                  block
                  :loading="loading"
                  :disabled="loading || !isFormValid"
                  class="mt-6"
                >
                  <template v-if="!loading">
                    <UserPlusIcon class="h-5 w-5 mr-2" />
                    Create Account
                  </template>
                  <template v-else>
                    Creating account...
                  </template>
                </OuiButton>
              </OuiStack>
            </form>

            <!-- Divider -->
            <OuiFlex align="center" gap="md" class="my-4">
              <div class="flex-1 h-px bg-border-muted"></div>
              <OuiText size="sm" color="secondary">OR</OuiText>
              <div class="flex-1 h-px bg-border-muted"></div>
            </OuiFlex>

            <!-- Sign In Link -->
            <OuiStack gap="sm" align="center">
              <OuiText size="sm" color="secondary">
                Already have an account?
              </OuiText>
              <NuxtLink
                to="/auth/login"
                class="text-sm font-medium text-primary hover:text-accent-primary transition-colors"
              >
                Sign in instead
              </NuxtLink>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Footer -->
      <OuiStack gap="xs" align="center" class="mt-8">
        <OuiText size="xs" color="muted" class="text-center">
          By creating an account, you agree to our Terms of Service and Privacy Policy
        </OuiText>
      </OuiStack>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import {
  EnvelopeIcon,
  LockClosedIcon,
  EyeIcon,
  EyeSlashIcon,
  UserIcon,
  UserPlusIcon,
  ExclamationCircleIcon,
  CheckCircleIcon,
} from "@heroicons/vue/24/outline";
import { useAuth } from "~/composables/useAuth";
import ObienteLogo from "~/components/app/ObienteLogo.vue";

// Page meta
definePageMeta({
  layout: false,
  auth: false,
});

// SEO
useHead({
  title: "Sign Up - Obiente Cloud",
  meta: [
    {
      name: "description",
      content: "Create your Obiente Cloud account",
    },
  ],
});

// Form state
const name = ref("");
const email = ref("");
const password = ref("");
const confirmPassword = ref("");
const acceptedTerms = ref(false);
const showPassword = ref(false);
const showConfirmPassword = ref(false);
const loading = ref(false);
const error = ref("");
const success = ref(false);

// Form validation
const errors = computed(() => ({
  name: name.value && name.value.length < 2
    ? "Name must be at least 2 characters"
    : "",
  email: email.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)
    ? "Please enter a valid email address"
    : "",
  password: password.value && password.value.length < 8
    ? "Password must be at least 8 characters"
    : password.value && !/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(password.value)
    ? "Password must contain uppercase, lowercase, and a number"
    : "",
  confirmPassword:
    confirmPassword.value && password.value !== confirmPassword.value
      ? "Passwords do not match"
      : "",
  terms: !acceptedTerms.value ? "You must accept the terms and conditions" : "",
}));

const isFormValid = computed(() => {
  return (
    name.value &&
    email.value &&
    password.value &&
    confirmPassword.value &&
    acceptedTerms.value &&
    !errors.value.name &&
    !errors.value.email &&
    !errors.value.password &&
    !errors.value.confirmPassword
  );
});

// Auth composable
const auth = useAuth();

// Handle signup
const handleSignup = async () => {
  // Clear previous errors
  error.value = "";
  success.value = false;

  // Validate form
  if (!isFormValid.value) {
    error.value = "Please fix the errors above";
    return;
  }

  loading.value = true;

  try {
    // Call signup API
    const response = await $fetch<{ success: boolean; message?: string }>(
      "/api/auth/signup",
      {
        method: "POST",
        body: {
          name: name.value,
          email: email.value,
          password: password.value,
        },
      }
    );

    if (response.success) {
      success.value = true;
      
      // Refresh auth state
      await auth.fetch();

      // Redirect to dashboard after a short delay
      setTimeout(async () => {
        const returnTo = useRoute().query.returnTo as string | undefined;
        await navigateTo(returnTo || "/dashboard");
      }, 1500);
    } else {
      error.value = response.message || "Failed to create account. Please try again.";
    }
  } catch (err: any) {
    console.error("Signup error:", err);
    error.value =
      err.data?.message ||
      err.message ||
      "An error occurred. Please try again.";
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
/* Additional custom styles if needed */
</style>

