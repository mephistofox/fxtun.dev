<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space align="center" justify="space-between">
      <n-select
        v-model:value="statusFilter"
        :options="statusOptions"
        style="width: 180px"
        @update:value="fetchSubscriptions"
      />
      <n-pagination
        v-model:page="currentPage"
        :page-count="totalPages"
        :page-slot="7"
        @update:page="fetchSubscriptions"
      />
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="subscriptions"
      :loading="loading"
      :row-key="(row: AdminSubscription) => row.id"
    />

    <!-- Extend Modal -->
    <n-modal
      v-model:show="showExtendModal"
      preset="card"
      title="Extend Subscription"
      style="width: 400px"
      :mask-closable="false"
    >
      <n-form-item label="Days to extend">
        <n-input-number v-model:value="extendDays" :min="1" :max="365" style="width: 100%" />
      </n-form-item>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showExtendModal = false">Cancel</n-button>
          <n-button type="primary" :loading="extending" @click="handleExtendConfirm">Extend</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useMessage, useDialog, NTag, NButton, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format } from 'date-fns'
import { adminApi } from '@/api/client'
import type { AdminSubscription } from '@/api/types'

const message = useMessage()
const dialog = useDialog()

const subscriptions = ref<AdminSubscription[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 20
const statusFilter = ref<string | null>(null)

// Extend modal
const showExtendModal = ref(false)
const extendDays = ref(30)
const extending = ref(false)
const extendingId = ref<number | null>(null)

const statusOptions = [
  { label: 'All Statuses', value: '' },
  { label: 'Active', value: 'active' },
  { label: 'Cancelled', value: 'cancelled' },
  { label: 'Expired', value: 'expired' },
  { label: 'Pending', value: 'pending' },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const statusTagType: Record<string, 'success' | 'warning' | 'error' | 'info' | 'default'> = {
  active: 'success',
  cancelled: 'error',
  expired: 'warning',
  pending: 'info',
}

const columns: DataTableColumns<AdminSubscription> = [
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
    title: 'Plan',
    key: 'plan',
    width: 120,
    render(row) {
      return row.plan?.name || `Plan #${row.plan_id}`
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
    key: 'recurring',
    width: 90,
    render(row) {
      return h(NTag, { type: row.recurring ? 'info' : 'default', size: 'small' }, {
        default: () => row.recurring ? 'Yes' : 'No',
      })
    },
  },
  {
    title: 'Period Start',
    key: 'current_period_start',
    width: 140,
    render(row) {
      return row.current_period_start ? format(new Date(row.current_period_start), 'yyyy-MM-dd') : '-'
    },
  },
  {
    title: 'Period End',
    key: 'current_period_end',
    width: 140,
    render(row) {
      return row.current_period_end ? format(new Date(row.current_period_end), 'yyyy-MM-dd') : '-'
    },
  },
  {
    title: 'Created At',
    key: 'created_at',
    width: 140,
    render(row) {
      return row.created_at ? format(new Date(row.created_at), 'yyyy-MM-dd HH:mm') : '-'
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 160,
    render(row) {
      const buttons: ReturnType<typeof h>[] = []
      if (row.status === 'active') {
        buttons.push(
          h(NButton, { size: 'small', type: 'info', quaternary: true, onClick: () => openExtendModal(row) }, { default: () => 'Extend' }),
          h(NButton, { size: 'small', type: 'error', quaternary: true, onClick: () => handleCancel(row) }, { default: () => 'Cancel' }),
        )
      }
      if (row.status === 'expired') {
        buttons.push(
          h(NButton, { size: 'small', type: 'info', quaternary: true, onClick: () => openExtendModal(row) }, { default: () => 'Extend' }),
        )
      }
      return buttons.length > 0 ? h(NSpace, { size: 4 }, { default: () => buttons }) : '-'
    },
  },
]

function openExtendModal(sub: AdminSubscription) {
  extendingId.value = sub.id
  extendDays.value = 30
  showExtendModal.value = true
}

async function handleExtendConfirm() {
  if (!extendingId.value) return
  extending.value = true
  try {
    await adminApi.extendSubscription(extendingId.value, extendDays.value)
    message.success('Subscription extended')
    showExtendModal.value = false
    await fetchSubscriptions()
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to extend subscription')
  } finally {
    extending.value = false
  }
}

function handleCancel(sub: AdminSubscription) {
  dialog.warning({
    title: 'Cancel Subscription',
    content: `Cancel subscription #${sub.id} for ${sub.user_phone || sub.user_email}?`,
    positiveText: 'Cancel Subscription',
    negativeText: 'Keep',
    onPositiveClick: async () => {
      try {
        await adminApi.cancelSubscription(sub.id)
        message.success('Subscription cancelled')
        await fetchSubscriptions()
      } catch (err: unknown) {
        const error = err as { response?: { data?: { error?: string } }; message?: string }
        message.error(error.response?.data?.error || error.message || 'Failed to cancel subscription')
      }
    },
  })
}

async function fetchSubscriptions() {
  loading.value = true
  try {
    const { data } = await adminApi.listSubscriptions(currentPage.value, pageSize, statusFilter.value || undefined)
    subscriptions.value = data.subscriptions || []
    total.value = data.total || 0
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load subscriptions')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSubscriptions()
})
</script>
