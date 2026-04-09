<template>
  <div class="p-6 space-y-6">
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-24">
      <svg
        class="h-8 w-8 animate-spin text-primary"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <template v-else-if="detail">
      <!-- Header -->
      <div class="flex flex-col sm:flex-row sm:items-center gap-4">
        <button
          type="button"
          class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
          @click="router.push({ name: 'users' })"
        >
          <ArrowLeft class="h-4 w-4" />
          Пользователи
        </button>
        <div class="flex-1" />
        <div class="flex items-center gap-2 flex-wrap">
          <Badge :variant="user.is_active ? 'success' : 'destructive'">
            {{ user.is_active ? 'Активен' : 'Заблокирован' }}
          </Badge>
          <Badge v-if="user.is_admin" variant="accent">Админ</Badge>
          <Badge variant="outline">{{ user.plan?.name || `Тариф #${user.plan_id}` }}</Badge>
        </div>
      </div>

      <!-- User info card -->
      <Card variant="glass" class="p-6">
        <div class="flex flex-col sm:flex-row gap-6">
          <!-- Avatar -->
          <div class="flex-shrink-0">
            <div class="w-16 h-16 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center">
              <span class="text-xl font-display font-bold text-primary">
                {{ initials }}
              </span>
            </div>
          </div>

          <!-- Info -->
          <div class="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Имя</p>
              <p class="text-sm font-medium text-foreground">{{ user.display_name || '---' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Email</p>
              <p class="text-sm font-medium text-foreground">{{ user.email || '---' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Телефон</p>
              <p class="text-sm font-medium text-foreground">{{ user.phone || '---' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Тариф</p>
              <p class="text-sm font-medium text-foreground">{{ user.plan?.name || `#${user.plan_id}` }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Создан</p>
              <p class="text-sm text-foreground">{{ formatDate(user.created_at) }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground mb-0.5">Последний вход</p>
              <p class="text-sm text-foreground">{{ user.last_login_at ? formatRelative(user.last_login_at) : '---' }}</p>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex flex-col gap-2 sm:items-end flex-shrink-0">
            <Button
              :variant="user.is_active ? 'destructive' : 'success'"
              size="sm"
              @click="toggleActive"
            >
              <component :is="user.is_active ? Ban : CheckCircle" class="h-4 w-4" />
              {{ user.is_active ? 'Заблокировать' : 'Разблокировать' }}
            </Button>
            <Button variant="outline" size="sm" @click="toggleAdmin">
              <component :is="user.is_admin ? ShieldOff : ShieldCheck" class="h-4 w-4" />
              {{ user.is_admin ? 'Убрать админа' : 'Сделать админом' }}
            </Button>
            <Button variant="outline" size="sm" @click="showResetPassword = true">
              <KeyRound class="h-4 w-4" />
              Сбросить пароль
            </Button>
            <Button variant="destructive" size="sm" @click="confirmDelete">
              <Trash2 class="h-4 w-4" />
              Удалить
            </Button>
          </div>
        </div>
      </Card>

      <!-- Tabs -->
      <Tabs v-model="activeTab" :tabs="tabs">
        <!-- Info tab -->
        <div v-if="activeTab === 'info'" class="space-y-4">
          <Card class="p-6">
            <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-4">
              <div>
                <dt class="text-xs text-muted-foreground">ID</dt>
                <dd class="text-sm font-mono text-foreground mt-0.5">{{ user.id }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Email</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ user.email || '---' }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Телефон</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ user.phone || '---' }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Имя</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ user.display_name || '---' }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">GitHub ID</dt>
                <dd class="text-sm font-mono text-foreground mt-0.5">{{ user.github_id || '---' }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Google ID</dt>
                <dd class="text-sm font-mono text-foreground mt-0.5">{{ user.google_id || '---' }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Токенов</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ detail.token_count }}</dd>
              </div>
              <div>
                <dt class="text-xs text-muted-foreground">Доменов</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ detail.domain_count }}</dd>
              </div>
              <div v-if="detail.tunnel_stats">
                <dt class="text-xs text-muted-foreground">Всего подключений</dt>
                <dd class="text-sm text-foreground mt-0.5">{{ detail.tunnel_stats.total_connections }}</dd>
              </div>
              <div v-if="detail.tunnel_stats">
                <dt class="text-xs text-muted-foreground">Трафик (отправлено / получено)</dt>
                <dd class="text-sm font-mono text-foreground mt-0.5">
                  {{ formatBytes(detail.tunnel_stats.total_bytes_sent) }} / {{ formatBytes(detail.tunnel_stats.total_bytes_received) }}
                </dd>
              </div>
            </dl>
          </Card>
        </div>

        <!-- Tunnels tab -->
        <div v-if="activeTab === 'tunnels'">
          <DataTable
            :columns="tunnelColumns"
            :data="detail.tunnel_history"
            :loading="false"
            empty-text="Нет истории тоннелей"
          >
            <template #id="{ value }">
              <span class="font-mono text-xs">{{ value }}</span>
            </template>
            <template #tunnel_type="{ value }">
              <Badge
                :variant="value === 'http' ? 'success' : value === 'tcp' ? 'info' : 'accent'"
              >
                {{ value.toUpperCase() }}
              </Badge>
            </template>
            <template #connected_at="{ value }">
              <span class="text-xs text-muted-foreground">{{ formatDate(value) }}</span>
            </template>
            <template #disconnected_at="{ value }">
              <span class="text-xs text-muted-foreground">{{ value ? formatDate(value) : 'Активен' }}</span>
            </template>
            <template #bytes_sent="{ row }">
              <span class="text-xs font-mono text-muted-foreground">
                {{ formatBytes(row.bytes_sent) }} / {{ formatBytes(row.bytes_received) }}
              </span>
            </template>
          </DataTable>
        </div>

        <!-- Subscriptions tab -->
        <div v-if="activeTab === 'subscriptions'">
          <DataTable
            :columns="subscriptionColumns"
            :data="detail.subscriptions"
            :loading="false"
            empty-text="Нет подписок"
          >
            <template #id="{ value }">
              <span class="font-mono text-xs">{{ value }}</span>
            </template>
            <template #plan="{ row }">
              <span class="text-sm">{{ row.plan?.name || `#${row.plan_id}` }}</span>
            </template>
            <template #status="{ value }">
              <Badge :variant="statusVariant(value)">{{ statusLabel(value) }}</Badge>
            </template>
            <template #current_period_end="{ value }">
              <span class="text-xs text-muted-foreground">{{ value ? formatDate(value) : '---' }}</span>
            </template>
            <template #actions="{ row }">
              <div class="flex gap-1">
                <Button
                  v-if="row.status === 'active'"
                  variant="ghost"
                  size="xs"
                  @click="cancelSubscription(row.id)"
                >
                  Отменить
                </Button>
                <Button
                  v-if="row.status === 'active'"
                  variant="ghost"
                  size="xs"
                  @click="extendSubscription(row.id)"
                >
                  Продлить
                </Button>
              </div>
            </template>
          </DataTable>
        </div>

        <!-- Payments tab -->
        <div v-if="activeTab === 'payments'">
          <DataTable
            :columns="paymentColumns"
            :data="detail.payments"
            :loading="false"
            empty-text="Нет платежей"
          >
            <template #id="{ value }">
              <span class="font-mono text-xs">{{ value }}</span>
            </template>
            <template #amount="{ value }">
              <span class="font-mono text-sm">{{ value }} ₽</span>
            </template>
            <template #status="{ value }">
              <Badge :variant="paymentStatusVariant(value)">{{ paymentStatusLabel(value) }}</Badge>
            </template>
            <template #created_at="{ value }">
              <span class="text-xs text-muted-foreground">{{ formatDate(value) }}</span>
            </template>
          </DataTable>
        </div>

        <!-- Audit tab -->
        <div v-if="activeTab === 'audit'">
          <DataTable
            :columns="auditColumns"
            :data="auditLogs"
            :loading="auditLoading"
            empty-text="Нет записей"
          >
            <template #id="{ value }">
              <span class="font-mono text-xs">{{ value }}</span>
            </template>
            <template #action="{ value }">
              <span class="text-sm font-medium">{{ value }}</span>
            </template>
            <template #ip_address="{ value }">
              <span class="font-mono text-xs text-muted-foreground">{{ value }}</span>
            </template>
            <template #created_at="{ value }">
              <span class="text-xs text-muted-foreground">{{ formatDate(value) }}</span>
            </template>
          </DataTable>
          <Pagination
            v-if="auditTotal > auditPageSize"
            class="mt-4"
            :page="auditPage"
            :total="auditTotal"
            :page-size="auditPageSize"
            @update:page="(v) => { auditPage = v; loadAuditLogs() }"
            @update:page-size="(v) => { auditPageSize = v; auditPage = 1; loadAuditLogs() }"
          />
        </div>
      </Tabs>
    </template>

    <!-- Reset password modal -->
    <Modal v-model:show="showResetPassword" title="Сбросить пароль">
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          Введите новый пароль для пользователя {{ user?.email || user?.phone }}:
        </p>
        <Input
          v-model="newPassword"
          type="password"
          placeholder="Новый пароль"
        />
      </div>
      <template #footer>
        <Button variant="ghost" @click="showResetPassword = false">Отмена</Button>
        <Button :disabled="!newPassword || newPassword.length < 6" @click="resetPassword">
          Сбросить
        </Button>
      </template>
    </Modal>

    <!-- Confirm dialog -->
    <ConfirmDialog
      v-model:show="showConfirmDelete"
      title="Удалить пользователя"
      :message="`Вы уверены, что хотите удалить ${user?.email || user?.phone}? Это действие необратимо.`"
      variant="destructive"
      confirm-text="Удалить"
      @confirm="deleteUser"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { format, formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import {
  ArrowLeft,
  Ban,
  CheckCircle,
  ShieldCheck,
  ShieldOff,
  KeyRound,
  Trash2,
} from 'lucide-vue-next'
import { adminApi } from '@/api/client'
import { useToast } from '@/composables/useToast'
import { getErrorMessage } from '@/utils/error'
import type { AdminUserDetail, AdminUser, AuditLog } from '@/api/types'
import type { Column } from '@/components/ui/DataTable.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Tabs from '@/components/ui/Tabs.vue'
import DataTable from '@/components/ui/DataTable.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Modal from '@/components/ui/Modal.vue'
import Input from '@/components/ui/Input.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()

// --- State ---
const detail = ref<AdminUserDetail | null>(null)
const loading = ref(false)
const activeTab = ref('info')

const showResetPassword = ref(false)
const newPassword = ref('')
const showConfirmDelete = ref(false)

// Audit logs pagination
const auditLogs = ref<AuditLog[]>([])
const auditLoading = ref(false)
const auditPage = ref(1)
const auditPageSize = ref(20)
const auditTotal = ref(0)

const user = computed<AdminUser>(() => detail.value?.user ?? {} as AdminUser)

const userId = computed(() => Number(route.params.id))

const initials = computed(() => {
  const name = user.value.display_name || user.value.email || user.value.phone || '?'
  return name.slice(0, 2).toUpperCase()
})

// --- Tabs ---
const tabs = [
  { key: 'info', label: 'Информация' },
  { key: 'tunnels', label: 'Тоннели' },
  { key: 'subscriptions', label: 'Подписки' },
  { key: 'payments', label: 'Платежи' },
  { key: 'audit', label: 'Аудит' },
]

// --- Table columns ---
const tunnelColumns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'tunnel_type', title: 'Тип', width: '80px' },
  { key: 'local_port', title: 'Локальный порт', width: '120px' },
  { key: 'url', title: 'URL' },
  { key: 'connected_at', title: 'Подключен', width: '140px' },
  { key: 'disconnected_at', title: 'Отключен', width: '140px' },
  { key: 'bytes_sent', title: 'Трафик', width: '160px' },
]

const subscriptionColumns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'plan', title: 'Тариф' },
  { key: 'status', title: 'Статус', width: '120px' },
  { key: 'current_period_end', title: 'До', width: '140px' },
  { key: 'actions', title: '', width: '180px', align: 'right' },
]

const paymentColumns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'invoice_id', title: 'Счет', width: '80px' },
  { key: 'amount', title: 'Сумма', width: '100px' },
  { key: 'status', title: 'Статус', width: '120px' },
  { key: 'created_at', title: 'Дата', width: '140px' },
]

