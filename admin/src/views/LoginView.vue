<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <!-- Background decorations -->
    <div class="fixed inset-0 pointer-events-none overflow-hidden">
      <div class="absolute top-1/4 -left-32 w-96 h-96 bg-primary/5 rounded-full blur-3xl" />
      <div class="absolute bottom-1/4 -right-32 w-96 h-96 bg-accent/5 rounded-full blur-3xl" />
    </div>

    <Card variant="glass" class="relative w-full max-w-[420px] p-8">
      <!-- Logo -->
      <div class="text-center mb-8">
        <h1 class="font-display font-bold text-3xl text-primary">fxTunnel</h1>
        <p class="text-sm text-muted-foreground mt-1">Админ-панель</p>
      </div>

      <!-- Error alert -->
      <div
        v-if="errorMessage"
        class="mb-6 rounded-lg border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive animate-shake"
      >
        {{ errorMessage }}
      </div>

      <!-- Login form -->
      <form @submit.prevent="handleLogin" class="space-y-5">
        <div>
          <label class="block text-sm font-medium text-foreground mb-1.5">
            Телефон или Email
          </label>
          <Input
            v-model="phone"
            placeholder="+7 (999) 123-45-67 или email@example.com"
            :disabled="loading"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-foreground mb-1.5">
            Пароль
          </label>
          <Input
            v-model="password"
            type="password"
            placeholder="Введите пароль"
            :disabled="loading"
          />
        </div>

        <!-- TOTP field (shown when required) -->
        <div v-if="showTotp">
          <label class="block text-sm font-medium text-foreground mb-1.5">
            Код TOTP
          </label>
          <Input
            v-model="totpCode"
            placeholder="6-значный код"
            :disabled="loading"
            class="font-mono tracking-widest text-center"
          />
        </div>

        <Button
          variant="glow"
          :loading="loading"
          :disabled="loading || !phone || !password"
          class="w-full !py-3"
          type="submit"
        >
          {{ loading ? 'Вход...' : 'Войти' }}
        </Button>
      </form>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getErrorMessage } from '@/utils/error'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

const phone = ref('')
const password = ref('')
const totpCode = ref('')
const showTotp = ref(false)
const loading = ref(false)
const errorMessage = ref('')

async function handleLogin() {
  errorMessage.value = ''
  loading.value = true

  try {
    await authStore.login(
      phone.value.trim(),
      password.value,
      showTotp.value ? totpCode.value.trim() : undefined,
    )
    router.push({ name: 'dashboard' })
  } catch (err: unknown) {
    const msg = getErrorMessage(err, 'Ошибка входа')

    // Detect TOTP requirement
    if (msg.toLowerCase().includes('totp') || msg.toLowerCase().includes('2fa')) {
      showTotp.value = true
      errorMessage.value = 'Введите код двухфакторной аутентификации'
    } else {
      errorMessage.value = msg
    }
  } finally {
    loading.value = false
  }
}
</script>
