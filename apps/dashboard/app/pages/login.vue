<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center">
    <div class="max-w-md w-full">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="mx-auto w-16 h-16 bg-blue-600 rounded-xl flex items-center justify-center mb-4">
          <span class="text-white font-bold text-2xl">O</span>
        </div>
        <h1 class="text-3xl font-bold text-gray-900">Obiente Cloud</h1>
        <p class="text-gray-600 mt-2">Sign in to your account</p>
      </div>
      
      <!-- Login Card -->
      <div class="card">
        <div class="card-body">
          <form @submit.prevent="handleLogin" class="space-y-6">
            <!-- Email -->
            <div class="form-group">
              <label for="email" class="form-label">Email address</label>
              <input
                id="email"
                v-model="form.email"
                type="email"
                required
                class="form-input"
                placeholder="Enter your email"
              >
              <p v-if="errors.email" class="form-error">{{ errors.email }}</p>
            </div>
            
            <!-- Password -->
            <div class="form-group">
              <label for="password" class="form-label">Password</label>
              <input
                id="password"
                v-model="form.password"
                type="password"
                required
                class="form-input"
                placeholder="Enter your password"
              >
              <p v-if="errors.password" class="form-error">{{ errors.password }}</p>
            </div>
            
            <!-- Remember me -->
            <div class="flex items-center justify-between">
              <div class="flex items-center">
                <input
                  id="remember"
                  v-model="form.remember"
                  type="checkbox"
                  class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                >
                <label for="remember" class="ml-2 block text-sm text-gray-700">
                  Remember me
                </label>
              </div>
              
              <div class="text-sm">
                <a href="#" class="font-medium text-blue-600 hover:text-blue-500">
                  Forgot your password?
                </a>
              </div>
            </div>
            
            <!-- Submit button -->
            <button
              type="submit"
              :disabled="isLoading"
              class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <span v-if="isLoading" class="flex items-center">
                <svg class="animate-spin -ml-1 mr-3 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Signing in...
              </span>
              <span v-else>Sign in</span>
            </button>
          </form>
          
          <!-- Or divider -->
          <div class="mt-6">
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-300" />
              </div>
              <div class="relative flex justify-center text-sm">
                <span class="px-2 bg-white text-gray-500">Or continue with</span>
              </div>
            </div>
          </div>
          
          <!-- SSO Button -->
          <div class="mt-6">
            <button
              @click="handleSsoLogin"
              class="w-full flex justify-center items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              <svg class="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
              </svg>
              Sign in with SSO
            </button>
          </div>
        </div>
      </div>
      
      <!-- Sign up link -->
      <div class="text-center mt-6">
        <p class="text-sm text-gray-600">
          Don't have an account?
          <a href="#" class="font-medium text-blue-600 hover:text-blue-500">
            Contact us for access
          </a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Page meta
definePageMeta({
  layout: false, // Use no layout for login page
});

// Form state
const form = ref({
  email: '',
  password: '',
  remember: false,
});

const errors = ref({
  email: '',
  password: '',
});

const isLoading = ref(false);

import { useUser } from '~/stores/user';
const userStore = useUser();

// Handle form login
const handleLogin = async () => {
  try {
    isLoading.value = true;
    errors.value = { email: '', password: '' };
    
    // Basic validation
    if (!form.value.email) {
      errors.value.email = 'Email is required';
      return;
    }
    
    if (!form.value.password) {
      errors.value.password = 'Password is required';
      return;
    }
    
    // TODO: Implement actual login logic
    console.log('Login attempt:', form.value);
    
    // Redirect to dashboard
    await navigateTo('/dashboard');
  } catch (error) {
    console.error('Login failed:', error);
    // Handle login error
  } finally {
    isLoading.value = false;
  }
};

// Handle SSO login
const handleSsoLogin = async () => {
  try {
    await userStore.login();
  } catch (error) {
    console.error('SSO login failed:', error);
  }
};
</script>