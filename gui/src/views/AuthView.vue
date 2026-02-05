<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { Button, Card, Input, Label, Alert, AlertDescription } from '@/components/ui'
import { Wifi, Server, Key, Zap, ArrowRight, Github, Loader2 } from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const serverAddress = ref('')
const token = ref('')
const remember = ref(true)
const showError = ref(false)

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

async function handleOAuth(provider: string) {
  const success = await authStore.loginWithOAuth(
    serverAddress.value,
    provider,
    remember.value
  )
  if (success) {
    await settingsStore.saveServerAddress(serverAddress.value)
    router.push('/dashboard')
  }
}

async function handleSubmit() {
  const success = await authStore.loginWithToken(
    serverAddress.value,
    token.value,
    remember.value
  )
  if (success) {
    await settingsStore.saveServerAddress(serverAddress.value)
    router.push('/dashboard')
  }
}
</script>

<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-background p-4">
    <!-- Animated grid background -->
    <div class="absolute inset-0 grid-pattern opacity-40" />

    <!-- Glowing orbs -->
    <div class="absolute top-[-20%] right-[-10%] w-[500px] h-[500px] rounded-full bg-primary/20 blur-[120px] animate-float" />
    <div class="absolute bottom-[-20%] left-[-10%] w-[400px] h-[400px] rounded-full bg-accent/20 blur-[100px] animate-float-delayed" />

    <!-- Scanline effect -->
    <div class="absolute inset-0 pointer-events-none overflow-hidden">
      <div class="scanline" />
    </div>

    <!-- Main card -->
    <Card
      :class="[
        'relative z-10 w-full max-w-md overflow-hidden border-border/50 bg-card/80 backdrop-blur-xl',
        showError && 'animate-shake'
      ]"
    >
      <!-- Gradient border effect -->
      <div class="absolute inset-0 rounded-xl bg-gradient-to-br from-primary/20 via-transparent to-accent/20 pointer-events-none" />
      <div class="absolute inset-[1px] rounded-xl bg-card" />

      <div class="relative">
        <!-- Header -->
        <div class="relative p-8 pb-6 text-center">
          <!-- Logo with glow -->
          <div class="relative mx-auto mb-6 w-fit">
            <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-30 blur-xl animate-pulse" />
            <div class="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-accent shadow-2xl">
              <Zap class="h-10 w-10 text-primary-foreground" />
            </div>
          </div>

          <h1 class="font-display text-3xl font-bold tracking-tight">
            <span class="gradient-text">fxTunnel</span>
          </h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ t('auth.subtitle') }}</p>
        </div>

        <form @submit.prevent="handleSubmit" class="space-y-5 p-8 pt-0">
          <!-- Server Address -->
          <div class="space-y-2">
            <Label for="server" class="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
              <Server class="h-3.5 w-3.5" />
              {{ t('auth.serverAddress') }}
            </Label>
            <div class="relative">
              <Input
                id="server"
                v-model="serverAddress"
                :placeholder="t('auth.serverAddressPlaceholder')"
                class="bg-muted/30 border-border/50 focus:border-primary/50 focus:ring-primary/20 pl-4 pr-10 font-mono text-sm"
              />
              <Wifi class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/50" />
            </div>
          </div>

          <!-- OAuth Buttons -->
          <Transition name="fade" mode="out-in">
            <div v-if="authStore.oauthWaiting" class="flex flex-col items-center gap-3 py-4">
              <Loader2 class="h-6 w-6 animate-spin text-primary" />
              <p class="text-sm text-muted-foreground">{{ t('auth.oauthWaiting') }}</p>
              <button
                type="button"
                class="text-xs text-muted-foreground hover:text-foreground underline"
                @click="authStore.cancelOAuth()"
              >
                {{ t('auth.oauthCancel') }}
              </button>
            </div>
            <div v-else key="oauth-buttons" class="space-y-3">
              <div class="flex gap-3">
                <Button
                  type="button"
                  variant="outline"
                  class="flex-1 h-11 gap-2 border-border/50 bg-muted/30 hover:bg-muted/50"
                  :disabled="authStore.isLoading"
                  @click="handleOAuth('github')"
                >
                  <Github class="h-5 w-5" />
                  GitHub
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  class="flex-1 h-11 gap-2 border-border/50 bg-muted/30 hover:bg-muted/50"
                  :disabled="authStore.isLoading"
                  @click="handleOAuth('google')"
                >
                  <svg class="h-5 w-5" viewBox="0 0 24 24">
                    <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/>
                    <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                    <path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                    <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                  </svg>
                  Google
                </Button>
              </div>

              <!-- Divider -->
              <div class="relative">
                <div class="absolute inset-0 flex items-center">
                  <span class="w-full border-t border-border/50" />
                </div>
                <div class="relative flex justify-center text-xs uppercase">
                  <span class="bg-card px-2 text-muted-foreground">{{ t('auth.orConnectWithToken') }}</span>
                </div>
              </div>
            </div>
          </Transition>

          <!-- Token Auth -->
          <div class="space-y-2">
            <Label for="token" class="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
              <Key class="h-3.5 w-3.5" />
              {{ t('auth.token') }}
            </Label>
            <Input
              id="token"
              v-model="token"
              type="password"
              :placeholder="t('auth.tokenPlaceholder')"
              class="bg-muted/30 border-border/50 focus:border-primary/50 focus:ring-primary/20 font-mono text-sm"
            />
          </div>

          <!-- Remember Me -->
          <label class="flex items-center gap-3 cursor-pointer group">
            <div class="relative">
              <input
                id="remember"
                v-model="remember"
                type="checkbox"
                class="peer sr-only"
              />
              <div class="h-5 w-5 rounded-md border-2 border-border/50 bg-muted/30 transition-all peer-checked:border-primary peer-checked:bg-primary/20" />
              <svg
                class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 h-3 w-3 text-primary opacity-0 transition-opacity peer-checked:opacity-100"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="3"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <span class="text-sm text-muted-foreground group-hover:text-foreground transition-colors">
              {{ t('auth.remember') }}
            </span>
          </label>

          <!-- Error -->
          <Transition name="slide-up">
            <Alert v-if="authStore.error" variant="destructive" class="border-destructive/50 bg-destructive/10">
              <AlertDescription>{{ authStore.error }}</AlertDescription>
            </Alert>
          </Transition>

          <!-- Submit -->
          <Button
            type="submit"
            class="w-full h-12 text-base font-semibold bg-gradient-to-r from-primary to-primary hover:from-primary hover:to-accent transition-all duration-300 shadow-lg shadow-primary/25 hover:shadow-primary/40"
            :loading="authStore.isLoading"
            :disabled="authStore.isLoading"
          >
            <span class="flex items-center gap-2">
              {{ authStore.isLoading ? t('auth.connecting') : t('auth.login') }}
              <ArrowRight v-if="!authStore.isLoading" class="h-4 w-4" />
            </span>
          </Button>
        </form>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.grid-pattern {
  background-image:
    linear-gradient(hsl(var(--border) / 0.5) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.5) 1px, transparent 1px);
  background-size: 50px 50px;
}

.gradient-text {
  background: linear-gradient(135deg, hsl(var(--primary)) 0%, hsl(var(--accent)) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(30px, -30px) scale(1.1); }
}

.animate-float {
  animation: float 8s ease-in-out infinite;
}

.animate-float-delayed {
  animation: float 8s ease-in-out infinite;
  animation-delay: -4s;
}

.scanline {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, transparent, hsl(var(--primary) / 0.3), transparent);
  animation: scan 4s linear infinite;
}

@keyframes scan {
  0% { top: 0; }
  100% { top: 100%; }
}
</style>
