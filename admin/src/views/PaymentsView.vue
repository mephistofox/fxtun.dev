<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Платежи</h1>
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
      :data="payments"
      :loading="loading"
      row-key="id"
      empty-text="Нет платежей"
    >
      <template #id="{ value }">
        <span class="font-mono text-sm text-muted-foreground">{{ value }}</span>
      </template>

      <template #user="{ row }">
        <span class="text-sm">{{ row.user_email || row.user_phone }}</span>
      </template>

      <template #amount="{ row }">
        <span class="text-sm font-medium">{{ formatAmount(row.amount) }}</span>
      </template>

      <template #provider="{ row }">
        <Badge variant="outline">{{ detectProvider(row) }}</Badge>
      </template>

      <template #status="{ value }">
        <Badge :variant="paymentStatusBadge(value)">{{ paymentStatusLabel(value) }}</Badge>
      </template>

      <template #is_recurring="{ value }">
        <span class="text-sm">{{ value ? 'Да' : 'Нет' }}</span>
      </template>

      <template #invoice_id="{ value }">
        <span class="font-mono text-sm text-muted-foreground">{{ value || '-' }}</span>
      </template>

      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      v-if="total > pageSize"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="(p) => { page = p; fetchPayments() }"
      @update:page-size="(s) => { pageSize = s; page = 1; fetchPayments() }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { AdminPayment } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Select from '@/components/ui/Select.vue'
import Pagination from '@/components/ui/Pagination.vue'

const payments = ref<AdminPayment[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const statusFilter = ref<string | number | null>('all')

const statusOptions = [
  { value: 'all', label: 'Все' },
  { value: 'success', label: 'Успешные' },
  { value: 'pending', label: 'Ожидают' },
  { value: 'failed', label: 'Неудачные' },
]

const columns: Column[] = [
  { key: 'id', title: 'ID', width: '70px' },
  { key: 'user', title: 'Пользователь' },
  { key: 'amount', title: 'Сумма', width: '120px' },
  { key: 'provider', title: 'Провайдер', width: '120px' },
  { key: 'status', title: 'Статус', width: '120px' },
  { key: 'is_recurring', title: 'Повторяемый', width: '110px' },
  { key: 'invoice_id', title: 'Invoice ID', width: '120px' },
  { key: 'created_at', title: 'Дата', width: '160px' },
]

function formatAmount(amount: number): string {
  // Amounts >= 100 are likely RUB, < 100 are likely USD
  if (amount >= 100) {
    return `${amount.toLocaleString('ru-RU')} \u20BD`
  }
  return `$${amount}`
}

function detectProvider(payment: AdminPayment): string {
  // Heuristic based on amount - RUB payments use YooKassa, USD use Creem
  if (payment.amount >= 100) return 'YooKassa'
  return 'Creem'
}

function paymentStatusBadge(status: string): 'success' | 'warning' | 'destructive' {
  if (status === 'success') return 'success'
  if (status === 'pending') return 'warning'
  return 'destructive'
}

function paymentStatusLabel(status: string): string {
  if (status === 'success') return 'Успешный'
  if (status === 'pending') return 'Ожидает'
  if (status === 'failed') return 'Неудачный'
  return status
}

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

async function fetchPayments() {
  loading.value = true
  try {
    const status = statusFilter.value === 'all' ? undefined : String(statusFilter.value)
    const { data } = await adminApi.listPayments(page.value, pageSize.value, status)
    payments.value = data.payments || []
    total.value = data.total
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

watch(statusFilter, () => {
  page.value = 1
  fetchPayments()
})

onMounted(() => {
  fetchPayments()
})
</script>
