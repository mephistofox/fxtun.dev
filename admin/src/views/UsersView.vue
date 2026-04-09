<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-display font-bold text-foreground">Пользователи</h1>
        <div v-if="userStats" class="flex gap-2 mt-2">
          <Badge>Всего: {{ userStats.total }}</Badge>
          <Badge variant="success">Активные: {{ userStats.active }}</Badge>
          <Badge variant="destructive">Заблокированные: {{ userStats.blocked }}</Badge>
          <Badge variant="accent">Админы: {{ userStats.admins }}</Badge>
        </div>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col sm:flex-row gap-3">
      <Input
        v-model="search"
        placeholder="Поиск по email, телефону, имени..."
        class="sm:max-w-xs"
      />
      <Select
        v-model="filter"
        :options="filterOptions"
        class="sm:w-48"
      />
      <Select
        v-model="sortBy"
        :options="sortOptions"
        class="sm:w-52"
      />
    </div>

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.length > 0"
      class="flex items-center gap-3 rounded-lg border border-primary/20 bg-primary/5 px-4 py-3"
    >
      <span class="text-sm font-medium text-foreground">Выбрано: {{ selectedIds.length }}</span>
      <div class="flex gap-2 ml-auto">
        <Button variant="outline" size="sm" @click="bulkAction('block')">
          <Ban class="h-4 w-4" /> Заблокировать
        </Button>
        <Button variant="outline" size="sm" @click="bulkAction('unblock')">
          <CheckCircle class="h-4 w-4" /> Разблокировать
        </Button>
        <Button variant="outline" size="sm" @click="showBulkPlanChange = true">
          <CreditCard class="h-4 w-4" /> Сменить тариф
        </Button>
        <Button variant="destructive" size="sm" @click="bulkAction('delete')">
          <Trash2 class="h-4 w-4" /> Удалить
        </Button>
      </div>
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="users"
      :loading="loading"
      selectable
      v-model:selected-keys="selectedIds"
      row-key="id"
    >
      <template #id="{ value }">
        <span class="font-mono text-xs text-muted-foreground">{{ value }}</span>
      </template>

      <template #email="{ row }">
        <div class="min-w-0">
          <p class="text-sm text-foreground truncate">{{ row.email || row.phone }}</p>
          <p v-if="row.email && row.phone" class="text-xs text-muted-foreground truncate">{{ row.phone }}</p>
        </div>
      </template>

      <template #display_name="{ value }">
        <span class="text-sm">{{ value || '---' }}</span>
      </template>

      <template #is_active="{ row }">
        <Badge :variant="row.is_active ? 'success' : 'destructive'">
          {{ row.is_active ? 'Активен' : 'Заблокирован' }}
        </Badge>
      </template>

      <template #is_admin="{ row }">
        <Badge v-if="row.is_admin" variant="accent">Админ</Badge>
      </template>

      <template #plan="{ row }">
        <Badge variant="outline">{{ row.plan?.name || `#${row.plan_id}` }}</Badge>
      </template>

      <template #created_at="{ value }">
        <span class="text-xs text-muted-foreground whitespace-nowrap">{{ formatDate(value) }}</span>
      </template>

      <template #actions="{ row }">
        <Dropdown :items="getRowActions(row)" @select="(key) => handleAction(key, row)">
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="(v) => { page = v; loadUsers() }"
      @update:page-size="(v) => { pageSize = v; page = 1; loadUsers() }"
    />

    <!-- Confirm dialog -->
    <ConfirmDialog
      v-model:show="confirmDialog.show"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :variant="confirmDialog.variant"
      :confirm-text="confirmDialog.confirmText"
      @confirm="confirmDialog.onConfirm"
    />

    <!-- Bulk plan change modal -->
    <Modal v-model:show="showBulkPlanChange" title="Сменить тариф">
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          Выберите новый тариф для {{ selectedIds.length }} пользователей:
        </p>
        <Select
          v-model="bulkPlanId"
          :options="planOptions"
          placeholder="Выберите тариф"
        />
      </div>
      <template #footer>
        <Button variant="ghost" @click="showBulkPlanChange = false">Отмена</Button>
        <Button :disabled="!bulkPlanId" @click="executeBulkPlanChange">Применить</Button>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { format } from 'date-fns'
