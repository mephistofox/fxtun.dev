<template>
  <n-spin :show="loading" style="min-height: 200px">
    <n-space vertical :size="24" v-if="detail">
      <!-- Header -->
      <n-space justify="space-between" align="center">
        <n-space align="center" :size="16">
          <n-button text @click="$router.push('/users')">
            <template #icon>
              <n-icon :component="ArrowBackOutline" />
            </template>
            Users
          </n-button>
          <n-avatar
            round
            :size="48"
            :src="detail.user.avatar_url || undefined"
          >
            {{ userInitials }}
          </n-avatar>
          <n-space vertical :size="0">
            <n-text strong style="font-size: 18px">
              {{ detail.user.display_name || detail.user.phone }}
            </n-text>
            <n-text depth="3">
              {{ detail.user.email || detail.user.phone }}
            </n-text>
          </n-space>
          <n-tag
            v-if="detail.user.is_admin"
            type="info"
            size="small"
            :bordered="false"
          >
            Admin
          </n-tag>
          <n-tag
            :type="detail.user.is_active ? 'success' : 'error'"
            size="small"
            :bordered="false"
          >
            {{ detail.user.is_active ? 'Active' : 'Blocked' }}
          </n-tag>
          <n-tag
            v-if="detail.user.plan"
            size="small"
            :bordered="false"
          >
            {{ detail.user.plan.name }}
          </n-tag>
        </n-space>
        <n-space :size="8">
          <n-button
            :type="detail.user.is_active ? 'warning' : 'success'"
            size="small"
            @click="toggleActive"
          >
            {{ detail.user.is_active ? 'Block' : 'Unblock' }}
          </n-button>
          <n-button size="small" @click="toggleAdmin">
            {{ detail.user.is_admin ? 'Remove Admin' : 'Make Admin' }}
          </n-button>
          <n-button size="small" @click="showResetPasswordModal = true">
            Reset Password
          </n-button>
          <n-button type="error" size="small" @click="handleDelete">
            Delete
          </n-button>
        </n-space>
      </n-space>

      <!-- Tabs -->
      <n-tabs type="line" animated>
        <!-- Info Tab -->
        <n-tab-pane name="info" tab="Info">
          <n-card size="small">
            <n-descriptions label-placement="left" :column="2" bordered>
              <n-descriptions-item label="ID">{{ detail.user.id }}</n-descriptions-item>
              <n-descriptions-item label="Phone">{{ detail.user.phone }}</n-descriptions-item>
              <n-descriptions-item label="Email">{{ detail.user.email || '-' }}</n-descriptions-item>
              <n-descriptions-item label="Display Name">{{ detail.user.display_name || '-' }}</n-descriptions-item>
              <n-descriptions-item label="Status">
                <n-tag
                  :type="detail.user.is_active ? 'success' : 'error'"
                  size="small"
                  :bordered="false"
                >
                  {{ detail.user.is_active ? 'Active' : 'Blocked' }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item label="Admin">
                <n-tag
                  :type="detail.user.is_admin ? 'info' : 'default'"
                  size="small"
                  :bordered="false"
                >
                  {{ detail.user.is_admin ? 'Yes' : 'No' }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item label="Plan">
                <n-space align="center" :size="8">
                  <n-select
                    v-model:value="selectedPlanId"
                    :options="planOptions"
                    size="small"
                    style="width: 160px"
                  />
                  <n-button
                    size="tiny"
                    type="primary"
                    :disabled="selectedPlanId === detail.user.plan_id"
                    @click="changePlan"
                  >
                    Save
                  </n-button>
                </n-space>
              </n-descriptions-item>
              <n-descriptions-item label="GitHub ID">{{ detail.user.github_id || '-' }}</n-descriptions-item>
              <n-descriptions-item label="Google ID">{{ detail.user.google_id || '-' }}</n-descriptions-item>
              <n-descriptions-item label="Created">{{ formatDate(detail.user.created_at) }}</n-descriptions-item>
              <n-descriptions-item label="Last Login">{{ detail.user.last_login_at ? formatDate(detail.user.last_login_at) : 'Never' }}</n-descriptions-item>
              <n-descriptions-item label="API Tokens">{{ detail.token_count }}</n-descriptions-item>
              <n-descriptions-item label="Custom Domains">{{ detail.domain_count }}</n-descriptions-item>
              <n-descriptions-item v-if="detail.tunnel_stats" label="Total Connections">
                {{ detail.tunnel_stats.total_connections }}
              </n-descriptions-item>
              <n-descriptions-item v-if="detail.tunnel_stats" label="Total Traffic">
                {{ formatBytes(detail.tunnel_stats.total_bytes_sent + detail.tunnel_stats.total_bytes_received) }}
              </n-descriptions-item>
            </n-descriptions>
          </n-card>
        </n-tab-pane>

        <!-- Active Tunnels Tab -->
        <n-tab-pane name="tunnels" tab="Active Tunnels">
          <n-data-table
            :columns="tunnelColumns"
            :data="activeTunnels"
            :loading="tunnelsLoading"
            :bordered="false"
            size="small"
          />
          <n-empty v-if="!tunnelsLoading && activeTunnels.length === 0" description="No active tunnels" />
        </n-tab-pane>

        <!-- Subscriptions Tab -->
        <n-tab-pane name="subscriptions" tab="Subscriptions">
          <n-data-table
            :columns="subscriptionColumns"
            :data="detail.subscriptions"
            :bordered="false"
            size="small"
          />
          <n-empty v-if="detail.subscriptions.length === 0" description="No subscriptions" />
        </n-tab-pane>

        <!-- Payments Tab -->
        <n-tab-pane name="payments" tab="Payments">
          <n-data-table
            :columns="paymentColumns"
            :data="detail.payments"
            :bordered="false"
            size="small"
          />
          <n-empty v-if="detail.payments.length === 0" description="No payments" />
        </n-tab-pane>

        <!-- Audit Tab -->
        <n-tab-pane name="audit" tab="Audit Log">
          <n-data-table
            :columns="auditColumns"
            :data="auditLogs"
            :loading="auditLoading"
            :bordered="false"
            size="small"
          />
        </n-tab-pane>
      </n-tabs>

      <!-- Reset Password Modal -->
      <n-modal
        v-model:show="showResetPasswordModal"
        preset="dialog"
        title="Reset Password"
        positive-text="Reset"
        negative-text="Cancel"
        :positive-button-props="{ disabled: newPassword.length < 8 }"
        @positive-click="handleResetPassword"
      >
        <n-space vertical :size="8">
          <n-text>Enter new password (min 8 characters):</n-text>
          <n-input
            v-model:value="newPassword"
            type="password"
            placeholder="New password"
            show-password-on="click"
            :minlength="8"
          />
        </n-space>
      </n-modal>
    </n-space>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NSpace,
  NButton,
  NText,
  NTag,
  NCard,
  NTabs,
  NTabPane,
  NDescriptions,
  NDescriptionsItem,
  NDataTable,
  NAvatar,
  NSelect,
  NModal,
  NInput,
  NIcon,
  NSpin,
  NEmpty,
  useMessage,
  useDialog,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { ArrowBackOutline } from '@vicons/ionicons5'
import { format } from 'date-fns'

import { adminApi } from '@/api/client'
import type {
  AdminUserDetail,
  AdminTunnel,
  AdminSubscription,
  Payment,
  AuditLog,
  Plan,
} from '@/api/types'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const userId = computed(() => Number(route.params.id))

// State
const detail = ref<AdminUserDetail | null>(null)
const loading = ref(false)
const plans = ref<Plan[]>([])
const selectedPlanId = ref<number>(0)
const planOptions = ref<Array<{ label: string; value: number }>>([])

// Reset password
const showResetPasswordModal = ref(false)
const newPassword = ref('')

// Active tunnels
const activeTunnels = ref<AdminTunnel[]>([])
const tunnelsLoading = ref(false)

// Audit logs
const auditLogs = ref<AuditLog[]>([])
const auditLoading = ref(false)

const userInitials = computed(() => {
  if (!detail.value) return '?'
  const name = detail.value.user.display_name || detail.value.user.phone
  return name.slice(0, 2).toUpperCase()
})

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'MMM d, yyyy HH:mm')
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

// Fetch data
async function fetchDetail() {
  loading.value = true
  try {
    const resp = await adminApi.getUserDetail(userId.value)
    detail.value = resp.data
    selectedPlanId.value = resp.data.user.plan_id
  } catch {
    message.error('Failed to load user detail')
    router.push('/users')
  } finally {
    loading.value = false
  }
}

async function fetchPlans() {
  try {
    const resp = await adminApi.listPlans()
    plans.value = resp.data.plans || []
    planOptions.value = plans.value.map((p) => ({ label: p.name, value: p.id }))
  } catch {
    // ignore
  }
}

async function fetchActiveTunnels() {
  tunnelsLoading.value = true
  try {
    const resp = await adminApi.listTunnels({ user_id: userId.value })
    activeTunnels.value = resp.data.tunnels || []
  } catch {
    // ignore - user might not have active tunnels
  } finally {
    tunnelsLoading.value = false
  }
}

async function fetchAuditLogs() {
  auditLoading.value = true
  try {
    const resp = await adminApi.listAuditLogs(1, 50, userId.value)
    auditLogs.value = resp.data.logs || []
  } catch {
    message.error('Failed to load audit logs')
  } finally {
    auditLoading.value = false
  }
}

// Actions
async function toggleActive() {
  if (!detail.value) return
  const user = detail.value.user
  const action = user.is_active ? 'Block' : 'Unblock'

  dialog.warning({
    title: `${action} User`,
    content: `${action} ${user.display_name || user.phone}?`,
    positiveText: action,
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.updateUser(user.id, { is_active: !user.is_active })
        message.success(`User ${action.toLowerCase()}ed`)
        fetchDetail()
      } catch {
        message.error(`Failed to ${action.toLowerCase()} user`)
      }
    },
  })
}

