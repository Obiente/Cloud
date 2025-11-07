<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-base p-4">
    <div class="w-full max-w-md">
      <!-- Logo and Header -->
      <OuiStack gap="lg" align="center" class="mb-8">
        <ObienteLogo size="lg" class="shadow-lg" />
        <OuiStack gap="xs" align="center">
          <OuiText size="3xl" weight="bold" color="primary">Welcome Back</OuiText>
          <OuiText size="md" color="secondary">Sign in to your account to continue</OuiText>
        </OuiStack>
      </OuiStack>

      <!-- Login Card -->
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

            <!-- Login Form -->
            <form @submit.prevent="handleLogin" class="space-y-4">
              <OuiStack gap="md">
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
                  placeholder="Enter your password"
                  required
                  :error="errors.password"
                  :disabled="loading"
                  size="lg"
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

                <!-- Remember Me & Forgot Password -->
                <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                  <OuiCheckbox v-model="rememberMe" :disabled="loading">
                    Remember me
                  </OuiCheckbox>
                  <NuxtLink
                    to="/auth/forgot-password"
                    class="text-sm text-primary hover:text-accent-primary transition-colors"
                  >
                    Forgot password?
                  </NuxtLink>
                </OuiFlex>

                <!-- Submit Button -->
                <OuiButton
                  type="submit"
                  size="lg"
                  block
                  :loading="loading"
                  :disabled="loading || !email || !password"
                  class="mt-6"
                >
                  <template v-if="!loading">
                    <ArrowRightOnRectangleIcon class="h-5 w-5 mr-2" />
                    Sign In
                  </template>
                  <template v-else>
                    Signing in...
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

            <!-- Sign Up Link -->
            <OuiStack gap="sm" align="center">
              <OuiText size="sm" color="secondary">
                Don't have an account?
              </OuiText>
              <button
                @click="auth.popupSignup()"
                class="text-sm font-medium text-primary hover:text-accent-primary transition-colors cursor-pointer"
              >
                Create an account
              </button>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Footer -->
      <OuiStack gap="xs" align="center" class="mt-8">
        <OuiText size="xs" color="muted" class="text-center">
          By signing in, you agree to our Terms of Service and Privacy Policy
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
  ArrowRightOnRectangleIcon,
  ExclamationCircleIcon,
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
  title: "Sign In - Obiente Cloud",
  meta: [
    {
      name: "description",
      content: "Sign in to your Obiente Cloud account",
    },
  ],
});

// Form state
const email = ref("");
const password = ref("");
const rememberMe = ref(false);
const showPassword = ref(false);
const loading = ref(false);
const error = ref("");

// Form errors
const errors = computed(() => ({
  email: email.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)
    ? "Please enter a valid email address"
    : "",
  password: password.value && password.value.length < 8
    ? "Password must be at least 8 characters"
    : "",
}));

// Auth composable
const auth = useAuth();

// Handle login
const handleLogin = async () => {
  // Clear previous errors
  error.value = "";

  // Validate form
  if (!email.value || !password.value) {
    error.value = "Please fill in all fields";
    return;
  }

  if (errors.value.email || errors.value.password) {
    error.value = "Please fix the errors above";
    return;
  }

  loading.value = true;

  try {
    // Call login API (which calls backend API with service account)
    const response = await $fetch<{ 
      success: boolean; 
      message?: string;
    }>(
      "/api/auth/login",
      {
        method: "POST",
        body: {
          email: email.value,
          password: password.value,
          rememberMe: rememberMe.value,
        },
      }
    );

    if (response.success) {
      // Refresh auth state
      await auth.fetch();

      // Redirect to dashboard or return URL
      const returnTo = useRoute().query.returnTo as string | undefined;
      await navigateTo(returnTo || "/dashboard");
    } else {
      error.value = response.message || "Invalid email or password";
    }
  } catch (err: any) {
    console.error("Login error:", err);
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

