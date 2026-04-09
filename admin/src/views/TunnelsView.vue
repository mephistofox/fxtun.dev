<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <h1 class="text-2xl font-display font-bold">Тоннели</h1>
        <div class="flex items-center gap-2">
          <Badge variant="outline">Всего: {{ tunnels.length }}</Badge>
          <Badge variant="success">HTTP: {{ httpCount }}</Badge>
          <Badge variant="info">TCP: {{ tcpCount }}</Badge>
          <Badge variant="accent">UDP: {{ udpCount }}</Badge>
        </div>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <Input
        v-model="search"
        placeholder="Поиск по URL, субдомену, пользователю..."
        class="w-80"
      />
      <Select
        v-model="typeFilter"
        :options="typeOptions"
        placeholder="Тип"
        class="w-40"
      />
      <Button
        :variant="liveMode ? 'default' : 'outline'"
        size="sm"
        @click="toggleLive"
      >
        <span
          v-if="liveMode"
          class="relative flex h-2 w-2"
        >
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary-foreground opacity-75" />
          <span class="relative inline-flex h-2 w-2 rounded-full bg-primary-foreground" />
        </span>
        <span v-else class="h-2 w-2 rounded-full bg-muted-foreground" />
        Live
      </Button>
      <span v-if="lastUpdated" class="text-xs text-muted-foreground">
        Обновлено {{ lastUpdatedText }}
      </span>
    </div>

    <!-- Bulk actions -->
    <div v-if="selectedKeys.length > 0" class="flex items-center gap-3">
      <Button variant="destructive" size="sm" @click="showBulkConfirm = true">
        Закрыть {{ selectedKeys.length }} тоннелей
      </Button>
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredTunnels"
      :loading="loading"
      selectable
      :selected-keys="selectedKeys"
      row-key="id"
      empty-text="Нет активных тоннелей"
      @update:selected-keys="selectedKeys = $event"
    >
      <template #type="{ value }">
        <Badge :variant="tunnelTypeBadge(value)">{{ value.toUpperCase() }}</Badge>
      </template>

      <template #url="{ row }">
        <span class="text-sm">{{ row.url || row.subdomain || '-' }}</span>
      </template>

      <template #user_phone="{ value }">
        <span class="text-sm">{{ value }}</span>
      </template>

      <template #local_port="{ value }">
        <span class="font-mono text-sm">{{ value }}</span>
      </template>

      <template #client_id="{ value }">
        <span class="font-mono text-xs text-muted-foreground">{{ value ? value.substring(0, 8) : '-' }}</span>
      </template>

      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
      </template>

      <template #actions="{ row }">
        <Dropdown :items="getRowActions()" @select="(key) => handleAction(key, row)">
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Bulk close confirm -->
    <ConfirmDialog
      v-model:show="showBulkConfirm"
      title="Закрыть тоннели"
      :message="`Вы уверены, что хотите закрыть ${selectedKeys.length} тоннелей? Это действие необратимо.`"
      confirm-text="Закрыть"
      variant="destructive"
      @confirm="bulkClose"
    />

    <!-- Single close confirm -->
    <ConfirmDialog
      v-model:show="showCloseConfirm"
      title="Закрыть тоннель"
      :message="`Вы уверены, что хотите закрыть тоннель ${closingTunnel?.id || ''}?`"
      confirm-text="Закрыть"
      variant="destructive"
      @confirm="closeSingle"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { adminApi } from '@/api/client'
import type { AdminTunnel } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format, formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import { MoreHorizontal } from 'lucide-vue-next'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const tunnels = ref<AdminTunnel[]>([])
const loading = ref(false)
const search = ref('')
const typeFilter = ref<string | number | null>('all')
const selectedKeys = ref<(string | number)[]>([])
const liveMode = ref(false)
const lastUpdated = ref<Date | null>(null)
let liveInterval: ReturnType<typeof setInterval> | null = null

const showBulkConfirm = ref(false)
const showCloseConfirm = ref(false)
const closingTunnel = ref<AdminTunnel | null>(null)

const typeOptions = [
  { value: 'all', label: 'Все' },
  { value: 'http', label: 'HTTP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
]

const columns: Column[] = [
  { key: 'type', title: 'Тип', width: '80px' },
  { key: 'url', title: 'URL / Субдомен' },
  { key: 'user_phone', title: 'Пользователь' },
  { key: 'local_port', title: 'Локальный порт', width: '120px' },
  { key: 'client_id', title: 'Нода', width: '100px' },
  { key: 'created_at', title: 'Создан', width: '160px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

const httpCount = computed(() => tunnels.value.filter(t => t.type === 'http').length)
const tcpCount = computed(() => tunnels.value.filter(t => t.type === 'tcp').length)
const udpCount = computed(() => tunnels.value.filter(t => t.type === 'udp').length)

const filteredTunnels = computed(() => {
  let result = tunnels.value
  if (typeFilter.value && typeFilter.value !== 'all') {
    result = result.filter(t => t.type === typeFilter.value)
  }
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(t =>
      (t.url?.toLowerCase().includes(q)) ||
      (t.subdomain?.toLowerCase().includes(q)) ||
      t.user_phone.toLowerCase().includes(q) ||
      t.id.toLowerCase().includes(q)
    )
  }
  return result
})

const lastUpdatedText = computed(() => {
  if (!lastUpdated.value) return ''
  return formatDistanceToNow(lastUpdated.value, { locale: ru, addSuffix: true })
})

function tunnelTypeBadge(type: string): 'success' | 'info' | 'accent' {
  if (type === 'http') return 'success'
  if (type === 'tcp') return 'info'
  return 'accent'
}

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

function getRowActions() {
  return [{ key: 'close', label: 'Закрыть', destructive: true }]
}

function handleAction(key: string, row: AdminTunnel) {
  if (key === 'close') {
    closingTunnel.value = row
    showCloseConfirm.value = true
  }
}

async function fetchTunnels() {
  loading.value = true
  try {
    const params: { type?: string } = {}
    const { data } = await adminApi.listTunnels(params)
    tunnels.value = data.tunnels || []
    lastUpdated.value = new Date()
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function closeSingle() {
  if (!closingTunnel.value) return
  try {
    await adminApi.closeTunnel(closingTunnel.value.id)
    await fetchTunnels()
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    closingTunnel.value = null
  }
}

async function bulkClose() {
  try {
    await adminApi.bulkCloseTunnels(selectedKeys.value as string[])
    selectedKeys.value = []
    await fetchTunnels()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

function toggleLive() {
  liveMode.value = !liveMode.value
  if (liveMode.value) {
    liveInterval = setInterval(fetchTunnels, 5000)
  } else if (liveInterval) {
    clearInterval(liveInterval)
    liveInterval = null
  }
}

onMounted(() => {
  fetchTunnels()
})

onUnmounted(() => {
  if (liveInterval) {
    clearInterval(liveInterval)
  }
})
</script>
