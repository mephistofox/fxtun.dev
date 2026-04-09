<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space justify="space-between" align="center" :wrap="false">
      <n-space align="center" :size="12">
        <n-input
          v-model:value="searchQuery"
          placeholder="Search users..."
          clearable
          style="width: 240px"
          @update:value="handleSearchDebounced"
        >
          <template #prefix>
            <n-icon :component="SearchOutline" />
          </template>
        </n-input>
        <n-select
          v-model:value="filterValue"
          :options="filterOptions"
          style="width: 140px"
          @update:value="fetchUsers"
        />
        <n-select
          v-model:value="sortValue"
          :options="sortOptions"
          style="width: 180px"
          @update:value="fetchUsers"
        />
      </n-space>
      <n-space align="center" :size="8">
        <n-tag size="small" :bordered="false">Total: {{ userStats.total }}</n-tag>
        <n-tag size="small" type="success" :bordered="false">Active: {{ userStats.active }}</n-tag>
        <n-tag size="small" type="error" :bordered="false">Blocked: {{ userStats.blocked }}</n-tag>
        <n-tag size="small" type="info" :bordered="false">Admins: {{ userStats.admins }}</n-tag>
      </n-space>
    </n-space>

    <!-- Bulk action bar -->
    <n-card v-if="checkedRowKeys.length > 0" size="small">
      <n-space align="center" :size="12">
        <n-text>{{ checkedRowKeys.length }} selected</n-text>
        <n-button size="small" @click="handleBulk('block')">Block</n-button>
        <n-button size="small" @click="handleBulk('unblock')">Unblock</n-button>
        <n-button size="small" @click="showChangePlanDialog = true">Change Plan</n-button>
        <n-button size="small" type="error" @click="handleBulk('delete')">Delete</n-button>
      </n-space>
    </n-card>

    <!-- Users table -->
    <n-data-table
      :columns="columns"
      :data="users"
      :loading="loading"
      :bordered="false"
      :row-key="(row: AdminUser) => row.id"
      v-model:checked-row-keys="checkedRowKeys"
      :row-props="rowProps"
      size="small"
      :scroll-x="1000"
    />

    <!-- Pagination -->
    <n-space justify="end">
      <n-pagination
        v-model:page="currentPage"
        v-model:page-size="pageSize"
        :item-count="totalUsers"
        :page-sizes="[10, 20, 50, 100]"
        show-size-picker
        @update:page="fetchUsers"
        @update:page-size="handlePageSizeChange"
      />
    </n-space>

    <!-- Change Plan Dialog -->
    <n-modal v-model:show="showChangePlanDialog" preset="dialog" title="Change Plan">
      <n-space vertical :size="12">
        <n-text>Select plan for {{ checkedRowKeys.length }} users:</n-text>
        <n-select
          v-model:value="selectedPlanId"
          :options="planOptions"
          placeholder="Select plan"
        />
      </n-space>
      <template #action>
        <n-button @click="showChangePlanDialog = false">Cancel</n-button>
        <n-button
          type="primary"
          :disabled="selectedPlanId === null"
          @click="handleChangePlan"
        >
          Apply
        </n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, h, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NSpace,
  NInput,
  NSelect,
  NTag,
  NCard,
  NButton,
  NText,
  NDataTable,
  NPagination,
  NModal,
  NIcon,
  NDropdown,
  useMessage,
  useDialog,
} from 'naive-ui'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { SearchOutline, EllipsisVertical } from '@vicons/ionicons5'
import { format } from 'date-fns'

import { adminApi } from '@/api/client'
import type { AdminUser, UserStats, Plan } from '@/api/types'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()

// State
const users = ref<AdminUser[]>([])
const loading = ref(false)
const totalUsers = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const searchQuery = ref('')
const filterValue = ref('all')
const sortValue = ref('created_desc')
const checkedRowKeys = ref<DataTableRowKey[]>([])
const userStats = ref<UserStats>({ total: 0, active: 0, blocked: 0, admins: 0 })

