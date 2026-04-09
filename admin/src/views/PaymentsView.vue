<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space align="center" justify="space-between">
      <n-select
        v-model:value="statusFilter"
        :options="statusOptions"
        style="width: 180px"
        @update:value="handleFilterChange"
      />
      <n-pagination
        v-model:page="currentPage"
        :page-count="totalPages"
        :page-slot="7"
        @update:page="fetchPayments"
      />
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="payments"
      :loading="loading"
      :row-key="(row: AdminPayment) => row.id"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useMessage, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format } from 'date-fns'
import { adminApi } from '@/api/client'
import type { AdminPayment } from '@/api/types'

const message = useMessage()

const payments = ref<AdminPayment[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 20
const statusFilter = ref('')

const statusOptions = [
  { label: 'All Statuses', value: '' },
  { label: 'Success', value: 'success' },
  { label: 'Pending', value: 'pending' },
  { label: 'Failed', value: 'failed' },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const statusTagType: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  success: 'success',
  pending: 'warning',
  failed: 'error',
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount)
}

const columns: DataTableColumns<AdminPayment> = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: 'User',
    key: 'user',
    width: 160,
    render(row) {
      return row.user_phone || row.user_email || '-'
    },
  },
  {
    title: 'Amount',
    key: 'amount',
    width: 120,
    render(row) {
      return formatCurrency(row.amount)
    },
  },
  {
    title: 'Status',
    key: 'status',
    width: 100,
    render(row) {
      return h(NTag, { type: statusTagType[row.status] || 'default', size: 'small' }, {
        default: () => row.status.charAt(0).toUpperCase() + row.status.slice(1),
      })
    },
  },
  {
    title: 'Recurring',
    key: 'is_recurring',
    width: 90,
    render(row) {
      return h(NTag, { type: row.is_recurring ? 'info' : 'default', size: 'small' }, {
        default: () => row.is_recurring ? 'Yes' : 'No',
      })
    },
  },
  {
    title: 'Invoice ID',
    key: 'invoice_id',
    width: 100,
  },
  {
    title: 'Created At',
    key: 'created_at',
    width: 160,
    render(row) {
      return row.created_at ? format(new Date(row.created_at), 'yyyy-MM-dd HH:mm') : '-'
    },
  },
]

function handleFilterChange() {
  currentPage.value = 1
  fetchPayments()
}

async function fetchPayments() {
  loading.value = true
  try {
    const { data } = await adminApi.listPayments(currentPage.value, pageSize, statusFilter.value || undefined)
    payments.value = data.payments || []
    total.value = data.total || 0
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load payments')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchPayments()
})
</script>