async function toggleAdmin() {
  if (!detail.value) return
  const user = detail.value.user
  try {
    await adminApi.updateUser(user.id, { is_admin: !user.is_admin })
    message.success(user.is_admin ? 'Admin removed' : 'Admin granted')
    fetchDetail()
  } catch {
    message.error('Failed to update admin status')
  }
}

async function changePlan() {
  if (!detail.value) return
  try {
    await adminApi.updateUser(detail.value.user.id, { plan_id: selectedPlanId.value })
    message.success('Plan updated')
    fetchDetail()
  } catch {
    message.error('Failed to update plan')
  }
}

async function handleResetPassword() {
  if (newPassword.value.length < 8) return
  try {
    await adminApi.resetPassword(userId.value, newPassword.value)
    message.success('Password reset successfully')
    newPassword.value = ''
    showResetPasswordModal.value = false
  } catch {
    message.error('Failed to reset password')
  }
}

function handleDelete() {
  if (!detail.value) return
  const user = detail.value.user

  dialog.error({
    title: 'Delete User',
    content: `Permanently delete ${user.display_name || user.phone}? This cannot be undone.`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.deleteUser(user.id)
        message.success('User deleted')
        router.push('/users')
      } catch {
        message.error('Failed to delete user')
      }
    },
  })
}

