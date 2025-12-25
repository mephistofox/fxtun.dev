<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type InviteCode } from '@/api/client'

const { t, locale } = useI18n()

const invites = ref<InviteCode[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20
const showUnusedOnly = ref(false)

const showCreateDialog = ref(false)
const expiresInDays = ref<number | undefined>(undefined)
const creating = ref(false)
const newCode = ref<string | null>(null)
const copied = ref(false)

async function loadInvites() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listInvites(page.value, limit, showUnusedOnly.value)
    invites.value = response.data.codes || []
    total.value = response.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function createInvite() {
  creating.value = true
  try {
    const response = await adminApi.createInvite(expiresInDays.value)
    newCode.value = response.data.code
    invites.value.unshift(response.data)
    total.value++
    expiresInDays.value = undefined
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.invites.failedToCreate')
  } finally {
    creating.value = false
  }
}

async function deleteInvite(invite: InviteCode) {
  if (!confirm(t('admin.invites.confirmDelete'))) return

  try {
    await adminApi.deleteInvite(invite.id)
    invites.value = invites.value.filter((i) => i.id !== invite.id)
    total.value--
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.invites.failedToDelete')
  }
}

async function copyCode() {
  if (newCode.value) {
    await navigator.clipboard.writeText(newCode.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 2000)
  }
}

function closeNewCodeDialog() {
  newCode.value = null
  showCreateDialog.value = false
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function toggleFilter() {
  showUnusedOnly.value = !showUnusedOnly.value
  page.value = 1
  loadInvites()
}

onMounted(loadInvites)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('admin.invites.title') }}</h1>
          <p class="text-muted-foreground">{{ t('admin.invites.subtitle') }}</p>
        </div>
        <div class="flex space-x-2">
          <Button @click="toggleFilter" variant="outline" :class="showUnusedOnly ? 'bg-primary/10' : ''">
            {{ showUnusedOnly ? t('admin.invites.showAll') : t('admin.invites.showUnused') }}
          </Button>
          <Button @click="showCreateDialog = true">{{ t('admin.invites.create') }}</Button>
        </div>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <!-- New Code Display Dialog -->
      <div
        v-if="newCode"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      >
        <Card class="w-full max-w-lg p-6">
          <h2 class="text-xl font-bold text-center mb-4">{{ t('admin.invites.codeCreated') }}</h2>
          <div class="bg-muted p-4 rounded-lg font-mono text-center text-lg break-all mb-4">
            {{ newCode }}
          </div>
          <div class="flex space-x-2">
            <Button @click="copyCode" variant="outline" class="flex-1">
              {{ copied ? t('common.copied') : t('common.copy') }}
            </Button>
            <Button @click="closeNewCodeDialog" class="flex-1">{{ t('common.done') }}</Button>
          </div>
        </Card>
      </div>

      <!-- Create Invite Dialog -->
      <div
        v-if="showCreateDialog && !newCode"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      >
        <Card class="w-full max-w-md p-6">
          <h2 class="text-xl font-bold mb-4">{{ t('admin.invites.createTitle') }}</h2>
          <form @submit.prevent="createInvite" class="space-y-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">{{ t('admin.invites.expiresIn') }}</label>
              <Input
                v-model.number="expiresInDays"
                type="number"
                min="1"
                :placeholder="t('admin.invites.noExpiry')"
              />
              <p class="text-xs text-muted-foreground">{{ t('admin.invites.expiresHint') }}</p>
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

      <div v-else-if="invites.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('admin.invites.noInvites') }}</p>
      </div>

      <div v-else class="space-y-4">
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Card v-for="invite in invites" :key="invite.id" class="p-4">
            <div class="flex items-start justify-between">
              <div class="space-y-2">
                <code class="text-sm font-mono bg-muted px-2 py-1 rounded">{{ invite.code }}</code>
                <div class="text-sm text-muted-foreground space-y-0.5">
                  <p>
                    <span
                      :class="[
                        'px-2 py-0.5 text-xs font-medium rounded-full',
                        invite.used
                          ? 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
                          : 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
                      ]"
                    >
                      {{ invite.used ? t('admin.invites.used') : t('admin.invites.available') }}
                    </span>
                  </p>
                  <p>{{ t('admin.invites.created') }}: {{ formatDate(invite.created_at) }}</p>
                  <p v-if="invite.expires_at">{{ t('admin.invites.expires') }}: {{ formatDate(invite.expires_at) }}</p>
                  <p v-if="invite.used_at">{{ t('admin.invites.usedAt') }}: {{ formatDate(invite.used_at) }}</p>
                </div>
              </div>
              <Button
                v-if="!invite.used"
                variant="ghost"
                size="icon"
                @click="deleteInvite(invite)"
                :title="t('common.delete')"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6" />
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                </svg>
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </div>
  </Layout>
</template>