const auditColumns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'action', title: 'Действие' },
  { key: 'ip_address', title: 'IP', width: '140px' },
  { key: 'created_at', title: 'Дата', width: '140px' },
]

// --- Load ---
async function loadUser() {
  loading.value = true
  try {
    const { data } = await adminApi.getUserDetail(userId.value)
    detail.value = data
  } catch (err) {
    toast.error(getErrorMessage(err, 'Ошибка загрузки пользователя'))
    router.push({ name: 'users' })
  } finally {
    loading.value = false
  }
}

async function loadAuditLogs() {
  auditLoading.value = true
  try {
    const { data } = await adminApi.listAuditLogs(auditPage.value, auditPageSize.value, userId.value)
    auditLogs.value = data.logs ?? []
    auditTotal.value = data.total ?? 0
  } catch {
    auditLogs.value = []
  } finally {
    auditLoading.value = false
  }
}

// --- Actions ---
async function toggleActive() {
  try {
    const newActive = !user.value.is_active
    await adminApi.updateUser(userId.value, { is_active: newActive })
    toast.success(newActive ? 'Пользователь разблокирован' : 'Пользователь заблокирован')
    loadUser()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function toggleAdmin() {
  try {
    const newAdmin = !user.value.is_admin
    await adminApi.updateUser(userId.value, { is_admin: newAdmin })
    toast.success(newAdmin ? 'Права админа назначены' : 'Права админа отозваны')
    loadUser()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function resetPassword() {
  if (!newPassword.value) return
  try {
    await adminApi.resetPassword(userId.value, newPassword.value)
    toast.success('Пароль успешно сброшен')
    showResetPassword.value = false
    newPassword.value = ''
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

function confirmDelete() {
  showConfirmDelete.value = true
}

async function deleteUser() {
  try {
    await adminApi.deleteUser(userId.value)
    toast.success('Пользователь удален')
    router.push({ name: 'users' })
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function cancelSubscription(subId: number) {
  try {
    await adminApi.cancelSubscription(subId)
    toast.success('Подписка отменена')
    loadUser()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function extendSubscription(subId: number) {
  try {
    await adminApi.extendSubscription(subId, 30)
    toast.success('Подписка продлена на 30 дней')
    loadUser()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

// --- Helpers ---
function formatDate(date: string): string {
  if (!date) return '---'
  return format(new Date(date), 'dd.MM.yyyy HH:mm')
}

function formatRelative(date: string): string {
  if (!date) return '---'
  return formatDistanceToNow(new Date(date), { addSuffix: true, locale: ru })
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function statusVariant(status: string) {
  switch (status) {
    case 'active': return 'success' as const
    case 'cancelled': return 'destructive' as const
    case 'expired': return 'warning' as const
    default: return 'default' as const
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'active': return 'Активна'
    case 'pending': return 'Ожидание'
    case 'cancelled': return 'Отменена'
    case 'expired': return 'Истекла'
    default: return status
  }
}

function paymentStatusVariant(status: string) {
  switch (status) {
    case 'success': return 'success' as const
    case 'failed': return 'destructive' as const
    default: return 'warning' as const
  }
}

function paymentStatusLabel(status: string) {
  switch (status) {
    case 'success': return 'Успешно'
    case 'pending': return 'Ожидание'
    case 'failed': return 'Ошибка'
    default: return status
  }
}

// --- Init ---
onMounted(() => {
  loadUser()
  loadAuditLogs()
})
</script>
