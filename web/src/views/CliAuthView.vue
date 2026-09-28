<script setup lang="ts">
import { ref } from 'vue'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { authApi } from '@/api/client'

// The code is typed by hand on purpose. Approving straight from a link let
// anyone create a session and have a logged-in victim authorize it with one
// click, handing the attacker a live API token for the victim's account.
const userCode = ref('')
const loading = ref(false)
const error = ref('')
const authorized = ref(false)

async function authorize() {
  const code = userCode.value.trim().toUpperCase()
  if (!code) {
    error.value = 'Enter the code shown in your terminal'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await authApi.deviceAuthorize(code)
    authorized.value = true
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Authorization failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Layout>
    <div class="max-w-md mx-auto mt-16">
      <Card class="p-8">
        <div class="text-center mb-6">
          <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5" />
              <line x1="12" y1="19" x2="20" y2="19" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold">Authorize CLI</h1>
          <p class="text-muted-foreground mt-2">Confirm access for the fxtun CLI client</p>
        </div>

        <!-- Success -->
        <div v-if="authorized" class="bg-green-500/10 text-green-600 dark:text-green-400 p-4 rounded-lg text-sm border border-green-500/20 text-center">
          Authorized! You can close this page and return to the terminal.
        </div>

        <!-- Authorize form -->
        <div v-else class="space-y-4">
          <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-lg text-sm border border-destructive/20">
            {{ error }}
          </div>

          <p class="text-sm text-muted-foreground text-center">
            Enter the code shown in your terminal. Only approve a code you started yourself —
            it grants the CLI a token for your account.
          </p>

          <input
            v-model="userCode"
            type="text"
            inputmode="text"
            autocomplete="off"
            spellcheck="false"
            placeholder="XXXX-XXXX"
            class="w-full rounded-lg border border-border bg-card px-4 py-3 text-center text-lg font-mono tracking-widest uppercase focus:outline-none focus:ring-2 focus:ring-primary/40"
            @keyup.enter="authorize"
          />

          <Button variant="glow" class="w-full" size="lg" :loading="loading" @click="authorize">
            Authorize CLI
          </Button>
        </div>
      </Card>
    </div>
  </Layout>
</template>
