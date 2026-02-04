<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type AdminSubscription, type AdminPayment, type Plan } from '@/api/client'

const { t } = useI18n()

const subscriptions = ref<AdminSubscription[]>([])
const payments = ref<AdminPayment[]>([])
const plans = ref<Plan[]>([])
const loading = ref(true)
const error = ref('')
const successMsg = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20

const activeTab = ref<'subscriptions' | 'payments'>('subscriptions')
const activeFilter = ref<'all' | 'active' | 'cancelled' | 'expired'>('active')
const paymentFilter = ref<'all' | 'success' | 'pending' | 'failed'>('all')

// Extend modal
const extendSubId = ref<number | null>(null)
const extendDays = ref(30)

const filteredSubscriptions = computed(() => {
  if (activeFilter.value === 'all') return subscriptions.value
  return subscriptions.value.filter(s => s.status === activeFilter.value)
})

const filteredPayments = computed(() => {
  if (paymentFilter.value === 'all') return payments.value
  return payments.value.filter(p => p.status === paymentFilter.value)
})

const filterTabs = computed(() => [
  { key: 'all' as const, label: t('admin.filterAll'), count: subscriptions.value.length },
  { key: 'active' as const, label: t('admin.subscriptions.active'), count: subscriptions.value.filter(s => s.status === 'active').length },
  { key: 'cancelled' as const, label: t('admin.subscriptions.cancelled'), count: subscriptions.value.filter(s => s.status === 'cancelled').length },
  { key: 'expired' as const, label: t('admin.subscriptions.expired'), count: subscriptions.value.filter(s => s.status === 'expired').length },
])

const paymentFilterTabs = computed(() => [
  { key: 'all' as const, label: t('admin.filterAll'), count: payments.value.length },
  { key: 'success' as const, label: t('admin.subscriptions.payment_success'), count: payments.value.filter(p => p.status === 'success').length },
  { key: 'pending' as const, label: t('admin.subscriptions.payment_pending'), count: payments.value.filter(p => p.status === 'pending').length },
  { key: 'failed' as const, label: t('admin.subscriptions.payment_failed'), count: payments.value.filter(p => p.status === 'failed').length },
])

const statusColors: Record<string, string> = {
  active: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  cancelled: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  expired: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  pending: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
}