// Plans for change plan dialog
const plans = ref<Plan[]>([])
const showChangePlanDialog = ref(false)
const selectedPlanId = ref<number | null>(null)

const planOptions = ref<Array<{ label: string; value: number }>>([])

const filterOptions = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Blocked', value: 'blocked' },
  { label: 'Admins', value: 'admins' },
]

const sortOptions = [
  { label: 'Created (newest)', value: 'created_desc' },
  { label: 'Created (oldest)', value: 'created_asc' },
  { label: 'Last login', value: 'last_login_desc' },
  { label: 'Email', value: 'email_asc' },
]

function getSortParams(): { sortBy: string; order: string } {
  const parts = sortValue.value.split('_')
  const order = parts.pop() || 'desc'
  const sortBy = parts.join('_')
  return { sortBy, order }
}

// Debounce
let searchTimer: ReturnType<typeof setTimeout> | null = null
function handleSearchDebounced() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    fetchUsers()
  }, 300)
}

function handlePageSizeChange() {
  currentPage.value = 1
  fetchUsers()
}

// Fetch users
async function fetchUsers() {
  loading.value = true
  try {
    const { sortBy, order } = getSortParams()
    const resp = await adminApi.listUsers(
      currentPage.value,
      pageSize.value,
      filterValue.value,
      searchQuery.value,
      sortBy,
      order,
    )
    users.value = resp.data.users || []
    totalUsers.value = resp.data.total
    if (resp.data.stats) {
      userStats.value = resp.data.stats
    }
  } catch {
    message.error('Failed to load users')
  } finally {
    loading.value = false
  }
}

// Fetch plans
async function fetchPlans() {
  try {
    const resp = await adminApi.listPlans()
    plans.value = resp.data.plans || []
    planOptions.value = plans.value.map((p) => ({ label: p.name, value: p.id }))
  } catch {
    // Plans might not be available
  }
}