import {
  MoreHorizontal,
  Ban,
  CheckCircle,
  ShieldCheck,
  ShieldOff,
  KeyRound,
  Trash2,
  CreditCard,
} from 'lucide-vue-next'
import { adminApi } from '@/api/client'
import { useToast } from '@/composables/useToast'
import { getErrorMessage } from '@/utils/error'
import type { AdminUser, UserStats, Plan } from '@/api/types'
import type { Column } from '@/components/ui/DataTable.vue'
import DataTable from '@/components/ui/DataTable.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import Modal from '@/components/ui/Modal.vue'

const router = useRouter()
const toast = useToast()

// --- State ---
const users = ref<AdminUser[]>([])
const userStats = ref<UserStats | null>(null)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const search = ref('')
const filter = ref<string | number>('all')
const sortBy = ref<string | number>('newest')
const selectedIds = ref<(string | number)[]>([])

// Plans for bulk change
const plans = ref<Plan[]>([])
const showBulkPlanChange = ref(false)
const bulkPlanId = ref<string | number | null>(null)

const planOptions = ref<{ value: string | number; label: string }[]>([])

// Confirm dialog
const confirmDialog = ref({
  show: false,
  title: '',
  message: '',
  variant: 'default' as 'default' | 'destructive',
  confirmText: 'Подтвердить',
  onConfirm: () => {},
})

// --- Options ---
const filterOptions = [
  { value: 'all', label: 'Все' },
  { value: 'active', label: 'Активные' },
  { value: 'blocked', label: 'Заблокированные' },
  { value: 'admins', label: 'Админы' },
]

const sortOptions = [
  { value: 'newest', label: 'По дате (новые)' },
  { value: 'oldest', label: 'По дате (старые)' },
  { value: 'email', label: 'По email' },
  { value: 'name', label: 'По имени' },
]

const columns: Column[] = [
  { key: 'id', title: 'ID', width: '70px' },
  { key: 'email', title: 'Email/Телефон' },
  { key: 'display_name', title: 'Имя' },
  { key: 'is_active', title: 'Статус', width: '120px' },
  { key: 'is_admin', title: 'Админ', width: '90px' },
  { key: 'plan', title: 'Тариф', width: '100px' },
  { key: 'created_at', title: 'Создан', width: '140px' },
  { key: 'actions', title: '', width: '50px', align: 'center' },
]

// --- Debounced search ---
let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    loadUsers()
  }, 400)
})

watch(filter, () => {
  page.value = 1
  loadUsers()
})

watch(sortBy, () => {
  page.value = 1
  loadUsers()
})

// --- Load ---
async function loadUsers() {
  loading.value = true
  try {
    const sortMap: Record<string, { sort_by: string; order: string }> = {
      newest: { sort_by: 'created_at', order: 'desc' },
      oldest: { sort_by: 'created_at', order: 'asc' },
      email: { sort_by: 'email', order: 'asc' },
      name: { sort_by: 'display_name', order: 'asc' },
    }
    const sort = sortMap[sortBy.value as string] || sortMap.newest
    const { data } = await adminApi.listUsers(
      page.value,
      pageSize.value,
      filter.value as string,
      search.value,
      sort.sort_by,
      sort.order,
    )
    users.value = data.users ?? []
    total.value = data.total ?? 0
    if (data.stats) userStats.value = data.stats
  } catch (err) {
    toast.error(getErrorMessage(err, 'Ошибка загрузки пользователей'))
  } finally {
    loading.value = false
  }
}

async function loadPlans() {
  try {
    const { data } = await adminApi.listPlans()
    plans.value = data.plans ?? []
    planOptions.value = plans.value.map(p => ({ value: p.id, label: p.name }))
  } catch {
    // Non-critical
  }
}

// --- Row actions ---
function getRowActions(row: AdminUser) {
  return [
    {
      key: row.is_active ? 'block' : 'unblock',
      label: row.is_active ? 'Заблокировать' : 'Разблокировать',
      icon: row.is_active ? Ban : CheckCircle,
    },
    {
      key: row.is_admin ? 'remove-admin' : 'make-admin',
      label: row.is_admin ? 'Убрать админа' : 'Сделать админом',
      icon: row.is_admin ? ShieldOff : ShieldCheck,
    },
    {
      key: 'reset-password',
      label: 'Сбросить пароль',
      icon: KeyRound,
    },
    { key: 'divider', label: '', divider: true },
    {
      key: 'delete',
      label: 'Удалить',
      icon: Trash2,
      destructive: true,
    },
  ]
}