const paymentStatusColors: Record<string, string> = {
  success: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  failed: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  pending: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
}

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [subsRes, plansRes] = await Promise.all([
      adminApi.listSubscriptions(page.value, limit),
      adminApi.listPlans()
    ])
    subscriptions.value = subsRes.data.subscriptions || []
    total.value = subsRes.data.total
    plans.value = plansRes.data.plans || []
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function loadPayments() {
  loading.value = true
  error.value = ''
  try {
    const res = await adminApi.listPayments(page.value, limit)
    payments.value = res.data.payments || []
    total.value = res.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

function switchTab(tab: 'subscriptions' | 'payments') {
  activeTab.value = tab
  page.value = 1
  if (tab === 'subscriptions') {
    loadData()
  } else {
    loadPayments()
  }
}

function showSuccess(msg: string) {
  successMsg.value = msg
  setTimeout(() => { successMsg.value = '' }, 3000)
}

async function cancelSubscription(id: number) {
  try {
    await adminApi.cancelSubscription(id)
    const sub = subscriptions.value.find(s => s.id === id)
    if (sub) {
      sub.status = 'cancelled'
      sub.recurring = false
    }
    showSuccess(t('admin.subscriptions.cancelledSuccess'))
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.subscriptions.cancelFailed')
  }
}

function openExtendModal(id: number) {
  extendSubId.value = id
  extendDays.value = 30
}

async function extendSubscription() {
  if (!extendSubId.value) return
  try {
    await adminApi.extendSubscription(extendSubId.value, extendDays.value)
    showSuccess(t('admin.subscriptions.extendedSuccess'))
    extendSubId.value = null
    loadData()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.subscriptions.extendFailed')
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('ru-RU')
}

function formatAmount(amount: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB' }).format(amount)
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold">{{ t('admin.subscriptions.title') }}</h1>
      </div>

      <!-- Success/Error messages -->
      <div v-if="successMsg" class="bg-green-900/30 border border-green-700 text-green-200 rounded-lg p-4">
        {{ successMsg }}
      </div>
      <div v-if="error" class="bg-destructive/10 text-destructive p-4 rounded-lg text-sm">
        {{ error }}
      </div>

      <!-- Tabs -->
      <div class="flex gap-2 border-b border-border pb-2">
        <button
          class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
          :class="activeTab === 'subscriptions' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
          @click="switchTab('subscriptions')"
        >
          {{ t('admin.subscriptions.subscriptions') }}
        </button>
        <button
          class="px-4 py-2 text-sm font-medium rounded-t-lg transition-colors"
          :class="activeTab === 'payments' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
          @click="switchTab('payments')"
        >
          {{ t('admin.subscriptions.payments') }}
        </button>
      </div>

      <!-- Subscriptions Tab -->
      <template v-if="activeTab === 'subscriptions'">
        <!-- Filter tabs -->
        <div class="flex gap-2 flex-wrap">
          <button
            v-for="tab in filterTabs"
            :key="tab.key"
            class="px-3 py-1.5 text-sm rounded-full transition-colors"
            :class="activeFilter === tab.key ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground'"
            @click="activeFilter = tab.key"
          >
            {{ tab.label }} ({{ tab.count }})
          </button>
        </div>

        <Card>
          <div v-if="loading" class="flex justify-center py-12">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>

          <div v-else-if="filteredSubscriptions.length === 0" class="text-center py-12 text-muted-foreground">
            {{ t('admin.subscriptions.noSubscriptions') }}
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full">
              <thead class="border-b border-border">
                <tr class="text-left text-sm text-muted-foreground">
                  <th class="p-4">{{ t('admin.subscriptions.user') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.plan') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.status') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.period') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.recurring') }}</th>
                  <th class="p-4">{{ t('common.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="sub in filteredSubscriptions" :key="sub.id" class="border-b border-border last:border-0 hover:bg-muted/50">
                  <td class="p-4">
                    <div>{{ sub.user_email || sub.user_phone }}</div>
                    <div v-if="sub.user_email && sub.user_phone" class="text-xs text-muted-foreground">{{ sub.user_phone }}</div>
                  </td>
                  <td class="p-4">
                    <span class="font-medium">{{ sub.plan?.name || '-' }}</span>
                    <div v-if="sub.next_plan" class="text-xs text-muted-foreground">
                      {{ t('admin.subscriptions.nextPlan') }}: {{ sub.next_plan.name }}
                    </div>
                  </td>
                  <td class="p-4">
                    <span class="px-2 py-1 text-xs font-medium rounded-full" :class="statusColors[sub.status]">
                      {{ t(`admin.subscriptions.${sub.status}`) }}
                    </span>
                  </td>
                  <td class="p-4 text-sm">
                    <div>{{ formatDate(sub.current_period_start) }}</div>
                    <div class="text-muted-foreground">{{ formatDate(sub.current_period_end) }}</div>
                  </td>
                  <td class="p-4">
                    <span :class="sub.recurring ? 'text-green-500' : 'text-muted-foreground'">
                      {{ sub.recurring ? t('common.yes') : t('common.no') }}
                    </span>
                  </td>
                  <td class="p-4">
                    <div class="flex gap-2">
                      <Button
                        v-if="sub.status === 'active'"
                        size="sm"
                        variant="ghost"
                        @click="openExtendModal(sub.id)"
                      >
                        {{ t('admin.subscriptions.extend') }}
                      </Button>
                      <Button
                        v-if="sub.status === 'active'"
                        size="sm"
                        variant="ghost"
                        class="text-destructive"
                        @click="cancelSubscription(sub.id)"
                      >
                        {{ t('admin.subscriptions.cancel') }}
                      </Button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </template>

      <!-- Payments Tab -->
      <template v-else>
        <!-- Payment filter tabs -->
        <div class="flex gap-2 flex-wrap">
          <button
            v-for="tab in paymentFilterTabs"
            :key="tab.key"
            class="px-3 py-1.5 text-sm rounded-full transition-colors"
            :class="paymentFilter === tab.key ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground'"
            @click="paymentFilter = tab.key"
          >
            {{ tab.label }} ({{ tab.count }})
          </button>
        </div>

        <Card>
          <div v-if="loading" class="flex justify-center py-12">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>

          <div v-else-if="filteredPayments.length === 0" class="text-center py-12 text-muted-foreground">
            {{ t('admin.subscriptions.noPayments') }}
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full">
              <thead class="border-b border-border">
                <tr class="text-left text-sm text-muted-foreground">
                  <th class="p-4">{{ t('admin.subscriptions.invoiceId') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.user') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.amount') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.status') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.recurring') }}</th>
                  <th class="p-4">{{ t('admin.subscriptions.date') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="payment in filteredPayments" :key="payment.id" class="border-b border-border last:border-0 hover:bg-muted/50">
                  <td class="p-4 font-mono text-sm">#{{ payment.invoice_id }}</td>
                  <td class="p-4">
                    <div>{{ payment.user_email || payment.user_phone }}</div>
                    <div v-if="payment.user_email && payment.user_phone" class="text-xs text-muted-foreground">{{ payment.user_phone }}</div>
                  </td>
                  <td class="p-4 font-medium">{{ formatAmount(payment.amount) }}</td>
                  <td class="p-4">
                    <span class="px-2 py-1 text-xs font-medium rounded-full" :class="paymentStatusColors[payment.status]">
                      {{ t(`admin.subscriptions.payment_${payment.status}`) }}
                    </span>
                  </td>
                  <td class="p-4">
                    <span :class="payment.is_recurring ? 'text-green-500' : 'text-muted-foreground'">
                      {{ payment.is_recurring ? t('common.yes') : t('common.no') }}
                    </span>
                  </td>
                  <td class="p-4 text-sm text-muted-foreground">{{ formatDate(payment.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </template>

      <!-- Extend Modal -->
      <div v-if="extendSubId" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="extendSubId = null">
        <Card class="w-full max-w-md p-6">
          <h3 class="text-lg font-bold mb-4">{{ t('admin.subscriptions.extendTitle') }}</h3>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">{{ t('admin.subscriptions.days') }}</label>
              <Input v-model.number="extendDays" type="number" min="1" max="365" />
            </div>
            <div class="flex gap-2 justify-end">
              <Button variant="ghost" @click="extendSubId = null">{{ t('common.cancel') }}</Button>
              <Button @click="extendSubscription">{{ t('admin.subscriptions.extend') }}</Button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  </Layout>
</template>
