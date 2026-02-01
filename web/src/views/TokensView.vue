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

const copiedWhat = ref<'token' | 'command' | ''>('')

async function copyToken() {
  if (newTokenVisible.value) {
    await navigator.clipboard.writeText(newTokenVisible.value)
    copiedWhat.value = 'token'
    setTimeout(() => (copiedWhat.value = ''), 2000)
  }
}

async function copyCommand() {
  if (newTokenVisible.value) {
    await navigator.clipboard.writeText(`fxtunnel login --token ${newTokenVisible.value}`)
    copiedWhat.value = 'command'
    setTimeout(() => (copiedWhat.value = ''), 2000)
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

function formatSubdomains(subs: string[]): string {
  if (!subs || subs.length === 0 || (subs.length === 1 && subs[0] === '*')) {
    return t('tokens.allSubdomains')
  }
  return subs.join(', ')
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
          <div class="flex items-center justify-center mb-4">
            <div class="w-12 h-12 rounded-full bg-emerald-500/15 flex items-center justify-center">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            </div>
          </div>
          <h2 class="text-xl font-bold text-center mb-4">{{ t('tokens.tokenCreated') }}</h2>

          <div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4 mb-4">
            <div class="flex gap-3">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              <p class="text-sm text-yellow-200">{{ t('tokens.copyWarning') }}</p>
            </div>
          </div>

          <div
            class="group/token relative bg-muted p-4 rounded-lg font-mono text-sm break-all mb-3 border border-border cursor-pointer transition-colors hover:border-primary/30"
            @click="copyToken"
          >
            {{ newTokenVisible }}
            <span class="absolute top-2 right-2 text-xs text-muted-foreground opacity-0 group-hover/token:opacity-100 transition-opacity">
              {{ copiedWhat === 'token' ? t('common.copied') : t('common.copy') }}
            </span>
          </div>

          <div
            class="group/cmd relative bg-muted/50 rounded-lg p-3 mb-4 border border-border/50 cursor-pointer transition-colors hover:border-primary/30"
            @click="copyCommand"
          >
            <p class="text-xs text-muted-foreground mb-1.5">{{ t('tokens.copyUsage') }}</p>
            <code class="text-xs font-mono text-primary">fxtunnel login --token {{ newTokenVisible }}</code>
            <span class="absolute top-2 right-2 text-xs text-muted-foreground opacity-0 group-hover/cmd:opacity-100 transition-opacity">
              {{ copiedWhat === 'command' ? t('common.copied') : t('common.copy') }}
            </span>
          </div>

          <div class="flex space-x-2">
            <Button @click="copyToken" variant="outline" class="flex-1">
              {{ copiedWhat === 'token' ? t('common.copied') : t('tokens.copyKey') }}
            </Button>
            <Button @click="copyCommand" variant="outline" class="flex-1">
              {{ copiedWhat === 'command' ? t('common.copied') : t('tokens.copyCommand') }}
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
          <h2 class="text-xl font-bold mb-2">{{ t('tokens.createTitle') }}</h2>
          <p class="text-sm text-muted-foreground mb-5">{{ t('tokens.createHint') }}</p>

          <form @submit.prevent="createToken" class="space-y-5">
            <div v-if="createError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
              {{ createError }}
            </div>

            <div class="space-y-1.5">
              <label class="text-sm font-medium">{{ t('tokens.tokenName') }}</label>
              <Input v-model="createForm.name" :placeholder="t('tokens.tokenNamePlaceholder')" required />
              <p class="text-xs text-muted-foreground">{{ t('tokens.tokenNameHint') }}</p>
            </div>

            <details class="group">
              <summary class="text-sm font-medium cursor-pointer text-muted-foreground hover:text-foreground transition-colors select-none flex items-center gap-1.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 transition-transform group-open:rotate-90" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
                {{ t('tokens.advancedSettings') }}
              </summary>
              <div class="space-y-5 mt-4 pl-0.5">
                <div class="space-y-1.5">
                  <label class="text-sm font-medium">{{ t('tokens.allowedSubdomains') }}</label>
                  <Input
                    v-model="createForm.allowed_subdomains"
                    :placeholder="t('tokens.allowedSubdomainsPlaceholder')"
                  />
                  <p class="text-xs text-muted-foreground">{{ t('tokens.allowedSubdomainsHint') }}</p>
                </div>

                <div class="space-y-1.5">
                  <label class="text-sm font-medium">{{ t('tokens.maxTunnels') }}</label>
                  <input
                    v-model.number="createForm.max_tunnels"
                    type="number"
                    min="1"
                    max="100"
                    class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                  <p class="text-xs text-muted-foreground">{{ t('tokens.maxTunnelsHint') }}</p>
                </div>
              </div>
            </details>

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

      <!-- Empty state -->
      <div v-else-if="tokens.length === 0" class="space-y-6">
        <div class="text-center py-8">
          <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          </div>
          <p class="text-lg font-medium mb-1">{{ t('tokens.noTokens') }}</p>
          <p class="text-sm text-muted-foreground max-w-sm mx-auto">{{ t('tokens.noTokensHint') }}</p>
        </div>

        <Card class="p-5 max-w-lg mx-auto">
          <h3 class="text-sm font-semibold mb-3">{{ t('tokens.howToUse') }}</h3>
          <ol class="space-y-2.5 text-sm text-muted-foreground">
            <li class="flex gap-3 items-start">
              <span class="flex-shrink-0 w-5 h-5 rounded-full bg-primary/15 text-primary text-xs font-bold flex items-center justify-center mt-0.5">1</span>
              <span>{{ t('tokens.howToUseStep1') }}</span>
            </li>
            <li class="flex gap-3 items-start">
              <span class="flex-shrink-0 w-5 h-5 rounded-full bg-primary/15 text-primary text-xs font-bold flex items-center justify-center mt-0.5">2</span>
              <span>{{ t('tokens.howToUseStep2') }}</span>
            </li>
            <li class="flex gap-3 items-start">
              <span class="flex-shrink-0 w-5 h-5 rounded-full bg-primary/15 text-primary text-xs font-bold flex items-center justify-center mt-0.5">3</span>
              <span>{{ t('tokens.howToUseStep3') }}</span>
            </li>
          </ol>
        </Card>
      </div>

      <!-- Token list -->
      <div v-else class="space-y-3">
        <Card v-for="token in tokens" :key="token.id" class="p-4">
          <div class="flex items-start justify-between">
            <div class="space-y-2 min-w-0">
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
                <h3 class="font-medium truncate">{{ token.name }}</h3>
              </div>
              <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span class="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                  {{ t('tokens.subdomains') }}: {{ formatSubdomains(token.allowed_subdomains) }}
                </span>
                <span class="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
                  {{ t('tokens.tunnelsLimit') }}: {{ token.max_tunnels }}
                </span>
                <span>{{ t('tokens.created') }}: {{ formatDate(token.created_at) }}</span>
                <span>
                  {{ t('tokens.lastUsed') }}: {{ token.last_used_at ? formatDate(token.last_used_at) : t('tokens.neverUsed') }}
                </span>
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
