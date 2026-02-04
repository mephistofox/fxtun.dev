<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'

const router = useRouter()
const authStore = useAuthStore()

const error = ref('')
const loading = ref(true)

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const errorParam = params.get('error')
  const accessToken = params.get('access_token')
  const refreshToken = params.get('refresh_token')

  if (errorParam) {
    error.value = errorParam
    loading.value = false
    return
  }

  if (accessToken && refreshToken) {
    localStorage.setItem('accessToken', accessToken)
    localStorage.setItem('refreshToken', refreshToken)
    await authStore.refreshProfile()
    authStore.initialized = true

    // Check for saved redirect (e.g., from pricing page)
    const savedRedirect = localStorage.getItem('authRedirect')
    if (savedRedirect) {
      localStorage.removeItem('authRedirect')
      router.replace(savedRedirect)
    } else {
      router.replace({ name: 'dashboard' })
    }
    return
  }

  error.value = 'Invalid callback parameters'
  loading.value = false
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center hero-gradient p-4">
    <Card variant="glass" class="w-full max-w-md p-8 animate-fade-in-up text-center">
      <div v-if="loading" class="space-y-4">
        <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mx-auto">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56" />
          </svg>
        </div>
        <p class="text-muted-foreground">Authorizing...</p>
      </div>

      <div v-else class="space-y-4">
        <div class="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mx-auto">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <p class="text-destructive font-medium">{{ error }}</p>
        <router-link to="/login" class="text-primary hover:underline text-sm">
          Back to login
        </router-link>
      </div>
    </Card>
  </div>
</template>
