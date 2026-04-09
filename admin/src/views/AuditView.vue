<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Журнал действий</h1>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <Input
        v-model="search"
        placeholder="Поиск по действию, телефону, IP..."
        class="w-80"
      />
      <Select
        v-model="categoryFilter"
        :options="categoryOptions"
        placeholder="Категория"
        class="w-48"
      />
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredLogs"
      :loading="loading"
      row-key="id"
      empty-text="Нет записей"
    >
      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground font-mono">
          {{ formatDateTime(value) }}
        </span>
      </template>

      <template #user_phone="{ value }">
        <span class="text-sm">{{ value || '-' }}</span>
      </template>

      <template #ip_address="{ value }">
        <span class="font-mono text-sm">{{ value }}</span>
      </template>

      <template #action="{ value }">
        <Badge variant="outline">{{ value }}</Badge>
      </template>

      <template #details="{ row }">
        <div v-if="row.details" class="max-w-xs">
          <button
            v-if="!expandedRows.has(row.id)"
            type="button"
            class="text-sm text-muted-foreground hover:text-foreground truncate block max-w-full text-left"
            @click="expandedRows.add(row.id)"
          >
            {{ truncateDetails(row.details) }}
          </button>
          <div v-else>
            <pre class="text-xs font-mono bg-background rounded-lg p-3 overflow-x-auto whitespace-pre-wrap break-all border border-border">{{ JSON.stringify(row.details, null, 2) }}</pre>
            <button
              type="button"
              class="text-xs text-primary hover:underline mt-1"
              @click="expandedRows.delete(row.id)"
            >
              Свернуть
            </button>
          </div>
        </div>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      v-if="total > pageSize"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="(p) => { page = p; fetchLogs() }"
      @update:page-size="(s) => { pageSize = s; page = 1; fetchLogs() }"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { AuditLog } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Pagination from '@/components/ui/Pagination.vue'

const logs = ref<AuditLog[]>([])
const loading = ref(false)
const search = ref('')
const categoryFilter = ref<string | number | null>('all')
const page = ref(1)
const pageSize = ref(30)
const total = ref(0)
const expandedRows = reactive(new Set<number>())

const categoryOptions = [
  { value: 'all', label: 'Все' },
  { value: 'auth', label: 'Авторизация' },
  { value: 'token', label: 'Токены' },
  { value: 'domain', label: 'Домены' },
  { value: 'user', label: 'Пользователи' },
  { value: 'other', label: 'Другое' },
]

const categoryPrefixes: Record<string, string[]> = {
  auth: ['login', 'logout', 'register', 'refresh', 'totp', 'password'],
  token: ['token', 'api_token'],
  domain: ['domain', 'subdomain', 'custom_domain'],
  user: ['user', 'admin_user', 'merge'],
}

const columns: Column[] = [
  { key: 'created_at', title: 'Время', width: '180px' },
  { key: 'user_phone', title: 'Пользователь', width: '160px' },
  { key: 'ip_address', title: 'IP адрес', width: '140px' },
  { key: 'action', title: 'Действие', width: '200px' },
  { key: 'details', title: 'Детали' },
]

const filteredLogs = computed(() => {
  let result = logs.value

  // Client-side category filtering
  if (categoryFilter.value && categoryFilter.value !== 'all') {
    const cat = String(categoryFilter.value)
    const prefixes = categoryPrefixes[cat]
    if (prefixes) {
      result = result.filter(log =>
        prefixes.some(p => log.action.toLowerCase().startsWith(p))
      )
    } else {
      // "other" - everything not matching known categories
      const allPrefixes = Object.values(categoryPrefixes).flat()
      result = result.filter(log =>
        !allPrefixes.some(p => log.action.toLowerCase().startsWith(p))
      )
    }
  }

  // Client-side search filtering
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(log =>
      log.action.toLowerCase().includes(q) ||
      (log.user_phone?.toLowerCase().includes(q)) ||
      log.ip_address.toLowerCase().includes(q)
    )
  }

  return result
})

function formatDateTime(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm:ss', { locale: ru })
}

function truncateDetails(details: Record<string, unknown>): string {
  const str = JSON.stringify(details)
  if (str.length > 60) return str.substring(0, 60) + '...'
  return str
}

async function fetchLogs() {
  loading.value = true
  try {
    const { data } = await adminApi.listAuditLogs(page.value, pageSize.value)
    logs.value = data.logs || []
    total.value = data.total
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLogs()
})
</script>
