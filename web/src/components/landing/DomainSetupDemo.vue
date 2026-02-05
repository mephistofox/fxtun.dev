<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// Phases: 'typing' → 'dns' → 'verifying' → 'issuing' → 'done' → pause → restart
type Phase = 'typing' | 'dns' | 'verifying' | 'issuing' | 'done'

const phase = ref<Phase>('typing')
const typedDomain = ref('')
const verifyProgress = ref(0)
const targetDomain = 'app.myproject.com'

let timer: ReturnType<typeof setTimeout> | null = null
let typingIdx = 0

function clearTimers() {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
}

function startTyping() {
  phase.value = 'typing'
  typedDomain.value = ''
  typingIdx = 0
  verifyProgress.value = 0
  typeNextChar()
}

function typeNextChar() {
  if (typingIdx < targetDomain.length) {
    typedDomain.value += targetDomain[typingIdx]
    typingIdx++
    const delay = 40 + Math.random() * 60
    timer = setTimeout(typeNextChar, delay)
  } else {
    // Done typing, show DNS step
    timer = setTimeout(() => {
      phase.value = 'dns'
      timer = setTimeout(startVerifying, 1800)
    }, 600)
  }
}

function startVerifying() {
  phase.value = 'verifying'
  verifyProgress.value = 0
  animateVerify()
}

function animateVerify() {
  if (verifyProgress.value < 100) {
    verifyProgress.value += 2 + Math.random() * 4
    if (verifyProgress.value > 100) verifyProgress.value = 100
    timer = setTimeout(animateVerify, 40)
  } else {
    timer = setTimeout(() => {
      phase.value = 'issuing'
      timer = setTimeout(() => {
        phase.value = 'done'
        // Restart cycle after 3 seconds
        timer = setTimeout(startTyping, 3000)
      }, 1200)
    }, 300)
  }
}

const dnsRecords = computed(() => [
  { type: 'CNAME', name: typedDomain.value || '...', value: 'tunnel.fxtun.dev' },
])

onMounted(() => {
  startTyping()
})

onUnmounted(() => {
  clearTimers()
})
</script>

