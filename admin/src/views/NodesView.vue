<template>
  <n-space vertical :size="16">
    <!-- Pending alert -->
    <n-alert v-if="pendingCount > 0" type="warning" :bordered="true">
      {{ pendingCount }} node{{ pendingCount > 1 ? 's' : '' }} pending approval
    </n-alert>

    <!-- Toolbar -->
    <n-space align="center" :size="12">
      <n-radio-group v-model:value="statusFilter">
        <n-radio-button value="">All ({{ statusCounts.all }})</n-radio-button>
        <n-radio-button value="active">Active ({{ statusCounts.active }})</n-radio-button>
        <n-radio-button value="pending">Pending ({{ statusCounts.pending }})</n-radio-button>
        <n-radio-button value="disabled">Disabled ({{ statusCounts.disabled }})</n-radio-button>
      </n-radio-group>
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="nodes"
      :loading="loading"
      :row-key="(row: EdgeNode) => row.id"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { getErrorMessage } from '@/utils/error'
import { useMessage, useDialog, NTag, NButton, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format, formatDistanceToNow, differenceInSeconds } from 'date-fns'
import { adminApi } from '@/api/client'
import type { EdgeNode } from '@/api/types'

const message = useMessage()
const dialog = useDialog()

const allNodes = ref<EdgeNode[]>([])
const loading = ref(false)
const statusFilter = ref('')

const nodes = computed(() => {
  if (!statusFilter.value) return allNodes.value
  return allNodes.value.filter(n => n.status === statusFilter.value)
})

const pendingCount = computed(() =>
  allNodes.value.filter(n => n.status === 'pending').length,
)

const statusCounts = computed(() => ({
  all: allNodes.value.length,
  active: allNodes.value.filter(n => n.status === 'active').length,
  pending: allNodes.value.filter(n => n.status === 'pending').length,
  disabled: allNodes.value.filter(n => n.status === 'disabled').length,
}))

function heartbeatHealth(heartbeat: string | undefined): { color: string; label: string } {
  if (!heartbeat) return { color: '#d03050', label: 'Never' }
  const seconds = differenceInSeconds(new Date(), new Date(heartbeat))
  if (seconds < 60) return { color: '#18a058', label: formatDistanceToNow(new Date(heartbeat), { addSuffix: true }) }
  if (seconds < 300) return { color: '#f0a020', label: formatDistanceToNow(new Date(heartbeat), { addSuffix: true }) }
  return { color: '#d03050', label: formatDistanceToNow(new Date(heartbeat), { addSuffix: true }) }
}

const statusTagType: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  active: 'success',
  pending: 'warning',
  disabled: 'error',
}

const columns: DataTableColumns<EdgeNode> = [
  { title: 'Name', key: 'name', width: 150 },
  { title: 'Region', key: 'region', width: 100 },
  {
    title: 'Public Address',
    key: 'public_addr',
    width: 180,
    ellipsis: { tooltip: true },
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
  { title: 'Version', key: 'version', width: 100 },
  {
    title: 'Last Heartbeat',
    key: 'last_heartbeat_at',
    width: 180,
    render(row) {
      const health = heartbeatHealth(row.last_heartbeat_at)
      return h('span', { style: { display: 'flex', alignItems: 'center', gap: '6px' } }, [
        h('span', {
          style: {
            width: '8px',
            height: '8px',
            borderRadius: '50%',
            backgroundColor: health.color,
            display: 'inline-block',
            flexShrink: 0,
          },
        }),
        health.label,
      ])
    },
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
    width: 180,
    render(row) {
      const buttons: ReturnType<typeof h>[] = []

      if (row.status === 'pending') {
        buttons.push(
          h(NButton, { size: 'small', type: 'success', quaternary: true, onClick: () => handleApprove(row) }, { default: () => 'Approve' }),
        )
      }

      if (row.status === 'active') {
        buttons.push(
          h(NButton, { size: 'small', type: 'warning', quaternary: true, onClick: () => handleDisable(row) }, { default: () => 'Disable' }),
        )
      }

      buttons.push(
        h(NButton, { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) }, { default: () => 'Delete' }),
      )

      return h(NSpace, { size: 4 }, { default: () => buttons })
    },
  },
]

async function fetchNodes() {
  loading.value = true
  try {
    const { data } = await adminApi.listNodes()
    allNodes.value = data.nodes || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err, 'Failed to load nodes'))
  } finally {
    loading.value = false
  }
}

function handleApprove(node: EdgeNode) {
  dialog.info({
    title: 'Approve Node',
    content: `Approve node "${node.name}"?`,
    positiveText: 'Approve',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.approveNode(node.id)
        message.success('Node approved')
        await fetchNodes()
      } catch (err: unknown) {
        message.error(getErrorMessage(err, 'Failed to approve node'))
      }
    },
  })
}

function handleDisable(node: EdgeNode) {
  dialog.warning({
    title: 'Disable Node',
    content: `Disable node "${node.name}"?`,
    positiveText: 'Disable',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.disableNode(node.id)
        message.success('Node disabled')
        await fetchNodes()
      } catch (err: unknown) {
        message.error(getErrorMessage(err, 'Failed to disable node'))
      }
    },
  })
}

function handleDelete(node: EdgeNode) {
  dialog.error({
    title: 'Delete Node',
    content: `Permanently delete node "${node.name}"? This cannot be undone.`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.deleteNode(node.id)
        message.success('Node deleted')
        await fetchNodes()
      } catch (err: unknown) {
        message.error(getErrorMessage(err, 'Failed to delete node'))
      }
    },
  })
}

onMounted(() => {
  fetchNodes()
})
</script>
