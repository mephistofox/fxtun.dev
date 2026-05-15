<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Подписки</h1>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <Select
        v-model="statusFilter"
        :options="statusOptions"
        placeholder="Статус"
        class="w-48"
      />
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="subscriptions"
      :loading="loading"
      row-key="id"
      empty-text="Нет подписок"
    >
      <template #id="{ value }">
        <span class="font-mono text-sm text-muted-foreground">{{ value }}</span>
      </template>

      <template #user="{ row }">
        <span class="text-sm">{{ row.user_email || row.user_phone }}</span>
      </template>

      <template #plan="{ row }">
        <span class="text-sm">{{ row.plan?.name || `Plan #${row.plan_id}` }}</span>
      </template>

      <template #status="{ value }">
        <Badge :variant="subscriptionStatusBadge(value)">{{ subscriptionStatusLabel(value) }}</Badge>
      </template>

      <template #recurring="{ value }">
        <span class="text-sm">{{ value ? 'Да' : 'Нет' }}</span>
      </template>

      <template #current_period_start="{ value }">
        <span class="text-sm text-muted-foreground">{{ value ? formatDate(value) : '-' }}</span>
      </template>

      <template #current_period_end="{ value }">
        <span class="text-sm text-muted-foreground">{{ value ? formatDate(value) : '-' }}</span>
      </template>

      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
      </template>

      <template #actions="{ row }">
        <Dropdown
          v-if="row.status === 'active'"
          :items="activeActions"
          @select="(key) => handleAction(key, row)"
        >
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      v-if="total > pageSize"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="(p) => { page = p; fetchSubscriptions() }"
      @update:page-size="(s) => { pageSize = s; page = 1; fetchSubscriptions() }"
    />

    <!-- Extend modal -->
    <Modal v-model:show="showExtendModal" title="Продлить подписку" width="max-w-sm">
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          Укажите количество дней для продления подписки.
        </p>
        <div>
          <label class="block text-sm font-medium text-foreground mb-1.5">Количество дней</label>
          <Input v-model="extendDays" type="number" placeholder="30" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="showExtendModal = false">Отмена</Button>
        <Button :loading="extending" @click="extendSubscription">Продлить</Button>
      </template>
    </Modal>

    <!-- Cancel confirm -->
    <ConfirmDialog
      v-model:show="showCancelConfirm"
      title="Отменить подписку"
      :message="`Отменить подписку #${cancellingId}? Пользователь потеряет доступ к тарифу.`"
      confirm-text="Отменить подписку"
      variant="destructive"
      @confirm="cancelSubscription"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { AdminSubscription } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import { MoreHorizontal } from 'lucide-vue-next'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Select from '@/components/ui/Select.vue'
import Input from '@/components/ui/Input.vue'
import Modal from '@/components/ui/Modal.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import Pagination from '@/components/ui/Pagination.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const subscriptions = ref<AdminSubscription[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const statusFilter = ref<string | number | null>('all')

const showExtendModal = ref(false)
const extending = ref(false)
const extendDays = ref<string | number>(30)
const extendingId = ref<number | null>(null)

const showCancelConfirm = ref(false)
const cancellingId = ref<number | null>(null)

const statusOptions = [
  { value: 'all', label: 'Все' },
  { value: 'active', label: 'Активные' },
  { value: 'cancelled', label: 'Отменённые' },
  { value: 'expired', label: 'Истекшие' },
  { value: 'pending', label: 'Ожидают' },
]

const columns: Column[] = [
  { key: 'id', title: 'ID', width: '70px' },
  { key: 'user', title: 'Пользователь' },
  { key: 'plan', title: 'Тариф' },
  { key: 'status', title: 'Статус', width: '120px' },
  { key: 'recurring', title: 'Повторяемая', width: '110px' },
  { key: 'current_period_start', title: 'Начало периода', width: '140px' },
  { key: 'current_period_end', title: 'Конец периода', width: '140px' },
  { key: 'created_at', title: 'Создана', width: '140px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

const activeActions = [
  { key: 'extend', label: 'Продлить' },
  { key: 'cancel', label: 'Отменить', destructive: true },
]

function subscriptionStatusBadge(status: string): 'success' | 'destructive' | 'warning' | 'outline' {
  if (status === 'active') return 'success'
  if (status === 'cancelled') return 'destructive'
  if (status === 'pending') return 'warning'
  return 'outline'
}

function subscriptionStatusLabel(status: string): string {
  if (status === 'active') return 'Активна'
  if (status === 'cancelled') return 'Отменена'
  if (status === 'pending') return 'Ожидает'
  if (status === 'expired') return 'Истекла'
  return status
}

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

function handleAction(key: string, row: AdminSubscription) {
  if (key === 'extend') {
    extendingId.value = row.id
    extendDays.value = 30
    showExtendModal.value = true
  } else if (key === 'cancel') {
    cancellingId.value = row.id
    showCancelConfirm.value = true
  }
}

async function fetchSubscriptions() {
  loading.value = true
  try {
    const status = statusFilter.value === 'all' ? undefined : String(statusFilter.value)
    const { data } = await adminApi.listSubscriptions(page.value, pageSize.value, status)
    subscriptions.value = data.subscriptions || []
    total.value = data.total
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function extendSubscription() {
  if (!extendingId.value) return
  extending.value = true
  try {
    await adminApi.extendSubscription(extendingId.value, Number(extendDays.value))
    showExtendModal.value = false
    await fetchSubscriptions()
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    extending.value = false
  }
}

async function cancelSubscription() {
  if (!cancellingId.value) return
  try {
    await adminApi.cancelSubscription(cancellingId.value)
    await fetchSubscriptions()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

watch(statusFilter, () => {
  page.value = 1
  fetchSubscriptions()
})

onMounted(() => {
  fetchSubscriptions()
})
</script>