// Table columns
const columns: DataTableColumns<AdminUser> = [
  { type: 'selection' },
  {
    title: 'ID',
    key: 'id',
    width: 60,
  },
  {
    title: 'Email / Phone',
    key: 'email',
    width: 200,
    ellipsis: { tooltip: true },
    render(row) {
      return row.email || row.phone || '-'
    },
  },
  {
    title: 'Name',
    key: 'display_name',
    width: 150,
    ellipsis: { tooltip: true },
  },
  {
    title: 'Status',
    key: 'is_active',
    width: 90,
    render(row) {
      return h(
        NTag,
        { type: row.is_active ? 'success' : 'error', size: 'small', bordered: false },
        { default: () => (row.is_active ? 'Active' : 'Blocked') },
      )
    },
  },
  {
    title: 'Admin',
    key: 'is_admin',
    width: 80,
    render(row) {
      if (!row.is_admin) return '-'
      return h(
        NTag,
        { type: 'info', size: 'small', bordered: false },
        { default: () => 'Admin' },
      )
    },
  },
  {
    title: 'Plan',
    key: 'plan',
    width: 100,
    render(row) {
      const planName = row.plan?.name || `#${row.plan_id}`
      return h(
        NTag,
        { size: 'small', bordered: false },
        { default: () => planName },
      )
    },
  },
  {
    title: 'Created',
    key: 'created_at',
    width: 120,
    render(row) {
      return format(new Date(row.created_at), 'MMM d, yyyy')
    },
  },
  {
    title: 'Last Login',
    key: 'last_login_at',
    width: 120,
    render(row) {
      if (!row.last_login_at) return '-'
      return format(new Date(row.last_login_at), 'MMM d, yyyy')
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 60,
    fixed: 'right',
    render(row) {
      const options = [
        {
          label: row.is_active ? 'Block' : 'Unblock',
          key: row.is_active ? 'block' : 'unblock',
        },
        {
          label: row.is_admin ? 'Remove Admin' : 'Make Admin',
          key: 'toggle_admin',
        },
        { label: 'Reset Password', key: 'reset_password' },
        { type: 'divider' as const, key: 'd1' },
        { label: 'Delete', key: 'delete' },
      ]

      return h(
        NDropdown,
        {
          options,
          trigger: 'click',
          onSelect: (key: string) => handleRowAction(key, row),
        },
        {
          default: () =>
            h(
              NButton,
              { text: true, size: 'small' },
              { default: () => h(NIcon, { component: EllipsisVertical }) },
            ),
        },
      )
    },
  },
]

function rowProps(row: AdminUser) {
  return {
    style: 'cursor: pointer',
    onClick: (e: MouseEvent) => {
      // Don't navigate if clicking checkbox or action button
      const target = e.target as HTMLElement
      if (target.closest('.n-checkbox') || target.closest('.n-dropdown') || target.closest('.n-button')) {
        return
      }
      router.push(`/users/${row.id}`)
    },
  }
}

// Row actions
async function handleRowAction(key: string, row: AdminUser) {
  if (key === 'block') {
    dialog.warning({
      title: 'Block User',
      content: `Block ${row.display_name || row.phone}?`,
      positiveText: 'Block',
      negativeText: 'Cancel',
      onPositiveClick: async () => {
        try {
          await adminApi.updateUser(row.id, { is_active: false })
          message.success('User blocked')
          fetchUsers()
        } catch {
          message.error('Failed to block user')
        }
      },
    })
  } else if (key === 'unblock') {
    try {
      await adminApi.updateUser(row.id, { is_active: true })
      message.success('User unblocked')
      fetchUsers()
    } catch {
      message.error('Failed to unblock user')
    }
  } else if (key === 'toggle_admin') {
    try {
      await adminApi.updateUser(row.id, { is_admin: !row.is_admin })
      message.success(row.is_admin ? 'Admin removed' : 'Admin granted')
      fetchUsers()
    } catch {
      message.error('Failed to update admin status')
    }
  } else if (key === 'reset_password') {
    router.push(`/users/${row.id}`)
  } else if (key === 'delete') {
    dialog.error({
      title: 'Delete User',
      content: `Permanently delete ${row.display_name || row.phone}? This cannot be undone.`,
      positiveText: 'Delete',
      negativeText: 'Cancel',
      onPositiveClick: async () => {
        try {
          await adminApi.deleteUser(row.id)
          message.success('User deleted')
          fetchUsers()
        } catch {
          message.error('Failed to delete user')
        }
      },
    })
  }
}

// Bulk operations
function handleBulk(action: string) {
  const count = checkedRowKeys.value.length
  const actionLabel = action === 'delete' ? 'delete' : action

  dialog.warning({
    title: `Bulk ${actionLabel}`,
    content: `Apply "${actionLabel}" to ${count} users?`,
    positiveText: 'Confirm',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        const ids = checkedRowKeys.value.map(Number)
        const resp = await adminApi.bulkUsers(action, ids)
        const result = resp.data
        message.success(
          `${result.success_count} succeeded, ${result.error_count} failed`,
        )
        checkedRowKeys.value = []
        fetchUsers()
      } catch {
        message.error(`Bulk ${actionLabel} failed`)
      }
    },
  })
}

async function handleChangePlan() {
  if (selectedPlanId.value === null) return
  try {
    const ids = checkedRowKeys.value.map(Number)
    const resp = await adminApi.bulkUsers('change_plan', ids, selectedPlanId.value)
    const result = resp.data
    message.success(
      `${result.success_count} succeeded, ${result.error_count} failed`,
    )
    checkedRowKeys.value = []
    showChangePlanDialog.value = false
    selectedPlanId.value = null
    fetchUsers()
  } catch {
    message.error('Failed to change plan')
  }
}

onMounted(() => {
  fetchUsers()
  fetchPlans()
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>