function handleAction(key: string, row: AdminUser) {
  switch (key) {
    case 'block':
      confirmDialog.value = {
        show: true,
        title: 'Заблокировать пользователя',
        message: `Заблокировать пользователя ${row.email || row.phone}?`,
        variant: 'destructive',
        confirmText: 'Заблокировать',
        onConfirm: () => toggleActive(row, false),
      }
      break
    case 'unblock':
      toggleActive(row, true)
      break
    case 'make-admin':
      confirmDialog.value = {
        show: true,
        title: 'Назначить администратором',
        message: `Сделать ${row.email || row.phone} администратором?`,
        variant: 'default',
        confirmText: 'Назначить',
        onConfirm: () => toggleAdmin(row, true),
      }
      break
    case 'remove-admin':
      toggleAdmin(row, false)
      break
    case 'reset-password':
      router.push({ name: 'user-detail', params: { id: row.id } })
      break
    case 'delete':
      confirmDialog.value = {
        show: true,
        title: 'Удалить пользователя',
        message: `Вы уверены, что хотите удалить ${row.email || row.phone}? Это действие необратимо.`,
        variant: 'destructive',
        confirmText: 'Удалить',
        onConfirm: () => deleteUser(row.id),
      }
      break
  }
}

// --- Actions ---
async function toggleActive(user: AdminUser, active: boolean) {
  try {
    await adminApi.updateUser(user.id, { is_active: active })
    toast.success(active ? 'Пользователь разблокирован' : 'Пользователь заблокирован')
    loadUsers()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function toggleAdmin(user: AdminUser, admin: boolean) {
  try {
    await adminApi.updateUser(user.id, { is_admin: admin })
    toast.success(admin ? 'Права админа назначены' : 'Права админа отозваны')
    loadUsers()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function deleteUser(id: number) {
  try {
    await adminApi.deleteUser(id)
    toast.success('Пользователь удален')
    loadUsers()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

// --- Bulk actions ---
async function bulkAction(action: string) {
  if (action === 'delete') {
    confirmDialog.value = {
      show: true,
      title: 'Удалить пользователей',
      message: `Удалить ${selectedIds.value.length} пользователей? Это действие необратимо.`,
      variant: 'destructive',
      confirmText: 'Удалить',
      onConfirm: () => executeBulk('delete'),
    }
    return
  }
  if (action === 'block') {
    confirmDialog.value = {
      show: true,
      title: 'Заблокировать пользователей',
      message: `Заблокировать ${selectedIds.value.length} пользователей?`,
      variant: 'destructive',
      confirmText: 'Заблокировать',
      onConfirm: () => executeBulk('block'),
    }
    return
  }
  await executeBulk(action)
}

async function executeBulk(action: string) {
  try {
    const result = await adminApi.bulkUsers(action, selectedIds.value as number[])
    toast.success(`Выполнено: ${result.data.success_count} успешно, ${result.data.error_count} ошибок`)
    selectedIds.value = []
    loadUsers()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

async function executeBulkPlanChange() {
  if (!bulkPlanId.value) return
  try {
    const result = await adminApi.bulkUsers('change_plan', selectedIds.value as number[], bulkPlanId.value as number)
    toast.success(`Тариф изменен: ${result.data.success_count} успешно`)
    selectedIds.value = []
    showBulkPlanChange.value = false
    loadUsers()
  } catch (err) {
    toast.error(getErrorMessage(err))
  }
}

// --- Helpers ---
function formatDate(date: string): string {
  if (!date) return '---'
  return format(new Date(date), 'dd.MM.yyyy HH:mm')
}

// --- Navigate to user detail on row click ---
// We handle this via the DataTable row click — but DataTable doesn't emit row click natively.
// The user can click the row's action dropdown or simply click the row area.
// For now, we rely on the action menu.

// --- Init ---
onMounted(() => {
  loadUsers()
  loadPlans()
})
</script>
