<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold text-foreground">Панель управления</h1>
      <div class="flex items-center gap-2">
        <span
          class="inline-flex items-center gap-1.5 text-xs"
          :class="sseConnected ? 'text-type-http' : 'text-muted-foreground'"
        >
          <span class="status-dot" :class="sseConnected ? 'status-connected' : 'status-disconnected'">
            <span v-if="sseConnected" class="status-dot-ping status-connected" />
          </span>
          {{ sseConnected ? 'Подключено' : 'Отключено' }}
        </span>
      </div>
    </div>

    <!-- Stats grid -->
    <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
      <Stat
        label="Активные клиенты"
        :value="currentStats.active_clients"
        :icon="MonitorSmartphone"
      />
      <Stat
        label="Активные тоннели"
        :value="currentStats.active_tunnels"
        :icon="Network"
      />
      <Stat
        label="HTTP тоннели"
        :value="currentStats.http_tunnels"
        :icon="Globe"
        class="border-type-http/20"
      />
      <Stat
        label="TCP тоннели"
        :value="currentStats.tcp_tunnels"
        :icon="Cable"
        class="border-type-tcp/20"
      />
      <Stat
        label="UDP тоннели"
        :value="currentStats.udp_tunnels"
        :icon="Radio"
        class="border-type-udp/20"
      />
      <Stat
        label="Всего пользователей"
        :value="currentStats.total_users"
        :icon="Users"
      />
    </div>

    <!-- Charts section -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Registrations chart -->
      <Card variant="glass" class="p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display font-semibold text-foreground">Регистрации</h2>
          <div class="flex gap-1">
            <button
              v-for="p in chartPeriods"
              :key="p.value"
              type="button"
              class="px-3 py-1 text-xs font-medium rounded-md transition-colors"
              :class="
                registrationPeriod === p.value
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-surface-elevated'
              "
              @click="loadChart('registrations', p.value)"
            >
              {{ p.label }}
            </button>
          </div>
        </div>
        <BarChart
          :data="registrationData"
          :loading="registrationLoading"
          color="hsl(75 100% 50% / 0.8)"
        />
      </Card>

      <!-- Revenue chart -->
      <Card variant="glass" class="p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="font-display font-semibold text-foreground">Доход</h2>
          <div class="flex gap-1">
            <button
              v-for="p in chartPeriods"
              :key="p.value"
              type="button"
              class="px-3 py-1 text-xs font-medium rounded-md transition-colors"
              :class="
                revenuePeriod === p.value
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-surface-elevated'
              "
              @click="loadChart('revenue', p.value)"
            >
              {{ p.label }}
            </button>
          </div>
        </div>
        <BarChart
          :data="revenueData"
          :loading="revenueLoading"
          color="hsl(280 100% 65% / 0.8)"
          :format-value="(v: number) => `${v} ₽`"
        />
      </Card>
    </div>

    <!-- Bottom section -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Recent events -->
      <Card variant="glass" class="p-6">
        <h2 class="font-display font-semibold text-foreground mb-4">Последние события</h2>
        <div v-if="auditLoading" class="flex items-center justify-center py-8">
          <svg
            class="h-6 w-6 animate-spin text-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
        <div v-else-if="auditLogs.length === 0" class="text-center py-8 text-muted-foreground text-sm">
          Нет событий
        </div>
        <div v-else class="space-y-0 -mx-2">
          <div
            v-for="log in auditLogs"
            :key="log.id"
            class="flex items-start gap-3 px-2 py-2.5 rounded-lg hover:bg-surface-elevated/50 transition-colors"
          >
            <div class="flex-shrink-0 mt-1">
              <FileText class="h-4 w-4 text-muted-foreground" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm text-foreground truncate">
                <span class="font-medium">{{ log.action }}</span>
                <span v-if="log.user_phone" class="text-muted-foreground"> — {{ log.user_phone }}</span>
              </p>
              <p class="text-xs text-muted-foreground mt-0.5">
                {{ formatRelative(log.created_at) }}
                <span v-if="log.ip_address" class="font-mono"> &middot; {{ log.ip_address }}</span>
              </p>
            </div>
          </div>
        </div>
      </Card>

      <!-- Problematic nodes -->
      <Card variant="glass" class="p-6">
        <h2 class="font-display font-semibold text-foreground mb-4">Проблемные ноды</h2>
        <div v-if="nodesLoading" class="flex items-center justify-center py-8">
          <svg
            class="h-6 w-6 animate-spin text-primary"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
        <div v-else-if="staleNodes.length === 0" class="text-center py-8 text-muted-foreground text-sm">
          <CheckCircle class="h-8 w-8 mx-auto mb-2 text-type-http opacity-50" />
          Все ноды работают нормально
        </div>
        <div v-else class="space-y-0 -mx-2">
          <div
            v-for="node in staleNodes"
            :key="node.id"
            class="flex items-center gap-3 px-2 py-2.5 rounded-lg hover:bg-surface-elevated/50 transition-colors"
          >
            <div class="flex-shrink-0">
              <AlertTriangle class="h-4 w-4 text-destructive" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-foreground truncate">{{ node.name }}</p>
              <p class="text-xs text-muted-foreground">
                {{ node.region }} &middot;
                <span class="font-mono">{{ node.public_addr }}</span>
              </p>
            </div>
            <div class="text-right flex-shrink-0">
              <Badge variant="destructive" size="sm">
                {{ formatRelative(node.last_heartbeat_at) }}
              </Badge>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import {
  MonitorSmartphone,
  Network,
  Globe,
  Cable,
  Radio,
  Users,
  FileText,
  AlertTriangle,
  CheckCircle,
} from 'lucide-vue-next'
import { adminApi } from '@/api/client'
import { useAdminSSE } from '@/composables/useSSE'
import type { AdminStats, AuditLog, EdgeNode, ChartDataPoint } from '@/api/types'
import Card from '@/components/ui/Card.vue'
import Stat from '@/components/ui/Stat.vue'
import Badge from '@/components/ui/Badge.vue'
import BarChart from '@/components/BarChart.vue'

