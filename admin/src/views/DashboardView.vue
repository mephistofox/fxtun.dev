<template>
  <n-space vertical :size="24">
    <!-- SSE Connection indicator -->
    <n-space align="center" :size="8">
      <div
        :style="{
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: sseConnected ? '#63e2b7' : '#e88080',
        }"
      />
      <n-text depth="3" style="font-size: 12px">
        {{ sseConnected ? 'Live updates active' : 'Reconnecting...' }}
      </n-text>
    </n-space>

    <!-- Metric cards -->
    <n-grid :cols="6" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="6 s:3 m:2 l:1" v-for="card in metricCards" :key="card.label">
        <n-card size="small">
          <n-statistic :label="card.label" :value="card.value" tabular-nums>
            <template #prefix>
              <n-icon :component="card.icon" :size="20" />
            </template>
          </n-statistic>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Charts -->
    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="2 m:1">
        <n-card title="Registrations" size="small">
          <template #header-extra>
            <n-radio-group
              v-model:value="registrationsPeriod"
              size="small"
              @update:value="fetchRegistrationsChart"
            >
              <n-radio-button value="24h">24h</n-radio-button>
              <n-radio-button value="7d">7d</n-radio-button>
              <n-radio-button value="30d">30d</n-radio-button>
            </n-radio-group>
          </template>
          <n-spin :show="registrationsLoading">
            <v-chart
              :option="registrationsChartOption"
              :style="{ height: '300px' }"
              autoresize
            />
          </n-spin>
        </n-card>
      </n-gi>
      <n-gi span="2 m:1">
        <n-card title="Revenue" size="small">
          <template #header-extra>
            <n-radio-group
              v-model:value="revenuePeriod"
              size="small"
              @update:value="fetchRevenueChart"
            >
              <n-radio-button value="24h">24h</n-radio-button>
              <n-radio-button value="7d">7d</n-radio-button>
              <n-radio-button value="30d">30d</n-radio-button>
            </n-radio-group>
          </template>
          <n-spin :show="revenueLoading">
            <v-chart
              :option="revenueChartOption"
              :style="{ height: '300px' }"
              autoresize
            />
          </n-spin>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Bottom section: Recent Activity + Problem Nodes -->
    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <n-gi span="3 m:2">
        <n-card title="Recent Activity" size="small">
          <template #header-extra>
            <n-button text type="primary" @click="$router.push('/audit')">
              View all
            </n-button>
          </template>
          <n-data-table
            :columns="auditColumns"
            :data="recentAuditLogs"
            :loading="auditLoading"
            :bordered="false"
            size="small"
            :row-props="auditRowProps"
          />
        </n-card>
      </n-gi>
      <n-gi span="3 m:1">
        <n-card title="Problem Nodes" size="small">
          <n-empty
            v-if="problemNodes.length === 0 && !nodesLoading"
            description="All nodes healthy"
            style="padding: 24px 0"
          />
          <n-spin :show="nodesLoading">
            <n-list v-if="problemNodes.length > 0" :show-divider="true">
              <n-list-item v-for="node in problemNodes" :key="node.id">
                <n-thing :title="node.name" :description="node.region">
                  <template #header-extra>
                    <n-tag type="error" size="small">Stale</n-tag>
                  </template>
                  <n-text depth="3" style="font-size: 12px">
                    Last heartbeat: {{ formatHeartbeat(node.last_heartbeat_at) }}
                  </n-text>
                </n-thing>
              </n-list-item>
            </n-list>
          </n-spin>
        </n-card>
      </n-gi>
    </n-grid>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { useRouter } from 'vue-router'
import {
  NSpace,
  NGrid,
  NGi,
  NCard,
  NStatistic,
  NIcon,
  NText,
  NDataTable,
  NButton,
  NRadioGroup,
  NRadioButton,
  NSpin,
  NTag,
  NList,
  NListItem,
  NThing,
  NEmpty,
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import {
  PeopleOutline,
  GitNetworkOutline,
  GlobeOutline,
  SwapHorizontalOutline,
  CloudUploadOutline,
  PersonOutline,
} from '@vicons/ionicons5'
import { format, formatDistanceToNow, differenceInMinutes } from 'date-fns'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
} from 'echarts/components'
import type { ComposeOption } from 'echarts/core'
import type { LineSeriesOption, BarSeriesOption } from 'echarts/charts'
import type {
  GridComponentOption,
  TooltipComponentOption,
} from 'echarts/components'

import { adminApi } from '@/api/client'
import { useAdminSSE } from '@/composables/useSSE'
import type {
  AdminStats,
  AuditLog,
  EdgeNode,
  ChartDataPoint,
} from '@/api/types'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

type ECOption = ComposeOption<
  LineSeriesOption | BarSeriesOption | GridComponentOption | TooltipComponentOption
>

const router = useRouter()
const message = useMessage()

// SSE
const { stats: sseStats, connected: sseConnected, connect: sseConnect } = useAdminSSE()

// Stats
const statsData = ref<AdminStats>({
  active_clients: 0,
  active_tunnels: 0,
  http_tunnels: 0,
  tcp_tunnels: 0,
  udp_tunnels: 0,
  total_users: 0,
})

const currentStats = computed(() => sseStats.value ?? statsData.value)

