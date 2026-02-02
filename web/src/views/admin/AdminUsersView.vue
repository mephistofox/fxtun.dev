<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type AdminUser, type Plan } from '@/api/client'

const { t, locale } = useI18n()

const users = ref<AdminUser[]>([])
const plans = ref<Plan[]>([])
const loading = ref(true)
const error = ref('')
const successMsg = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20

const search = ref('')
const activeFilter = ref<'all' | 'active' | 'blocked' | 'admins'>('all')

// Inline confirmation state
const confirmAction = ref<{ userId: number; action: string } | null>(null)
let confirmTimer: ReturnType<typeof setTimeout> | null = null

// Reset password inline state
const resetPasswordUserId = ref<number | null>(null)
const resetPasswordValue = ref('')

// Merge inline state
const mergeUserId = ref<number | null>(null)
const mergeSecondaryId = ref('')

// Plan dropdown state
const planDropdownUserId = ref<number | null>(null)

const planColors: Record<string, string> = {
  free: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  pro: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  business: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  enterprise: 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300',
}

function getPlanColor(slug?: string): string {
  if (!slug) return planColors.free
  return planColors[slug] || 'bg-primary/10 text-primary dark:bg-primary/20'
}

function getPlanForUser(user: AdminUser): Plan | undefined {
  return plans.value.find(p => p.id === user.plan_id)
}

const filteredUsers = computed(() => {
  let result = users.value

  // Filter by tab
  if (activeFilter.value === 'active') {
    result = result.filter(u => u.is_active)
  } else if (activeFilter.value === 'blocked') {
    result = result.filter(u => !u.is_active)
  } else if (activeFilter.value === 'admins') {
    result = result.filter(u => u.is_admin)
  }

  // Filter by search
  const q = search.value.toLowerCase().trim()
  if (q) {
    result = result.filter(u =>
      u.phone.toLowerCase().includes(q) ||
      (u.display_name && u.display_name.toLowerCase().includes(q))
    )
  }

  return result
})

const filterTabs = computed(() => [
  { key: 'all' as const, label: t('admin.filterAll'), count: users.value.length },
  { key: 'active' as const, label: t('admin.users.active'), count: users.value.filter(u => u.is_active).length },
  { key: 'blocked' as const, label: t('admin.users.blocked'), count: users.value.filter(u => !u.is_active).length },
  { key: 'admins' as const, label: t('admin.users.admin'), count: users.value.filter(u => u.is_admin).length },
])

const paginationFrom = computed(() => (page.value - 1) * limit + 1)
const paginationTo = computed(() => Math.min(page.value * limit, total.value))

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

function showSuccess(msg: string) {
  successMsg.value = msg
  setTimeout(() => { successMsg.value = '' }, 3000)
}

function clearConfirm() {
  confirmAction.value = null
  if (confirmTimer) { clearTimeout(confirmTimer); confirmTimer = null }
}

function requestConfirm(userId: number, action: string) {
  clearConfirm()
  confirmAction.value = { userId, action }
  confirmTimer = setTimeout(clearConfirm, 3000)
}

