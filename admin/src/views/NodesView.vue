<template>
  <div class="p-6 space-y-6">
    <!-- Pending alert -->
    <div
      v-if="pendingCount > 0"
      class="flex items-center gap-3 rounded-xl border border-[hsl(var(--warning)/0.3)] bg-[hsl(var(--warning)/0.08)] px-4 py-3"
    >
      <AlertTriangle class="h-5 w-5 text-[hsl(var(--warning))] flex-shrink-0" />
      <span class="text-sm text-foreground">
        {{ pendingCount }} нод ожидают подтверждения
      </span>
    </div>

    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Ноды</h1>
    </div>

    <!-- Filter buttons -->
    <div class="flex items-center gap-2">
      <Button
        v-for="f in filters"
        :key="f.key"
        :variant="statusFilter === f.key ? 'default' : 'outline'"
        size="sm"
        @click="statusFilter = f.key"
      >
        {{ f.label }}
        <Badge v-if="f.count > 0" variant="outline" size="sm" class="ml-1.5">
          {{ f.count }}
        </Badge>
      </Button>
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredNodes"
      :loading="loading"
      row-key="id"
      empty-text="Нет нод"
    >
      <template #name="{ value }">
        <span class="font-medium">{{ value }}</span>
      </template>

      <template #region="{ value }">
        <span class="text-sm">{{ value || '-' }}</span>
      </template>

      <template #public_addr="{ value }">
        <span class="font-mono text-sm">{{ value }}</span>
      </template>

      <template #status="{ value }">
        <Badge :variant="statusBadge(value)">{{ statusLabel(value) }}</Badge>
      </template>

      <template #version="{ value }">
        <span class="font-mono text-sm">{{ value || '-' }}</span>
      </template>

      <template #last_heartbeat_at="{ value }">
        <div v-if="value" class="flex items-center gap-2">
          <span
            class="h-2 w-2 rounded-full"
            :class="heartbeatColor(value)"
          />
          <span class="text-sm text-muted-foreground">
            {{ formatDistanceToNow(new Date(value), { locale: ru, addSuffix: true }) }}
          </span>
        </div>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>

      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
      </template>

      <template #actions="{ row }">
        <Dropdown :items="getRowActions(row)" @select="(key) => handleAction(key, row)">
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Approve confirm -->
    <ConfirmDialog
      v-model:show="showApproveConfirm"
      title="Одобрить ноду"
      :message="`Одобрить ноду «${actionNode?.name || ''}»?`"
      confirm-text="Одобрить"
      @confirm="approveNode"
    />

    <!-- Disable confirm -->
    <ConfirmDialog
      v-model:show="showDisableConfirm"
      title="Отключить ноду"
      :message="`Отключить ноду «${actionNode?.name || ''}»? Все тоннели на этой ноде будут закрыты.`"
      confirm-text="Отключить"
      variant="destructive"
      @confirm="disableNode"
    />

    <!-- Delete confirm -->
    <ConfirmDialog
      v-model:show="showDeleteConfirm"
      title="Удалить ноду"
      :message="`Удалить ноду «${actionNode?.name || ''}»? Это действие необратимо.`"
      confirm-text="Удалить"
      variant="destructive"
      @confirm="deleteNode"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { EdgeNode } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format, formatDistanceToNow, differenceInSeconds } from 'date-fns'
import { ru } from 'date-fns/locale'
import { MoreHorizontal, AlertTriangle } from 'lucide-vue-next'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const nodes = ref<EdgeNode[]>([])
const loading = ref(false)
const statusFilter = ref('all')
const actionNode = ref<EdgeNode | null>(null)
const showApproveConfirm = ref(false)
const showDisableConfirm = ref(false)
const showDeleteConfirm = ref(false)

const columns: Column[] = [
  { key: 'name', title: 'Имя' },
  { key: 'region', title: 'Регион', width: '120px' },
  { key: 'public_addr', title: 'Адрес' },
  { key: 'status', title: 'Статус', width: '120px' },
  { key: 'version', title: 'Версия', width: '100px' },
  { key: 'last_heartbeat_at', title: 'Heartbeat', width: '180px' },
  { key: 'created_at', title: 'Создана', width: '160px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

const pendingCount = computed(() => nodes.value.filter(n => n.status === 'pending').length)
const activeCount = computed(() => nodes.value.filter(n => n.status === 'active').length)
const disabledCount = computed(() => nodes.value.filter(n => n.status === 'disabled').length)

const filters = computed(() => [
  { key: 'all', label: 'Все', count: nodes.value.length },
  { key: 'active', label: 'Активные', count: activeCount.value },
  { key: 'pending', label: 'Ожидают', count: pendingCount.value },
  { key: 'disabled', label: 'Отключены', count: disabledCount.value },
])

const filteredNodes = computed(() => {
  if (statusFilter.value === 'all') return nodes.value
  return nodes.value.filter(n => n.status === statusFilter.value)
})

function statusBadge(status: string): 'success' | 'warning' | 'destructive' {
  if (status === 'active') return 'success'
  if (status === 'pending') return 'warning'
  return 'destructive'
}

function statusLabel(status: string): string {
  if (status === 'active') return 'Активна'
  if (status === 'pending') return 'Ожидает'
  return 'Отключена'
}

function heartbeatColor(dateStr: string): string {
  const seconds = differenceInSeconds(new Date(), new Date(dateStr))
  if (seconds < 60) return 'bg-type-http'
  if (seconds < 300) return 'bg-[hsl(var(--warning))]'
  return 'bg-destructive'
}

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

function getRowActions(row: EdgeNode) {
  const actions = []
  if (row.status === 'pending') {
    actions.push({ key: 'approve', label: 'Одобрить' })
  }
  if (row.status === 'active') {
    actions.push({ key: 'disable', label: 'Отключить', destructive: true })
  }
  actions.push({ key: 'delete', label: 'Удалить', destructive: true })
  return actions
}

function handleAction(key: string, row: EdgeNode) {
  actionNode.value = row
  if (key === 'approve') showApproveConfirm.value = true
  if (key === 'disable') showDisableConfirm.value = true
  if (key === 'delete') showDeleteConfirm.value = true
}

async function fetchNodes() {
  loading.value = true
  try {
    const { data } = await adminApi.listNodes()
    nodes.value = data.nodes || []
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function approveNode() {
  if (!actionNode.value) return
  try {
    await adminApi.approveNode(actionNode.value.id)
    await fetchNodes()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

async function disableNode() {
  if (!actionNode.value) return
  try {
    await adminApi.disableNode(actionNode.value.id)
    await fetchNodes()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

async function deleteNode() {
  if (!actionNode.value) return
  try {
    await adminApi.deleteNode(actionNode.value.id)
    await fetchNodes()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

onMounted(() => {
  fetchNodes()
})
</script>
