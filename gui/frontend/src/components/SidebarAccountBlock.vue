<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Tooltip } from '@/components/ui'
import { Crown, ChevronUp, ArrowUpRight } from 'lucide-vue-next'
import { GetAccountInfo, GetUpgradeURL, GetManageURL } from '@/wailsjs/wailsjs/go/gui/AccountService'
import { BrowserOpenURL } from '@/wailsjs/wailsjs/runtime/runtime'

interface AccountInfo {
  display_name: string
  email: string
  phone: string
  avatar_url: string
  plan_name: string
  plan_slug: string
  max_tunnels: number
  max_domains: number
  max_custom_domains: number
  max_tokens: number
  inspector_enabled: boolean
  tunnel_count: number
  domain_count: number
  token_count: number
}

defineProps<{ collapsed: boolean }>()

const { t } = useI18n()

const account = ref<AccountInfo | null>(null)
const upgradeURL = ref('')
const manageURL = ref('')
const popoverOpen = ref(false)
const popoverRef = ref<HTMLElement | null>(null)

const initials = computed(() => {
  if (!account.value) return 'U'
  const name = account.value.display_name || account.value.email || account.value.phone
  if (!name) return 'U'
  return name.charAt(0).toUpperCase()
})

const displayName = computed(() => {
  if (!account.value) return ''
  return account.value.display_name || account.value.email || account.value.phone || 'User'
})

const isFree = computed(() => account.value?.plan_slug === 'free')

function usagePercent(used: number, max: number): number {
  if (max <= 0) return 0
  return Math.min(100, Math.round((used / max) * 100))
}

function barColor(percent: number): string {
  if (percent >= 100) return 'bg-red-500'
  if (percent >= 80) return 'bg-orange-500'
  if (percent >= 50) return 'bg-yellow-500'
  return 'bg-primary'
}

const usageItems = computed(() => {
  if (!account.value) return []
  const items = [
    {
      label: t('account.tunnels'),
      used: account.value.tunnel_count,
      max: account.value.max_tunnels,
    },
    {
      label: t('account.domains'),
      used: account.value.domain_count,
      max: account.value.max_domains,
    },
    {
      label: t('account.tokens'),
      used: account.value.token_count,
      max: account.value.max_tokens,
    },
  ]
  return items.map(item => ({
    ...item,
    percent: usagePercent(item.used, item.max),
    color: barColor(usagePercent(item.used, item.max)),
  }))
})

function togglePopover() {
  popoverOpen.value = !popoverOpen.value
}

function handleClickOutside(e: MouseEvent) {
  if (popoverRef.value && !popoverRef.value.contains(e.target as Node)) {
    popoverOpen.value = false
  }
}

async function openUpgrade() {
  if (upgradeURL.value) {
    BrowserOpenURL(upgradeURL.value)
  }
  popoverOpen.value = false
}

async function openManage() {
  if (manageURL.value) {
    BrowserOpenURL(manageURL.value)
  }
  popoverOpen.value = false
}

onMounted(async () => {
  try {
    const [info, upgrade, manage] = await Promise.all([
      GetAccountInfo(),
      GetUpgradeURL(),
      GetManageURL(),
    ])
    account.value = info as AccountInfo
    upgradeURL.value = upgrade
    manageURL.value = manage
  } catch {
    // silently fail — account info is not critical
  }
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div v-if="account" ref="popoverRef" class="relative">
    <!-- Popover (positioned above) -->
    <Transition name="popover">
      <div
        v-if="popoverOpen"
        class="absolute bottom-full left-0 right-0 mb-2 rounded-xl border border-border/50 bg-card/95 backdrop-blur-xl shadow-xl z-50 overflow-hidden"
        :class="collapsed ? 'left-0 w-64' : ''"
      >
        <!-- Header -->
        <div class="px-4 pt-4 pb-3 border-b border-border/30">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-accent text-sm font-bold text-primary-foreground">
              {{ initials }}
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold truncate">{{ displayName }}</p>
              <p v-if="account.email && account.email !== displayName" class="text-[11px] text-muted-foreground truncate">
                {{ account.email }}
              </p>
            </div>
          </div>
        </div>

        <!-- Plan info -->
        <div class="px-4 py-3 space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Crown v-if="!isFree" class="h-3.5 w-3.5 text-amber-400" />
              <span class="text-xs font-medium text-muted-foreground">{{ t('account.plan') }}</span>
            </div>
            <span
              :class="[
                'inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold',
                isFree
                  ? 'bg-muted/50 text-muted-foreground'
                  : 'bg-amber-500/10 text-amber-400'
              ]"
            >
              {{ account.plan_name }}
            </span>
          </div>

          <!-- Usage bars -->
          <div class="space-y-2.5">
            <div v-for="item in usageItems" :key="item.label" class="space-y-1">
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-muted-foreground">{{ item.label }}</span>
                <span class="font-medium tabular-nums">{{ item.used }}/{{ item.max }}</span>
              </div>
              <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted/30">
                <div
                  :class="['h-full rounded-full transition-all duration-300', item.color]"
                  :style="{ width: item.percent + '%' }"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Action button -->
        <div class="px-4 pb-3">
          <button
            v-if="isFree"
            @click="openUpgrade"
            class="flex w-full items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-primary to-accent px-3 py-2 text-xs font-semibold text-primary-foreground transition-all hover:opacity-90"
          >
            <ArrowUpRight class="h-3.5 w-3.5" />
            {{ t('account.upgrade') }}
          </button>
          <button
            v-else
            @click="openManage"
            class="flex w-full items-center justify-center gap-2 rounded-lg border border-border/50 bg-muted/30 px-3 py-2 text-xs font-medium text-foreground transition-all hover:bg-muted/50"
          >
            <ArrowUpRight class="h-3.5 w-3.5" />
            {{ t('account.manage') }}
          </button>
        </div>
      </div>
    </Transition>

    <!-- Trigger: collapsed -->
    <Tooltip v-if="collapsed" :content="account.plan_name" :delay-duration="0" side="right">
      <button
        @click.stop="togglePopover"
        class="flex w-full items-center justify-center rounded-xl px-3 py-2.5 transition-all hover:bg-muted/50"
      >
        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-accent text-xs font-bold text-primary-foreground">
          {{ initials }}
        </div>
      </button>
    </Tooltip>

    <!-- Trigger: expanded -->
    <button
      v-else
      @click.stop="togglePopover"
      class="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-all hover:bg-muted/50"
    >
      <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-accent text-xs font-bold text-primary-foreground">
        {{ initials }}
      </div>
      <div class="flex-1 min-w-0 text-left">
        <p class="text-xs font-medium truncate">{{ displayName }}</p>
        <p
          :class="[
            'text-[10px] truncate',
            isFree ? 'text-muted-foreground' : 'text-amber-400'
          ]"
        >
          {{ account.plan_name }}
        </p>
      </div>
      <ChevronUp
        :class="[
          'h-4 w-4 text-muted-foreground transition-transform duration-200',
          popoverOpen && 'rotate-180'
        ]"
      />
    </button>
  </div>
</template>

<style scoped>
.popover-enter-active,
.popover-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.popover-enter-from,
.popover-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>
