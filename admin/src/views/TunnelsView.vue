<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space align="center" justify="space-between">
      <n-space align="center" :size="12">
        <n-input
          v-model:value="searchText"
          placeholder="Search by URL, subdomain, user..."
          clearable
          style="width: 280px"
        />
        <n-select
          v-model:value="typeFilter"
          :options="typeOptions"
          style="width: 140px"
        />
        <n-tag type="default">Total: {{ tunnels.length }}</n-tag>
        <n-tag type="info">HTTP: {{ stats.http }}</n-tag>
        <n-tag type="success">TCP: {{ stats.tcp }}</n-tag>
        <n-tag type="warning">UDP: {{ stats.udp }}</n-tag>
      </n-space>
      <n-space align="center" :size="12">
        <n-text v-if="lastUpdated" depth="3" style="font-size: 12px">
          Last updated: {{ lastUpdatedText }}
        </n-text>
        <n-switch v-model:value="liveMode">
          <template #checked>Live</template>
          <template #unchecked>Live</template>
        </n-switch>
        <n-button
          v-if="checkedRowKeys.length > 0"
          type="error"
          @click="handleBulkClose"
        >
          Close {{ checkedRowKeys.length }} tunnels
        </n-button>
      </n-space>
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="filteredTunnels"
      :loading="loading"
      :row-key="(row: AdminTunnel) => row.id"
      :checked-row-keys="checkedRowKeys"
      @update:checked-row-keys="handleCheck"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h, watch } from 'vue'
import { useMessage, useDialog, NTag, NButton } from 'naive-ui'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { format, formatDistanceToNow } from 'date-fns'
import { adminApi } from '@/api/client'
import type { AdminTunnel } from '@/api/types'
import { getErrorMessage } from '@/utils/error'

const message = useMessage()
const dialog = useDialog()

const tunnels = ref<AdminTunnel[]>([])
const loading = ref(false)
const searchText = ref('')
const typeFilter = ref<string | null>(null)
const liveMode = ref(false)
const checkedRowKeys = ref<DataTableRowKey[]>([])
const lastUpdated = ref<Date | null>(null)
let refreshInterval: ReturnType<typeof setInterval> | null = null

const typeOptions = [
  { label: 'All Types', value: '' },
  { label: 'HTTP', value: 'http' },
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]

const stats = computed(() => ({
  http: tunnels.value.filter(t => t.type === 'http').length,
  tcp: tunnels.value.filter(t => t.type === 'tcp').length,
  udp: tunnels.value.filter(t => t.type === 'udp').length,
}))

const lastUpdatedText = computed(() => {
  if (!lastUpdated.value) return ''
  return formatDistanceToNow(lastUpdated.value, { addSuffix: true })
})

const filteredTunnels = computed(() => {
  let result = tunnels.value
  if (typeFilter.value) {
    result = result.filter(t => t.type === typeFilter.value)
  }
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(t =>
      (t.url?.toLowerCase().includes(q)) ||
      (t.subdomain?.toLowerCase().includes(q)) ||
      (t.user_phone?.toLowerCase().includes(q)) ||
      (t.name?.toLowerCase().includes(q))
    )
  }
  return result
})

const typeTagMap: Record<string, 'info' | 'success' | 'warning'> = {
  http: 'info',
  tcp: 'success',
  udp: 'warning',
}

const columns: DataTableColumns<AdminTunnel> = [
  { type: 'selection' },
  {
    title: 'Type',
    key: 'type',
    width: 80,
    render(row) {
      return h(NTag, { type: typeTagMap[row.type] || 'default', size: 'small' }, {
        default: () => row.type.toUpperCase(),
      })
    },
  },
  {
    title: 'URL / Subdomain',
    key: 'url',
    ellipsis: { tooltip: true },
    render(row) {
      return row.url || row.subdomain || `port:${row.remote_port}` || '-'
    },
  },
  {
    title: 'User',
    key: 'user_phone',
    width: 150,
  },
  {
    title: 'Local Port',
    key: 'local_port',
    width: 100,
  },
  {
    title: 'Client ID',
    key: 'client_id',
    width: 120,
    ellipsis: { tooltip: true },
  },
  {
    title: 'Created At',
    key: 'created_at',
    width: 160,
    render(row) {
      return row.created_at ? format(new Date(row.created_at), 'yyyy-MM-dd HH:mm') : '-'
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 100,
    render(row) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          quaternary: true,
          onClick: () => handleCloseTunnel(row),
        },
        { default: () => 'Close' },
      )
    },
  },
]

function handleCheck(keys: DataTableRowKey[]) {
  checkedRowKeys.value = keys
}

async function fetchTunnels() {
  loading.value = true
  try {
    const { data } = await adminApi.listTunnels()
    tunnels.value = data.tunnels || []
    lastUpdated.value = new Date()
  } catch (err: unknown) {
    message.error(getErrorMessage(err, 'Failed to load tunnels'))
  } finally {
    loading.value = false
  }
}

function handleCloseTunnel(tunnel: AdminTunnel) {
  dialog.warning({
    title: 'Close Tunnel',
    content: `Are you sure you want to close tunnel "${tunnel.url || tunnel.subdomain || tunnel.id}"?`,
    positiveText: 'Close',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.closeTunnel(tunnel.id)
        message.success('Tunnel closed')
        await fetchTunnels()
      } catch (err: unknown) {
        message.error(getErrorMessage(err, 'Failed to close tunnel'))
      }
    },
  })
}

function handleBulkClose() {
  const ids = checkedRowKeys.value as string[]
  dialog.warning({
    title: 'Close Tunnels',
    content: `Are you sure you want to close ${ids.length} tunnels?`,
    positiveText: 'Close All',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        const { data } = await adminApi.bulkCloseTunnels(ids)
        message.success(`Closed ${data.success_count} tunnels`)
        if (data.error_count > 0) {
          message.warning(`${data.error_count} tunnels failed to close`)
        }
        checkedRowKeys.value = []
        await fetchTunnels()
      } catch (err: unknown) {
        message.error(getErrorMessage(err, 'Bulk close failed'))
      }
    },
  })
}

function startAutoRefresh() {
  stopAutoRefresh()
  refreshInterval = setInterval(fetchTunnels, 5000)
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

watch(liveMode, (val) => {
  if (val) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

onMounted(() => {
  fetchTunnels()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>
