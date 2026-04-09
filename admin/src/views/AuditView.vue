<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space align="center" justify="space-between">
      <n-space align="center" :size="12">
        <n-input
          v-model:value="searchText"
          placeholder="Search action, phone, IP..."
          clearable
          style="width: 280px"
        />
        <n-select
          v-model:value="categoryFilter"
          :options="categoryOptions"
          style="width: 160px"
          @update:value="handleFilterChange"
        />
      </n-space>
      <n-pagination
        v-model:page="currentPage"
        :page-count="totalPages"
        :page-slot="7"
        @update:page="fetchLogs"
      />
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="filteredLogs"
      :loading="loading"
      :row-key="(row: AuditLog) => row.id"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useMessage, NCode, NButton, NEllipsis } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format } from 'date-fns'
import { adminApi } from '@/api/client'
import type { AuditLog } from '@/api/types'

const message = useMessage()

const logs = ref<AuditLog[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 30
const searchText = ref('')
const categoryFilter = ref('')

const categoryOptions = [
  { label: 'All Categories', value: '' },
  { label: 'Auth', value: 'auth' },
  { label: 'Tokens', value: 'token' },
  { label: 'Domains', value: 'domain' },
  { label: 'Users', value: 'user' },
  { label: 'Other', value: 'other' },
]

const categoryPrefixes: Record<string, string[]> = {
  auth: ['login', 'logout', 'register', 'auth', 'totp', 'password', 'refresh'],
  token: ['token', 'api_token'],
  domain: ['domain', 'subdomain'],
  user: ['user', 'admin', 'block', 'unblock', 'plan'],
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const filteredLogs = computed(() => {
  let result = logs.value

  if (categoryFilter.value) {
    const prefixes = categoryPrefixes[categoryFilter.value]
    if (prefixes) {
      result = result.filter(log =>
        prefixes.some(prefix => log.action.toLowerCase().startsWith(prefix)),
      )
    } else {
      // "other" category — exclude all known prefixes
      const allKnown = Object.values(categoryPrefixes).flat()
      result = result.filter(log =>
        !allKnown.some(prefix => log.action.toLowerCase().startsWith(prefix)),
      )
    }
  }

  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(log =>
      log.action.toLowerCase().includes(q) ||
      log.user_phone?.toLowerCase().includes(q) ||
      log.ip_address?.toLowerCase().includes(q),
    )
  }

  return result
})

const columns: DataTableColumns<AuditLog> = [
  {
    title: 'Timestamp',
    key: 'created_at',
    width: 160,
    render(row) {
      return row.created_at ? format(new Date(row.created_at), 'yyyy-MM-dd HH:mm:ss') : '-'
    },
  },
  {
    title: 'User',
    key: 'user_phone',
    width: 140,
    render(row) {
      return row.user_phone || (row.user_id ? `User #${row.user_id}` : 'System')
    },
  },
  {
    title: 'IP Address',
    key: 'ip_address',
    width: 140,
  },
  {
    title: 'Action',
    key: 'action',
    width: 200,
  },
  {
    title: 'Details',
    key: 'details',
    render(row) {
      if (!row.details || Object.keys(row.details).length === 0) return '-'
      const jsonStr = JSON.stringify(row.details, null, 2)
      const truncated = jsonStr.length > 80 ? jsonStr.substring(0, 80) + '...' : jsonStr
      return h(
        'div',
        null,
        [
          h(NEllipsis, { style: 'max-width: 400px', tooltip: false }, {
            default: () => truncated,
          }),
          jsonStr.length > 80
            ? h(
                NButton,
                {
                  size: 'tiny',
                  text: true,
                  type: 'info',
                  style: 'margin-left: 8px',
                  onClick: () => {
                    expandedRow.value = expandedRow.value === row.id ? null : row.id
                  },
                },
                { default: () => expandedRow.value === row.id ? 'Collapse' : 'Expand' },
              )
            : null,
          expandedRow.value === row.id
            ? h(NCode, {
                code: jsonStr,
                language: 'json',
                style: 'margin-top: 8px; max-height: 300px; overflow: auto',
              })
            : null,
        ],
      )
    },
  },
]

const expandedRow = ref<number | null>(null)

function handleFilterChange() {
  currentPage.value = 1
  fetchLogs()
}

async function fetchLogs() {
  loading.value = true
  try {
    const { data } = await adminApi.listAuditLogs(currentPage.value, pageSize)
    logs.value = data.logs || []
    total.value = data.total || 0
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load audit logs')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLogs()
})
</script>
