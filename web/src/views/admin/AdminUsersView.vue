<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type AdminUser } from '@/api/client'

const { t, locale } = useI18n()

const users = ref<AdminUser[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20

async function loadUsers() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listUsers(page.value, limit)
    users.value = response.data.users || []
    total.value = response.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function toggleAdmin(user: AdminUser) {
  try {
    await adminApi.updateUser(user.id, { is_admin: !user.is_admin })
    user.is_admin = !user.is_admin
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToUpdate')
  }
}

async function toggleActive(user: AdminUser) {
  try {
    await adminApi.updateUser(user.id, { is_active: !user.is_active })
    user.is_active = !user.is_active
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToUpdate')
  }
}

async function resetPassword(user: AdminUser) {
  const newPassword = prompt(t('admin.users.resetPasswordPrompt', { phone: user.phone }))
  if (!newPassword) return
  if (newPassword.length < 8) {
    error.value = t('auth.passwordTooShort')
    return
  }
  try {
    await adminApi.resetPassword(user.id, newPassword)
    error.value = ''
    alert(t('admin.users.passwordResetSuccess'))
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToResetPassword')
  }
}

async function mergeUsers(user: AdminUser) {
  const secondaryIdStr = prompt(t('admin.users.mergePrompt', { phone: user.phone, id: user.id }))
  if (!secondaryIdStr) return
  const secondaryId = parseInt(secondaryIdStr, 10)
  if (isNaN(secondaryId) || secondaryId <= 0) {
    error.value = 'Invalid user ID'
    return
  }
  if (!confirm(t('admin.users.confirmMerge', { primaryId: user.id, secondaryId, phone: user.phone }))) return
  try {
    await adminApi.mergeUsers(user.id, secondaryId)
    error.value = ''
    alert(t('admin.users.mergeSuccess'))
    loadUsers()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToMerge')
  }
}

async function deleteUser(user: AdminUser) {
  if (!confirm(t('admin.users.confirmDelete', { phone: user.phone }))) return

  try {
    await adminApi.deleteUser(user.id)
    users.value = users.value.filter((u) => u.id !== user.id)
    total.value--
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToDelete')
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function nextPage() {
  if (page.value * limit < total.value) {
    page.value++
    loadUsers()
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadUsers()
  }
}

onMounted(loadUsers)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('admin.users.title') }}</h1>
          <p class="text-muted-foreground">{{ t('admin.users.subtitle') }}</p>
        </div>
        <Button @click="loadUsers" :loading="loading" variant="outline">{{ t('common.refresh') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="users.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('admin.users.noUsers') }}</p>
      </div>

      <div v-else class="space-y-4">
        <Card class="overflow-hidden">
          <table class="w-full">
            <thead class="bg-muted/50">
              <tr>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.users.phone') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.users.name') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.users.status') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.users.role') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.users.lastLogin') }}</th>
                <th class="text-right p-3 text-sm font-medium">{{ t('admin.users.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id" class="border-t">
                <td class="p-3 font-mono text-sm">{{ user.phone }}</td>
                <td class="p-3">{{ user.display_name || '-' }}</td>
                <td class="p-3">
                  <span
                    :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      user.is_active
                        ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                        : 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
                    ]"
                  >
                    {{ user.is_active ? t('admin.users.active') : t('admin.users.blocked') }}
                  </span>
                </td>
                <td class="p-3">
                  <span
                    :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      user.is_admin
                        ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                        : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
                    ]"
                  >
                    {{ user.is_admin ? t('admin.users.admin') : t('admin.users.user') }}
                  </span>
                </td>
                <td class="p-3 text-sm text-muted-foreground">
                  {{ user.last_login_at ? formatDate(user.last_login_at) : '-' }}
                </td>
                <td class="p-3">
                  <div class="flex items-center justify-end space-x-2">
                    <Button variant="ghost" size="sm" @click="toggleActive(user)" :title="user.is_active ? t('admin.users.block') : t('admin.users.unblock')">
                      <svg v-if="user.is_active" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                      </svg>
                      <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                        <path d="M7 11V7a5 5 0 0 1 9.9-1" />
                      </svg>
                    </Button>
                    <Button variant="ghost" size="sm" @click="toggleAdmin(user)" :title="user.is_admin ? t('admin.users.removeAdmin') : t('admin.users.makeAdmin')">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" :class="user.is_admin ? 'text-purple-500' : ''" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                      </svg>
                    </Button>
                    <Button variant="ghost" size="sm" @click="resetPassword(user)" :title="t('admin.users.resetPassword')">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
                      </svg>
                    </Button>
                    <Button variant="ghost" size="sm" @click="mergeUsers(user)" :title="t('admin.users.merge')">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5" />
                      </svg>
                    </Button>
                    <Button variant="ghost" size="sm" @click="deleteUser(user)" :title="t('common.delete')">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </Card>

        <!-- Pagination -->
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            {{ t('admin.pagination.showing', { from: (page - 1) * limit + 1, to: Math.min(page * limit, total), total }) }}
          </p>
          <div class="flex space-x-2">
            <Button variant="outline" size="sm" @click="prevPage" :disabled="page === 1">
              {{ t('admin.pagination.prev') }}
            </Button>
            <Button variant="outline" size="sm" @click="nextPage" :disabled="page * limit >= total">
              {{ t('admin.pagination.next') }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
