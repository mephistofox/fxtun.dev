<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { tokensApi, type APIToken } from '@/api/client'

const { t, locale } = useI18n()

const tokens = ref<APIToken[]>([])
const loading = ref(true)
const error = ref('')
const showCreateDialog = ref(false)
const newTokenVisible = ref<string | null>(null)
const copied = ref(false)

const createForm = ref({
  name: '',
  allowed_subdomains: '*',
  max_tunnels: 10,
})
const creating = ref(false)
const createError = ref('')

async function loadTokens() {
  loading.value = true
  error.value = ''
  try {
    const response = await tokensApi.list()
    tokens.value = response.data.tokens || []
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('tokens.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function createToken() {
  creating.value = true
  createError.value = ''
  try {
    const subdomains = createForm.value.allowed_subdomains
      .split(',')
      .map((s) => s.trim())
      .filter((s) => s)

    const response = await tokensApi.create({
      name: createForm.value.name,
      allowed_subdomains: subdomains,
      max_tunnels: createForm.value.max_tunnels,
    })

    newTokenVisible.value = response.data.token
    tokens.value.push(response.data.info)

    createForm.value = {
      name: '',
      allowed_subdomains: '*',
      max_tunnels: 10,
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    createError.value = err.response?.data?.error || t('tokens.failedToCreate')
  } finally {
    creating.value = false
  }
}

async function deleteToken(id: number) {
  if (!confirm(t('tokens.confirmDelete'))) return

  try {
    await tokensApi.delete(id)
    tokens.value = tokens.value.filter((t) => t.id !== id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('tokens.failedToDelete')
  }
}

async function copyToken() {
  if (newTokenVisible.value) {
    await navigator.clipboard.writeText(newTokenVisible.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  }
}

function closeNewTokenDialog() {
  newTokenVisible.value = null
  showCreateDialog.value = false
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

onMounted(loadTokens)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('tokens.title') }}</h1>
          <p class="text-muted-foreground">{{ t('tokens.subtitle') }}</p>
        </div>
        <Button @click="showCreateDialog = true">{{ t('tokens.createToken') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <!-- New Token Display Dialog -->
      <div
        v-if="newTokenVisible"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      >
        <Card class="w-full max-w-lg p-6">
          <h2 class="text-xl font-bold text-center mb-4">{{ t('tokens.tokenCreated') }}</h2>
          <div class="bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-4">
            <p class="text-sm text-yellow-800 dark:text-yellow-200 font-medium mb-2">
              {{ t('tokens.copyWarning') }}
            </p>
          </div>
          <div class="bg-muted p-4 rounded-lg font-mono text-sm break-all mb-4">
            {{ newTokenVisible }}
          </div>
          <div class="flex space-x-2">
            <Button @click="copyToken" variant="outline" class="flex-1">
              {{ copied ? t('common.copied') : t('common.copy') }}
            </Button>
            <Button @click="closeNewTokenDialog" class="flex-1">{{ t('common.done') }}</Button>
          </div>
        </Card>
      </div>

      <!-- Create Token Dialog -->
      <div
        v-if="showCreateDialog && !newTokenVisible"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      >
        <Card class="w-full max-w-md p-6">
          <h2 class="text-xl font-bold mb-4">{{ t('tokens.createTitle') }}</h2>
          <form @submit.prevent="createToken" class="space-y-4">
            <div v-if="createError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
              {{ createError }}
            </div>

            <div class="space-y-2">
              <label class="text-sm font-medium">{{ t('tokens.tokenName') }}</label>
              <Input v-model="createForm.name" placeholder="my-dev-token" required />
            </div>

            <div class="space-y-2">
              <label class="text-sm font-medium">{{ t('tokens.allowedSubdomains') }}</label>
              <Input
                v-model="createForm.allowed_subdomains"
                placeholder="* or dev-*, test-*"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('tokens.allowedSubdomainsHint') }}
              </p>
            </div>

            <div class="space-y-2">
              <label class="text-sm font-medium">{{ t('tokens.maxTunnels') }}</label>
              <input
                v-model.number="createForm.max_tunnels"
                type="number"
                min="1"
                max="100"
                class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>

            <div class="flex space-x-2">
              <Button type="button" variant="outline" @click="showCreateDialog = false" class="flex-1">
                {{ t('common.cancel') }}
              </Button>
              <Button type="submit" :loading="creating" class="flex-1">{{ t('common.create') }}</Button>
            </div>
          </form>
        </Card>
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="tokens.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('tokens.noTokens') }}</p>
        <p class="text-sm text-muted-foreground mt-2">
          {{ t('tokens.noTokensHint') }}
        </p>
      </div>

      <div v-else class="space-y-4">
        <Card v-for="token in tokens" :key="token.id" class="p-4">
          <div class="flex items-start justify-between">
            <div class="space-y-1">
              <h3 class="font-medium">{{ token.name }}</h3>
              <div class="text-sm text-muted-foreground space-y-0.5">
                <p>{{ t('tokens.subdomains') }}: {{ token.allowed_subdomains.join(', ') }}</p>
                <p>{{ t('tokens.maxTunnels') }}: {{ token.max_tunnels }}</p>
                <p>{{ t('tokens.created') }}: {{ formatDate(token.created_at) }}</p>
                <p v-if="token.last_used_at">
                  {{ t('tokens.lastUsed') }}: {{ formatDate(token.last_used_at) }}
                </p>
              </div>
            </div>
            <Button variant="ghost" size="icon" @click="deleteToken(token.id)" :title="t('tokens.deleteToken')">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6" />
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              </svg>
            </Button>
          </div>
        </Card>
      </div>
    </div>
  </Layout>
</template>