function isConfirming(userId: number, action: string): boolean {
  return confirmAction.value?.userId === userId && confirmAction.value?.action === action
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

function startResetPassword(userId: number) {
  resetPasswordUserId.value = userId
  resetPasswordValue.value = ''
  mergeUserId.value = null
}

async function submitResetPassword(user: AdminUser) {
  if (resetPasswordValue.value.length < 8) {
    error.value = t('auth.passwordTooShort')
    return
  }
  try {
    await adminApi.resetPassword(user.id, resetPasswordValue.value)
    error.value = ''
    resetPasswordUserId.value = null
    resetPasswordValue.value = ''
    showSuccess(t('admin.users.passwordResetSuccess'))
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToResetPassword')
  }
}

function startMerge(userId: number) {
  mergeUserId.value = userId
  mergeSecondaryId.value = ''
  resetPasswordUserId.value = null
}

async function submitMerge(user: AdminUser) {
  const secondaryId = parseInt(mergeSecondaryId.value, 10)
  if (isNaN(secondaryId) || secondaryId <= 0) {
    error.value = 'Invalid user ID'
    return
  }
  if (!isConfirming(user.id, 'merge')) {
    requestConfirm(user.id, 'merge')
    return
  }
  clearConfirm()
  try {
    await adminApi.mergeUsers(user.id, secondaryId)
    error.value = ''
    mergeUserId.value = null
    mergeSecondaryId.value = ''
    showSuccess(t('admin.users.mergeSuccess'))
    loadUsers()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToMerge')
  }
}

async function deleteUser(user: AdminUser) {
  if (!isConfirming(user.id, 'delete')) {
    requestConfirm(user.id, 'delete')
    return
  }
  clearConfirm()
  try {
    await adminApi.deleteUser(user.id)
    users.value = users.value.filter(u => u.id !== user.id)
    total.value--
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.users.failedToDelete')
  }
}

async function changePlan(user: AdminUser, planId: number) {
  planDropdownUserId.value = null
  try {
    await adminApi.updateUser(user.id, { plan_id: planId })
    user.plan_id = planId
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.plans.failedToUpdate')
  }
}

function togglePlanDropdown(userId: number) {
  planDropdownUserId.value = planDropdownUserId.value === userId ? null : userId
}

function closePlanDropdown(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('[data-plan-dropdown]')) {
    planDropdownUserId.value = null
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

onMounted(() => {
  loadUsers()
  adminApi.listPlans().then(r => { plans.value = r.data.plans || [] })
  document.addEventListener('click', closePlanDropdown)
})

onUnmounted(() => {
  document.removeEventListener('click', closePlanDropdown)
  if (confirmTimer) clearTimeout(confirmTimer)
})
</script>

<template>
  <Layout>
    <div class="space-y-4">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-foreground">{{ t('admin.users.title') }}</h1>
          <p class="text-sm text-muted-foreground">{{ t('admin.users.subtitle') }}</p>
        </div>
        <Button @click="loadUsers" :loading="loading" variant="outline" size="sm">
          <!-- refresh icon -->
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          {{ t('common.refresh') }}
        </Button>
      </div>

      <!-- Success message -->
      <div v-if="successMsg" class="bg-green-500/10 text-green-600 dark:text-green-400 px-3 py-2 rounded-md text-sm">
        {{ successMsg }}
      </div>

      <!-- Error message -->
      <div v-if="error" class="bg-destructive/10 text-destructive px-3 py-2 rounded-md text-sm flex items-center justify-between">
        <span>{{ error }}</span>
        <button @click="error = ''" class="text-destructive hover:text-destructive/80 ml-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>

      <!-- Search + Filter -->
      <div class="flex flex-col sm:flex-row gap-3">
        <div class="relative flex-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
          <Input
            v-model="search"
            :placeholder="t('admin.searchUsers')"
            class="pl-9"
          />
        </div>
        <div class="flex gap-1 bg-muted/50 rounded-lg p-1">
          <button
            v-for="tab in filterTabs"
            :key="tab.key"
            @click="activeFilter = tab.key"
            :class="[
              'px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap',
              activeFilter === tab.key
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            ]"
          >
            {{ tab.label }}
            <span class="ml-1 text-[10px] opacity-60">{{ tab.count }}</span>
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-12 text-muted-foreground text-sm">{{ t('common.loading') }}</div>

      <!-- Empty state -->
      <div v-else-if="filteredUsers.length === 0" class="text-center py-12">
        <p class="text-muted-foreground text-sm">{{ search || activeFilter !== 'all' ? t('admin.noResults') : t('admin.users.noUsers') }}</p>
      </div>

      <!-- Table -->
      <Card v-else class="overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b bg-muted/30">
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.phone') }}</th>
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.name') }}</th>
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.status') }}</th>
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.role') }}</th>
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.plan') }}</th>
                <th class="text-left px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.lastLogin') }}</th>
                <th class="text-right px-3 py-2.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('admin.users.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="user in filteredUsers" :key="user.id">
                <tr class="border-b border-border/50 hover:bg-muted/20 transition-colors">
                  <!-- Phone -->
                  <td class="px-3 py-2.5 font-mono text-xs whitespace-nowrap">{{ user.phone }}</td>

                  <!-- Name -->
                  <td class="px-3 py-2.5 whitespace-nowrap">
                    <span v-if="user.display_name" class="text-foreground">{{ user.display_name }}</span>
                    <span v-else class="text-muted-foreground">-</span>
                  </td>

                  <!-- Status -->
                  <td class="px-3 py-2.5">
                    <span
                      :class="[
                        'inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full',
                        user.is_active
                          ? 'bg-green-500/10 text-green-600 dark:text-green-400'
                          : 'bg-red-500/10 text-red-600 dark:text-red-400',
                      ]"
                    >
                      <span :class="['w-1.5 h-1.5 rounded-full mr-1', user.is_active ? 'bg-green-500' : 'bg-red-500']" />
                      {{ user.is_active ? t('admin.users.active') : t('admin.users.blocked') }}
                    </span>
                  </td>

                  <!-- Role -->
                  <td class="px-3 py-2.5">
                    <span
                      :class="[
                        'inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full',
                        user.is_admin
                          ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400'
                          : 'bg-muted text-muted-foreground',
                      ]"
                    >
                      {{ user.is_admin ? t('admin.users.admin') : t('admin.users.user') }}
                    </span>
                  </td>

                  <!-- Plan -->
                  <td class="px-3 py-2.5">
                    <div class="relative" data-plan-dropdown>
                      <button
                        @click.stop="togglePlanDropdown(user.id)"
                        :class="[
                          'inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full transition-colors cursor-pointer',
                          getPlanColor(getPlanForUser(user)?.slug),
                        ]"
                      >
                        {{ getPlanForUser(user)?.name || '-' }}
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
                      </button>

                      <!-- Plan dropdown -->
                      <div
                        v-if="planDropdownUserId === user.id"
                        class="absolute z-50 left-0 top-full mt-1 w-56 bg-background border border-border rounded-lg shadow-lg py-1"
                      >
                        <button
                          v-for="plan in plans"
                          :key="plan.id"
                          @click.stop="changePlan(user, plan.id)"
                          :class="[
                            'w-full text-left px-3 py-2 hover:bg-muted/50 transition-colors flex items-center justify-between',
                            user.plan_id === plan.id ? 'bg-muted/30' : '',
                          ]"
                        >
                          <div>
                            <span :class="['inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full', getPlanColor(plan.slug)]">
                              {{ plan.name }}
                            </span>
                            <div class="text-[10px] text-muted-foreground mt-0.5">
                              {{ plan.max_tunnels === -1 ? t('admin.plans.unlimited') : plan.max_tunnels }} {{ t('admin.plans.maxTunnels').toLowerCase() }}
                              /
                              {{ plan.max_domains === -1 ? t('admin.plans.unlimited') : plan.max_domains }} {{ t('admin.plans.maxDomains').toLowerCase() }}
                            </div>
                          </div>
                          <svg v-if="user.plan_id === plan.id" xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                        </button>
                      </div>
                    </div>
                  </td>

                  <!-- Last Login -->
                  <td class="px-3 py-2.5 text-xs text-muted-foreground whitespace-nowrap">
                    {{ user.last_login_at ? formatDate(user.last_login_at) : '-' }}
                  </td>

                  <!-- Actions -->
                  <td class="px-3 py-2.5">
                    <div class="flex items-center justify-end gap-0.5">
                      <!-- Toggle active -->
                      <button
                        @click="toggleActive(user)"
                        :title="user.is_active ? t('admin.users.block') : t('admin.users.unblock')"
                        class="p-1.5 rounded-md hover:bg-muted/50 transition-colors text-muted-foreground hover:text-foreground"
                      >
                        <svg v-if="user.is_active" xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                        <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>
                      </button>

                      <!-- Toggle admin -->
                      <button
                        @click="toggleAdmin(user)"
                        :title="user.is_admin ? t('admin.users.removeAdmin') : t('admin.users.makeAdmin')"
                        class="p-1.5 rounded-md hover:bg-muted/50 transition-colors"
                        :class="user.is_admin ? 'text-purple-500' : 'text-muted-foreground hover:text-foreground'"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                      </button>

                      <!-- Reset password -->
                      <button
                        @click="startResetPassword(user.id)"
                        :title="t('admin.users.resetPassword')"
                        class="p-1.5 rounded-md hover:bg-muted/50 transition-colors text-muted-foreground hover:text-foreground"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
                      </button>

                      <!-- Merge -->
                      <button
                        @click="startMerge(user.id)"
                        :title="t('admin.users.merge')"
                        class="p-1.5 rounded-md hover:bg-muted/50 transition-colors text-muted-foreground hover:text-foreground"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5"/></svg>
                      </button>

                      <!-- Delete -->
                      <button
                        @click="deleteUser(user)"
                        :title="t('common.delete')"
                        :class="[
                          'p-1.5 rounded-md transition-colors',
                          isConfirming(user.id, 'delete')
                            ? 'bg-destructive text-destructive-foreground'
                            : 'hover:bg-muted/50 text-destructive/70 hover:text-destructive'
                        ]"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                      </button>
                    </div>
                  </td>
                </tr>

                <!-- Inline: Reset Password -->
                <tr v-if="resetPasswordUserId === user.id" class="border-b border-border/50 bg-muted/10">
                  <td colspan="7" class="px-3 py-2">
                    <div class="flex items-center gap-2 max-w-md">
                      <span class="text-xs text-muted-foreground whitespace-nowrap">{{ t('admin.users.resetPassword') }}:</span>
                      <Input
                        v-model="resetPasswordValue"
                        type="password"
                        :placeholder="t('admin.users.resetPasswordPrompt', { phone: user.phone })"
                        class="h-7 text-xs flex-1"
                        @keyup.enter="submitResetPassword(user)"
                      />
                      <Button size="xs" @click="submitResetPassword(user)">{{ t('common.save') }}</Button>
                      <Button size="xs" variant="ghost" @click="resetPasswordUserId = null">{{ t('common.cancel') }}</Button>
                    </div>
                  </td>
                </tr>

                <!-- Inline: Merge -->
                <tr v-if="mergeUserId === user.id" class="border-b border-border/50 bg-muted/10">
                  <td colspan="7" class="px-3 py-2">
                    <div class="flex items-center gap-2 max-w-md">
                      <span class="text-xs text-muted-foreground whitespace-nowrap">{{ t('admin.users.merge') }}:</span>
                      <Input
                        v-model="mergeSecondaryId"
                        type="number"
                        :placeholder="t('admin.users.mergePrompt', { phone: user.phone, id: user.id })"
                        class="h-7 text-xs flex-1"
                        @keyup.enter="submitMerge(user)"
                      />
                      <Button
                        size="xs"
                        :variant="isConfirming(user.id, 'merge') ? 'destructive' : 'default'"
                        @click="submitMerge(user)"
                      >
                        {{ isConfirming(user.id, 'merge') ? t('admin.users.confirmMerge', { primaryId: user.id, secondaryId: mergeSecondaryId, phone: user.phone }) : t('admin.users.merge') }}
                      </Button>
                      <Button size="xs" variant="ghost" @click="mergeUserId = null; clearConfirm()">{{ t('common.cancel') }}</Button>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </Card>

      <!-- Pagination -->
      <div v-if="!loading && total > 0" class="flex items-center justify-between">
        <p class="text-xs text-muted-foreground">
          {{ t('admin.pagination.showing', { from: paginationFrom, to: paginationTo, total }) }}
        </p>
        <div class="flex gap-1">
          <Button variant="outline" size="xs" @click="prevPage" :disabled="page === 1">
            {{ t('admin.pagination.prev') }}
          </Button>
          <Button variant="outline" size="xs" @click="nextPage" :disabled="page * limit >= total">
            {{ t('admin.pagination.next') }}
          </Button>
        </div>
      </div>
    </div>
  </Layout>
</template>