// Table columns
const tunnelColumns: DataTableColumns<AdminTunnel> = [
  { title: 'ID', key: 'id', width: 80, ellipsis: { tooltip: true } },
  {
    title: 'Type',
    key: 'type',
    width: 80,
    render(row) {
      const typeColors: Record<string, string> = {
        http: 'success',
        tcp: 'info',
        udp: 'warning',
      }
      return h(
        NTag,
        { type: (typeColors[row.type] || 'default') as 'success' | 'info' | 'warning' | 'default', size: 'small', bordered: false },
        { default: () => row.type.toUpperCase() },
      )
    },
  },
  { title: 'Name', key: 'name', ellipsis: { tooltip: true } },
  { title: 'Subdomain', key: 'subdomain', width: 150, render(row) { return row.subdomain || '-' } },
  { title: 'Remote Port', key: 'remote_port', width: 110, render(row) { return row.remote_port ?? '-' } },
  { title: 'Local Port', key: 'local_port', width: 100 },
  {
    title: 'Created',
    key: 'created_at',
    width: 150,
    render(row) {
      return format(new Date(row.created_at), 'MMM d, HH:mm')
    },
  },
]

const subscriptionColumns: DataTableColumns<AdminSubscription> = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: 'Plan',
    key: 'plan',
    width: 120,
    render(row) {
      return row.plan?.name || `#${row.plan_id}`
    },
  },
  {
    title: 'Status',
    key: 'status',
    width: 100,
    render(row) {
      const statusType: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
        active: 'success',
        pending: 'warning',
        cancelled: 'error',
        expired: 'default',
      }
      return h(
        NTag,
        { type: statusType[row.status] || 'default', size: 'small', bordered: false },
        { default: () => row.status },
      )
    },
  },
  {
    title: 'Recurring',
    key: 'recurring',
    width: 90,
    render(row) {
      return row.recurring ? 'Yes' : 'No'
    },
  },
  {
    title: 'Period Start',
    key: 'current_period_start',
    width: 130,
    render(row) {
      return row.current_period_start ? format(new Date(row.current_period_start), 'MMM d, yyyy') : '-'
    },
  },
  {
    title: 'Period End',
    key: 'current_period_end',
    width: 130,
    render(row) {
      return row.current_period_end ? format(new Date(row.current_period_end), 'MMM d, yyyy') : '-'
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 140,
    render(row) {
      if (row.status !== 'active') return '-'
      return h(NSpace, { size: 4 }, {
        default: () => [
          h(NButton, { size: 'tiny', onClick: () => cancelSubscription(row.id) }, { default: () => 'Cancel' }),
          h(NButton, { size: 'tiny', type: 'primary', onClick: () => extendSubscription(row.id) }, { default: () => 'Extend' }),
        ],
      })
    },
  },
]