const metricCards = computed(() => [
  {
    label: 'Active Clients',
    value: currentStats.value.active_clients,
    icon: PeopleOutline,
  },
  {
    label: 'Active Tunnels',
    value: currentStats.value.active_tunnels,
    icon: GitNetworkOutline,
  },
  {
    label: 'HTTP Tunnels',
    value: currentStats.value.http_tunnels,
    icon: GlobeOutline,
  },
  {
    label: 'TCP Tunnels',
    value: currentStats.value.tcp_tunnels,
    icon: SwapHorizontalOutline,
  },
  {
    label: 'UDP Tunnels',
    value: currentStats.value.udp_tunnels,
    icon: CloudUploadOutline,
  },
  {
    label: 'Total Users',
    value: currentStats.value.total_users,
    icon: PersonOutline,
  },
])

// Charts
const registrationsPeriod = ref('30d')
const revenuePeriod = ref('30d')
const registrationsData = ref<ChartDataPoint[]>([])
const revenueData = ref<ChartDataPoint[]>([])
const registrationsLoading = ref(false)
const revenueLoading = ref(false)

const chartAxisStyle = {
  axisLine: { lineStyle: { color: '#555' } },
  axisLabel: { color: '#999' },
  splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } },
}

const registrationsChartOption = computed<ECOption>(() => ({
  backgroundColor: 'transparent',
  tooltip: { trigger: 'axis' },
  grid: { left: 40, right: 16, top: 16, bottom: 32 },
  xAxis: {
    type: 'category',
    data: registrationsData.value.map((p) => formatChartDate(p.date)),
    ...chartAxisStyle,
  },
  yAxis: {
    type: 'value',
    minInterval: 1,
    ...chartAxisStyle,
  },
  series: [
    {
      type: 'line',
      data: registrationsData.value.map((p) => p.value),
      smooth: true,
      areaStyle: { opacity: 0.15 },
      lineStyle: { color: '#63e2b7' },
      itemStyle: { color: '#63e2b7' },
    },
  ],
}))

const revenueChartOption = computed<ECOption>(() => ({
  backgroundColor: 'transparent',
  tooltip: { trigger: 'axis' },
  grid: { left: 50, right: 16, top: 16, bottom: 32 },
  xAxis: {
    type: 'category',
    data: revenueData.value.map((p) => formatChartDate(p.date)),
    ...chartAxisStyle,
  },
  yAxis: {
    type: 'value',
    ...chartAxisStyle,
  },
  series: [
    {
      type: 'bar',
      data: revenueData.value.map((p) => p.value),
      itemStyle: { color: '#70c0e8', borderRadius: [4, 4, 0, 0] },
    },
  ],
}))

function formatChartDate(dateStr: string): string {
  try {
    const d = new Date(dateStr)
    return format(d, 'MMM d')
  } catch {
    return dateStr
  }
}

async function fetchRegistrationsChart() {
  registrationsLoading.value = true
  try {
    const resp = await adminApi.getChartData('registrations', registrationsPeriod.value)
    registrationsData.value = resp.data.points || []
  } catch {
    message.error('Failed to load registrations chart')
  } finally {
    registrationsLoading.value = false
  }
}

async function fetchRevenueChart() {
  revenueLoading.value = true
  try {
    const resp = await adminApi.getChartData('payments', revenuePeriod.value)
    revenueData.value = resp.data.points || []
  } catch {
    message.error('Failed to load revenue chart')
  } finally {
    revenueLoading.value = false
  }
}

// Audit
const recentAuditLogs = ref<AuditLog[]>([])
const auditLoading = ref(false)

const auditColumns: DataTableColumns<AuditLog> = [
  {
    title: 'Time',
    key: 'created_at',
    width: 160,
    render(row) {
      return format(new Date(row.created_at), 'MMM d, HH:mm:ss')
    },
  },
  {
    title: 'User',
    key: 'user_phone',
    width: 140,
    render(row) {
      return row.user_phone || '-'
    },
  },
  {
    title: 'Action',
    key: 'action',
    ellipsis: { tooltip: true },
  },
  {
    title: 'IP',
    key: 'ip_address',
    width: 130,
  },
]

function auditRowProps(row: AuditLog) {
  return {
    style: 'cursor: pointer',
    onClick: () => {
      router.push('/audit')
    },
  }
}

async function fetchAuditLogs() {
  auditLoading.value = true
  try {
    const resp = await adminApi.listAuditLogs(1, 10)
    recentAuditLogs.value = resp.data.logs || []
  } catch {
    message.error('Failed to load audit logs')
  } finally {
    auditLoading.value = false
  }
}

// Nodes
const allNodes = ref<EdgeNode[]>([])
const nodesLoading = ref(false)

const problemNodes = computed(() => {
  const now = new Date()
  return allNodes.value.filter((node) => {
    if (!node.last_heartbeat_at) return true
    return differenceInMinutes(now, new Date(node.last_heartbeat_at)) > 5
  })
})

function formatHeartbeat(dt: string | undefined): string {
  if (!dt) return 'Never'
  return formatDistanceToNow(new Date(dt), { addSuffix: true })
}

async function fetchNodes() {
  nodesLoading.value = true
  try {
    const resp = await adminApi.listNodes()
    allNodes.value = resp.data.nodes || []
  } catch {
    message.error('Failed to load nodes')
  } finally {
    nodesLoading.value = false
  }
}

// Initial fetch
async function fetchStats() {
  try {
    const resp = await adminApi.getStats()
    statsData.value = resp.data
  } catch {
    message.error('Failed to load stats')
  }
}

onMounted(() => {
  fetchStats()
  fetchRegistrationsChart()
  fetchRevenueChart()
  fetchAuditLogs()
  fetchNodes()
  sseConnect()
})
</script>