// --- SSE stats ---
const { stats: sseStats, connected: sseConnected, connect: sseConnect, disconnect: sseDisconnect } = useAdminSSE()

const fallbackStats = ref<AdminStats>({
  active_clients: 0,
  active_tunnels: 0,
  http_tunnels: 0,
  tcp_tunnels: 0,
  udp_tunnels: 0,
  total_users: 0,
})

const currentStats = computed(() => sseStats.value ?? fallbackStats.value)

// --- Chart data ---
const chartPeriods = [
  { label: '7д', value: '7d' },
  { label: '30д', value: '30d' },
]

const registrationPeriod = ref('30d')
const registrationData = ref<ChartDataPoint[]>([])
const registrationLoading = ref(false)

const revenuePeriod = ref('30d')
const revenueData = ref<ChartDataPoint[]>([])
const revenueLoading = ref(false)

async function loadChart(metric: string, period: string) {
  if (metric === 'registrations') {
    registrationPeriod.value = period
    registrationLoading.value = true
    try {
      const { data } = await adminApi.getChartData('registrations', period)
      registrationData.value = data.points ?? []
    } catch {
      registrationData.value = []
    } finally {
      registrationLoading.value = false
    }
  } else {
    revenuePeriod.value = period
    revenueLoading.value = true
    try {
      const { data } = await adminApi.getChartData('payments', period)
      revenueData.value = data.points ?? []
    } catch {
      revenueData.value = []
    } finally {
      revenueLoading.value = false
    }
  }
}

// --- Audit logs ---
const auditLogs = ref<AuditLog[]>([])
const auditLoading = ref(false)

async function loadAuditLogs() {
  auditLoading.value = true
  try {
    const { data } = await adminApi.listAuditLogs(1, 10)
    auditLogs.value = data.logs ?? []
  } catch {
    auditLogs.value = []
  } finally {
    auditLoading.value = false
  }
}

// --- Nodes ---
const allNodes = ref<EdgeNode[]>([])
const nodesLoading = ref(false)

const staleNodes = computed(() => {
  const fiveMinAgo = Date.now() - 5 * 60 * 1000
  return allNodes.value.filter((n) => {
    if (!n.last_heartbeat_at) return true
    return new Date(n.last_heartbeat_at).getTime() < fiveMinAgo
  })
})

async function loadNodes() {
  nodesLoading.value = true
  try {
    const { data } = await adminApi.listNodes()
    allNodes.value = data.nodes ?? []
  } catch {
    allNodes.value = []
  } finally {
    nodesLoading.value = false
  }
}

// --- Helpers ---
function formatRelative(date?: string): string {
  if (!date) return 'Нет данных'
  return formatDistanceToNow(new Date(date), { addSuffix: true, locale: ru })
}

// --- Init ---
async function loadInitialStats() {
  try {
    const { data } = await adminApi.getStats()
    fallbackStats.value = data
  } catch {
    // SSE will take over
  }
}

onMounted(() => {
  sseConnect()
  loadInitialStats()
  loadChart('registrations', '30d')
  loadChart('revenue', '30d')
  loadAuditLogs()
  loadNodes()
})

onUnmounted(() => {
  sseDisconnect()
})
</script>
