<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { TooltipProvider } from 'radix-vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useTunnelsStore } from '@/stores/tunnels'
import Layout from '@/components/Layout.vue'
import { Button, Toaster } from '@/components/ui'
import { Loader2, ShieldX } from 'lucide-vue-next'
import { EventsOn } from '@/wailsjs/wailsjs/runtime/runtime'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const tunnelsStore = useTunnelsStore()

const initialized = ref(false)
const isAuthRoute = computed(() => route.name === 'auth')

onMounted(async () => {
  // Initialize settings store
  await settingsStore.init()
  tunnelsStore.init()

  // Apply theme
  if (settingsStore.theme === 'dark' ||
      (settingsStore.theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark')
  }

  // Subscribe to user_blocked event from backend
  EventsOn('user_blocked', () => {
    authStore.setBlocked()
  })

  initialized.value = true
})

// Watch for auth state changes (logout handling)
watch(() => authStore.isAuthenticated, (isAuth) => {
  if (!isAuth && route.meta.requiresAuth) {
    router.push('/auth')
  }
})

function handleBlockedLogout() {
  authStore.logout()
  router.push('/auth')
}
</script>

<template>
  <TooltipProvider>
    <div class="min-h-screen bg-background text-foreground">
      <!-- Blocked user overlay -->
      <div v-if="authStore.isBlocked" class="fixed inset-0 z-50 flex items-center justify-center bg-background">
        <div class="text-center space-y-4 max-w-md px-6">
          <div class="mx-auto h-16 w-16 rounded-2xl bg-destructive/10 flex items-center justify-center">
            <ShieldX class="h-8 w-8 text-destructive" />
          </div>
          <h1 class="text-2xl font-bold">{{ t('auth.accountBlocked') }}</h1>
          <p class="text-muted-foreground">{{ t('auth.accountBlockedHint') }}</p>
          <Button @click="handleBlockedLogout">{{ t('auth.backToLogin') }}</Button>
          <div>
            <a href="https://t.me/mephistofx" target="_blank" class="text-sm text-primary hover:underline">
              {{ t('auth.contactSupport') }}
            </a>
          </div>
        </div>
      </div>

      <!-- Loading state while initializing -->
      <template v-if="!initialized">
        <div class="flex min-h-screen items-center justify-center">
          <div class="flex items-center gap-3 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin" />
            <span>{{ t('common.loading') }}</span>
          </div>
        </div>
      </template>
      <!-- Auth page (no layout) -->
      <template v-else-if="isAuthRoute">
        <Transition name="fade" mode="out-in">
          <router-view />
        </Transition>
      </template>
      <!-- Main app with layout -->
      <template v-else>
        <Layout>
          <Transition name="fade" mode="out-in">
            <router-view />
          </Transition>
        </Layout>
      </template>

      <!-- Global Toast notifications -->
      <Toaster />
    </div>
  </TooltipProvider>
</template>