<template>
  <div class="rounded-xl border border-border bg-[hsl(220,20%,4%)] overflow-hidden font-mono text-xs select-none h-[360px] flex flex-col shadow-2xl">
    <!-- Toolbar -->
    <div class="flex items-center gap-2.5 px-3 py-2 border-b border-border/70 bg-[hsl(220,20%,7%)]">
      <svg class="w-4 h-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
      <span class="text-foreground/70 text-[11px] font-medium">{{ t('landing.advanced.customDomains.demo.title') }}</span>
    </div>

    <!-- Content -->
    <div class="flex-1 p-4 space-y-4 overflow-hidden">
      <!-- Domain input -->
      <div>
        <label class="text-[10px] text-muted-foreground/70 uppercase tracking-wider font-semibold mb-1.5 block">
          {{ t('landing.advanced.customDomains.demo.yourDomain') }}
        </label>
        <div class="flex items-center rounded-lg bg-white/[0.03] border border-border/50 px-3 py-2">
          <span class="text-foreground/80 text-[12px]">{{ typedDomain }}</span>
          <span
            v-if="phase === 'typing'"
            class="w-[2px] h-[14px] bg-primary ml-0.5 animate-blink"
          />
        </div>
      </div>

      <!-- Step 1: DNS Records -->
      <div
        class="transition-all duration-300"
        :class="phase === 'typing' ? 'opacity-30 translate-y-1' : 'opacity-100 translate-y-0'"
      >
        <div class="flex items-center gap-2 mb-2">
          <!-- Step indicator -->
          <div
            class="w-5 h-5 rounded-full flex items-center justify-center text-[9px] font-bold transition-colors duration-300"
            :class="phase !== 'typing' && phase !== 'dns' ? 'bg-emerald-500/20 text-emerald-400' : 'border border-primary/40 text-primary'"
          >
            <svg v-if="phase !== 'typing' && phase !== 'dns'" class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
            <span v-else>1</span>
          </div>
          <span class="text-[10px] font-medium text-foreground/80">{{ t('landing.advanced.customDomains.demo.step1') }}</span>
        </div>

        <!-- DNS record table -->
        <div class="rounded-lg bg-white/[0.02] border border-border/30 overflow-hidden ml-7">
          <div class="grid grid-cols-[48px_1fr_1fr] gap-px text-[9px]">
            <div class="px-2 py-1 text-muted-foreground/50 uppercase bg-white/[0.02]">{{ t('landing.advanced.customDomains.demo.type') }}</div>
            <div class="px-2 py-1 text-muted-foreground/50 uppercase bg-white/[0.02]">{{ t('landing.advanced.customDomains.demo.name') }}</div>
            <div class="px-2 py-1 text-muted-foreground/50 uppercase bg-white/[0.02]">{{ t('landing.advanced.customDomains.demo.value') }}</div>
            <div v-for="rec in dnsRecords" :key="rec.type" class="contents">
              <div class="px-2 py-1.5 text-primary/80 font-semibold text-[10px]">{{ rec.type }}</div>
              <div class="px-2 py-1.5 text-foreground/60 truncate text-[10px]">{{ rec.name }}</div>
              <div class="px-2 py-1.5 text-foreground/60 truncate text-[10px]">{{ rec.value }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Step 2: Verifying -->
      <div
        class="transition-all duration-300"
        :class="phase === 'typing' || phase === 'dns' ? 'opacity-30 translate-y-1' : 'opacity-100 translate-y-0'"
      >
        <div class="flex items-center gap-2 mb-2">
          <div
            class="w-5 h-5 rounded-full flex items-center justify-center text-[9px] font-bold transition-colors duration-300"
            :class="phase === 'issuing' || phase === 'done' ? 'bg-emerald-500/20 text-emerald-400' : phase === 'verifying' ? 'border border-amber-500/40 text-amber-400' : 'border border-border/50 text-muted-foreground/50'"
          >
            <svg v-if="phase === 'issuing' || phase === 'done'" class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
            <svg
              v-else-if="phase === 'verifying'"
              class="w-3 h-3 animate-spin"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
            >
              <path d="M12 2a10 10 0 0 1 10 10" />
            </svg>
            <span v-else>2</span>
          </div>
          <span
            class="text-[10px] font-medium transition-colors"
            :class="phase === 'verifying' ? 'text-amber-400' : phase === 'issuing' || phase === 'done' ? 'text-emerald-400' : 'text-foreground/50'"
          >
            {{ phase === 'issuing' || phase === 'done' ? t('landing.advanced.customDomains.demo.verified') : t('landing.advanced.customDomains.demo.step2') }}
          </span>
        </div>

        <!-- Progress bar -->
        <div v-if="phase === 'verifying'" class="ml-7 h-1.5 rounded-full bg-white/[0.05] overflow-hidden">
          <div
            class="h-full rounded-full bg-gradient-to-r from-amber-500 to-amber-400 transition-all duration-100"
            :style="{ width: `${verifyProgress}%` }"
          />
        </div>
      </div>

      <!-- Step 3: Certificate -->
      <div
        class="transition-all duration-300"
        :class="phase === 'done' || phase === 'issuing' ? 'opacity-100 translate-y-0' : 'opacity-30 translate-y-1'"
      >
        <div class="flex items-center gap-2 mb-2">
          <div
            class="w-5 h-5 rounded-full flex items-center justify-center text-[9px] font-bold transition-colors duration-300"
            :class="phase === 'done' ? 'bg-emerald-500/20 text-emerald-400' : phase === 'issuing' ? 'border border-amber-500/40 text-amber-400' : 'border border-border/50 text-muted-foreground/50'"
          >
            <svg v-if="phase === 'done'" class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
            <svg
              v-else-if="phase === 'issuing'"
              class="w-3 h-3 animate-spin"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
            >
              <path d="M12 2a10 10 0 0 1 10 10" />
            </svg>
            <span v-else>3</span>
          </div>
          <span
            class="text-[10px] font-medium transition-colors"
            :class="phase === 'done' ? 'text-emerald-400' : 'text-foreground/50'"
          >
            {{ t('landing.advanced.customDomains.demo.step3') }}
          </span>
        </div>

        <!-- Success result -->
        <Transition name="fade-up">
          <div v-if="phase === 'done'" class="ml-7 space-y-2">
            <div class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20">
              <svg class="w-3 h-3 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
              <span class="text-[10px] text-emerald-400 font-medium">{{ t('landing.advanced.customDomains.demo.https') }}</span>
            </div>
            <div class="rounded-lg bg-white/[0.02] border border-emerald-500/20 px-3 py-2">
              <span class="text-[10px] text-muted-foreground/60">{{ t('landing.advanced.customDomains.demo.liveAt') }}</span>
              <span class="text-[11px] text-emerald-400 font-medium ml-1">https://{{ typedDomain }}</span>
              <span class="text-[10px] text-muted-foreground/40 ml-2">→ localhost:3000</span>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}

.animate-blink {
  animation: blink 0.8s infinite;
}

.fade-up-enter-active {
  transition: all 0.4s ease-out;
}
.fade-up-leave-active {
  transition: all 0.2s ease-in;
}
.fade-up-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.fade-up-leave-to {
  opacity: 0;
}
</style>
