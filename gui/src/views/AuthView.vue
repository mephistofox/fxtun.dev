<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { Button, Card, Input, Label, Tabs, Alert, AlertDescription } from '@/components/ui'
import { Wifi, Eye, EyeOff, Server, Key, Phone, Lock, Shield } from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const authMethod = ref<'token' | 'password'>('token')
const serverAddress = ref('')
const token = ref('')
const phone = ref('')
const password = ref('')
const totpCode = ref('')
const remember = ref(true)
const showPassword = ref(false)
const showError = ref(false)

const tabs = computed(() => [
  { value: 'token', label: t('auth.loginWithToken') },
  { value: 'password', label: t('auth.loginWithPassword') },
])

// Reset TOTP state when switching auth method
watch(authMethod, () => {
  authStore.resetTotpRequired()
  totpCode.value = ''
  showError.value = false
})

// Show error animation
watch(() => authStore.error, (error) => {
  if (error) {
    showError.value = true
    setTimeout(() => { showError.value = false }, 500)
  }
})

onMounted(async () => {
  await settingsStore.init()
  serverAddress.value = settingsStore.serverAddress || 'localhost:4443'
})

async function handleSubmit() {
  let success = false

  if (authMethod.value === 'token') {
    success = await authStore.loginWithToken(
      serverAddress.value,
      token.value,
      remember.value
    )
  } else {
    success = await authStore.loginWithPassword(
      serverAddress.value,
      phone.value,
      password.value,
      totpCode.value || undefined,
      remember.value
    )
  }

  if (success) {
    await settingsStore.saveServerAddress(serverAddress.value)
    router.push('/dashboard')
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gradient-to-br from-background via-muted/30 to-background p-4">
    <Card
      :class="['w-full max-w-md overflow-hidden', showError && 'animate-shake']"
    >
      <!-- Header with gradient -->
      <div class="bg-gradient-to-r from-primary/10 via-primary/5 to-transparent p-6 pb-4">
        <div class="mb-2 flex items-center justify-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 shadow-sm">
            <Wifi class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h1 class="text-2xl font-bold tracking-tight">{{ t('auth.title') }}</h1>
            <p class="text-sm text-muted-foreground">{{ t('auth.subtitle') }}</p>
          </div>
        </div>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4 p-6 pt-4">
        <!-- Server Address -->
        <div class="space-y-2">
          <Label for="server" class="flex items-center gap-2">
            <Server class="h-3.5 w-3.5 text-muted-foreground" />
            {{ t('auth.serverAddress') }}
          </Label>
          <Input
            id="server"
            v-model="serverAddress"
            :placeholder="t('auth.serverAddressPlaceholder')"
          />
        </div>

        <!-- Auth Method Tabs -->
        <Tabs
          v-model="authMethod"
          :tabs="tabs"
          class="w-full"
        />

        <!-- Token Auth -->
        <Transition name="fade" mode="out-in">
          <div v-if="authMethod === 'token'" key="token" class="space-y-4">
            <div class="space-y-2">
              <Label for="token" class="flex items-center gap-2">
                <Key class="h-3.5 w-3.5 text-muted-foreground" />
                {{ t('auth.token') }}
              </Label>
              <Input
                id="token"
                v-model="token"
                type="password"
                :placeholder="t('auth.tokenPlaceholder')"
              />
            </div>
          </div>

          <!-- Password Auth -->
          <div v-else key="password" class="space-y-4">
            <div class="space-y-2">
              <Label for="phone" class="flex items-center gap-2">
                <Phone class="h-3.5 w-3.5 text-muted-foreground" />
                {{ t('auth.phone') }}
              </Label>
              <Input
                id="phone"
                v-model="phone"
                phone
                :placeholder="t('auth.phonePlaceholder')"
              />
            </div>

            <div class="space-y-2">
              <Label for="password" class="flex items-center gap-2">
                <Lock class="h-3.5 w-3.5 text-muted-foreground" />
                {{ t('auth.password') }}
              </Label>
              <div class="relative">
                <Input
                  id="password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  class="pr-10"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                  @click="showPassword = !showPassword"
                >
                  <component :is="showPassword ? EyeOff : Eye" class="h-4 w-4" />
                </button>
              </div>
            </div>

            <Transition name="slide-up">
              <div v-if="authStore.totpRequired" class="space-y-2">
                <Label for="totp" class="flex items-center gap-2">
                  <Shield class="h-3.5 w-3.5 text-muted-foreground" />
                  {{ t('auth.totpCode') }}
                </Label>
                <Input
                  id="totp"
                  v-model="totpCode"
                  :placeholder="t('auth.totpPlaceholder')"
                  maxlength="6"
                  required
                  class="text-center font-mono text-lg tracking-widest"
                />
                <p class="text-xs text-muted-foreground">{{ t('auth.totpHint') }}</p>
              </div>
            </Transition>
          </div>
        </Transition>

        <!-- Remember Me -->
        <div class="flex items-center gap-2">
          <input
            id="remember"
            v-model="remember"
            type="checkbox"
            class="h-4 w-4 rounded border-input accent-primary"
          />
          <Label for="remember" class="cursor-pointer text-sm">{{ t('auth.remember') }}</Label>
        </div>

        <!-- Error -->
        <Transition name="slide-up">
          <Alert v-if="authStore.error" variant="destructive">
            <AlertDescription>{{ authStore.error }}</AlertDescription>
          </Alert>
        </Transition>

        <!-- Submit -->
        <Button
          type="submit"
          class="w-full"
          :loading="authStore.isLoading"
          :disabled="authStore.isLoading"
        >
          {{ authStore.isLoading ? t('auth.connecting') : t('auth.login') }}
        </Button>
      </form>
    </Card>
  </div>
</template>