const paymentColumns: DataTableColumns<Payment> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: 'Invoice', key: 'invoice_id', width: 90 },
  {
    title: 'Amount',
    key: 'amount',
    width: 100,
    render(row) {
      return `${row.currency === 'RUB' ? '' : '$'}${row.amount}${row.currency === 'RUB' ? ' RUB' : ''}`
    },
  },
  {
    title: 'Status',
    key: 'status',
    width: 90,
    render(row) {
      const statusType: Record<string, 'success' | 'warning' | 'error'> = {
        success: 'success',
        pending: 'warning',
        failed: 'error',
      }
      return h(
        NTag,
        { type: statusType[row.status] || 'default', size: 'small', bordered: false },
        { default: () => row.status },
      )
    },
  },
  {
    title: 'Provider',
    key: 'provider',
    width: 100,
  },
  {
    title: 'Recurring',
    key: 'is_recurring',
    width: 90,
    render(row) {
      return row.is_recurring ? 'Yes' : 'No'
    },
  },
  {
    title: 'Date',
    key: 'created_at',
    width: 140,
    render(row) {
      return format(new Date(row.created_at), 'MMM d, yyyy HH:mm')
    },
  },
]

const auditColumns: DataTableColumns<AuditLog> = [
  {
    title: 'Time',
    key: 'created_at',
    width: 160,
    render(row) {
      return format(new Date(row.created_at), 'MMM d, HH:mm:ss')
    },
  },
  { title: 'Action', key: 'action', ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip_address', width: 130 },
  {
    title: 'Details',
    key: 'details',
    ellipsis: { tooltip: true },
    render(row) {
      if (!row.details) return '-'
      return JSON.stringify(row.details)
    },
  },
]

// Subscription actions
async function cancelSubscription(id: number) {
  dialog.warning({
    title: 'Cancel Subscription',
    content: 'Cancel this subscription?',
    positiveText: 'Cancel Subscription',
    negativeText: 'Go Back',
    onPositiveClick: async () => {
      try {
        await adminApi.cancelSubscription(id)
        message.success('Subscription cancelled')
        fetchDetail()
      } catch {
        message.error('Failed to cancel subscription')
      }
    },
  })
}

async function extendSubscription(id: number) {
  dialog.info({
    title: 'Extend Subscription',
    content: 'Extend by 30 days?',
    positiveText: 'Extend',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.extendSubscription(id, 30)
        message.success('Subscription extended by 30 days')
        fetchDetail()
      } catch {
        message.error('Failed to extend subscription')
      }
    },
  })
}

watch(userId, (newId) => {
  if (newId) {
    fetchDetail()
    fetchActiveTunnels()
    fetchAuditLogs()
  }
})

onMounted(() => {
  fetchDetail()
  fetchPlans()
  fetchActiveTunnels()
  fetchAuditLogs()
})
</script>
