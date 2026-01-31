<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { authApi } from '@/api/client'

const route = useRoute()
const { t } = useI18n()

const sessionId = ref('')
const loading = ref(false)
const error = ref('')
const authorized = ref(false)
const missingSession = ref(false)

onMounted(() => {
  const session = route.query.session as string | undefined
  if (!session) {
    missingSession.value = true
    return
  }
  sessionId.value = session
})

async function authorize() {
  loading.value = true
  error.value = ''
  try {
    await authApi.deviceAuthorize(sessionId.value)
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
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5" />
              <line x1="12" y1="19" x2="20" y2="19" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold">Authorize CLI</h1>
          <p class="text-muted-foreground mt-2">Confirm access for the fxTunnel CLI client</p>
        </div>

        <!-- Missing session -->
        <div v-if="missingSession" class="bg-destructive/10 text-destructive p-4 rounded-lg text-sm border border-destructive/20 text-center">
          Missing session parameter. Please use the link provided by the CLI.
        </div>

        <!-- Success -->
        <div v-else-if="authorized" class="bg-green-500/10 text-green-600 dark:text-green-400 p-4 rounded-lg text-sm border border-green-500/20 text-center">
          Authorized! You can close this page and return to the terminal.
        </div>

        <!-- Authorize form -->
        <div v-else class="space-y-4">
          <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-lg text-sm border border-destructive/20">
            {{ error }}
          </div>

          <p class="text-sm text-muted-foreground text-center">
            A CLI client is requesting access to your account. Click the button below to authorize it.
          </p>

          <Button variant="glow" class="w-full" size="lg" :loading="loading" @click="authorize">
            Authorize CLI
          </Button>
        </div>
      </Card>
    </div>
  </Layout>
</template>
