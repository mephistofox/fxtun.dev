<template>
  <div class="p-6 space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-display font-bold">TLS сертификаты</h1>
        <p class="text-sm text-muted-foreground mt-1">
          Автообновление через certbot.timer (2× в день). Уведомления в Telegram при &lt; 14 дней.
        </p>
      </div>
      <button
        @click="load"
        :disabled="loading"
        class="px-4 py-2 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:opacity-90 disabled:opacity-50 inline-flex items-center gap-2"
      >
        <RefreshCw :class="['h-4 w-4', loading && 'animate-spin']" />
        Обновить
      </button>
    </div>

    <div v-if="error" class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-200">
      {{ error }}
    </div>

    <DataTable
      :columns="columns"
      :data="certs"
      :loading="loading"
      row-key="hostname"
      empty-text="Нет сертификатов"
    >
      <template #hostname="{ value, row }">
        <div class="flex items-center gap-2">
          <span class="font-mono text-sm">{{ value }}</span>
          <Badge v-if="row.wildcard" variant="info">wildcard</Badge>
        </div>
      </template>

      <template #issuer="{ value }">
        <span class="text-sm text-muted-foreground">{{ value || '—' }}</span>
      </template>

      <template #days_left="{ value, row }">
        <span :class="['font-mono text-sm font-medium', daysColor(row.status)]">
          {{ row.status === 'error' ? '—' : `${value} дн.` }}
        </span>
      </template>

      <template #not_after="{ value }">
        <span class="text-sm">{{ formatDate(value) }}</span>
      </template>

      <template #source="{ value }">
        <Badge :variant="value === 'tls' ? 'default' : 'outline'">
          {{ value === 'tls' ? 'TLS-probe' : 'БД' }}
        </Badge>
      </template>

      <template #status="{ value, row }">
        <Badge :variant="statusVariant(value)">
          {{ statusLabel(value) }}
        </Badge>
        <span v-if="row.error" class="block text-xs text-red-400 mt-0.5 font-mono">
          {{ row.error }}
        </span>
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import DataTable, { type Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import { adminApi } from '@/api/client'
import type { CertificateInfo } from '@/api/types'

const certs = ref<CertificateInfo[]>([])
const loading = ref(false)
const error = ref('')

const columns: Column[] = [
  { key: 'hostname', title: 'Hostname' },
  { key: 'issuer', title: 'Выдан кем', width: '180px' },
  { key: 'days_left', title: 'Осталось', width: '110px' },
  { key: 'not_after', title: 'Истекает', width: '180px' },
  { key: 'source', title: 'Источник', width: '110px' },
  { key: 'status', title: 'Статус', width: '160px' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await adminApi.listCertificates()
    certs.value = data.certificates || []
  } catch (e: any) {
    error.value = e?.response?.data?.error || e?.message || 'Не удалось загрузить'
  } finally {
    loading.value = false
  }
}

function formatDate(s: string): string {
  if (!s) return '—'
  return new Date(s).toLocaleString('ru-RU', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

function statusVariant(s: string): 'success' | 'warning' | 'destructive' | 'default' {
  switch (s) {
    case 'ok': return 'success'
    case 'expiring': return 'warning'
    case 'critical':
    case 'expired':
    case 'error': return 'destructive'
    default: return 'default'
  }
}

function statusLabel(s: string): string {
  switch (s) {
    case 'ok': return 'OK'
    case 'expiring': return 'Истекает (<30 дн.)'
    case 'critical': return 'Критично (<7 дн.)'
    case 'expired': return 'Истёк'
    case 'error': return 'Ошибка'
    default: return s
  }
}

function daysColor(status: string): string {
  switch (status) {
    case 'critical':
    case 'expired': return 'text-red-400'
    case 'expiring': return 'text-amber-400'
    case 'error': return 'text-muted-foreground'
    default: return 'text-foreground'
  }
}

onMounted(load)
</script>
