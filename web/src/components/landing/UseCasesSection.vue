<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const copiedIndex = ref<number | null>(null)

const cases = [
  { key: 'webhooks', icon: 'webhook', color: 'type-http' },
  { key: 'demos', icon: 'share', color: 'primary' },
  { key: 'ssh', icon: 'terminal', color: 'type-tcp' },
  { key: 'gameServers', icon: 'gamepad', color: 'type-udp' },
  { key: 'api', icon: 'code', color: 'type-http' },
  { key: 'iot', icon: 'cpu', color: 'primary' },
]

function copyCommand(command: string, index: number) {
  navigator.clipboard.writeText(command)
  copiedIndex.value = index
  setTimeout(() => { copiedIndex.value = null }, 2000)
}

onMounted(() => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          isVisible.value = true
          observer.disconnect()
        }
      })
    },
    { threshold: 0.1 }
  )
  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }
})
</script>

<template>
  <section id="use-cases" ref="sectionRef" class="py-16 md:py-32 relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-gradient-to-b from-background via-surface/20 to-background" />

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-16">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.useCases.label') }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.useCases.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.useCases.subtitle') }}
        </p>
      </div>

      <!-- Use cases grid -->
      <div class="max-w-6xl mx-auto grid md:grid-cols-2 lg:grid-cols-3 gap-5">
        <div
          v-for="(uc, index) in cases"
          :key="uc.key"
          class="group relative rounded-2xl bg-surface/50 border border-border hover:border-primary/30 transition-all duration-300 hover:shadow-lg reveal"
          :class="[
            { 'visible': isVisible },
            `reveal-delay-${3 + index}`
          ]"
        >
          <div class="p-6 flex flex-col h-full">
            <!-- Icon + Title -->
            <div class="flex items-center gap-3 mb-3">
              <div
                class="w-10 h-10 rounded-xl flex items-center justify-center"
                :style="`background: hsl(var(--${uc.color}) / 0.1);`"
              >
                <!-- webhook -->
                <svg v-if="uc.icon === 'webhook'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244" />
                </svg>
                <!-- share -->
                <svg v-else-if="uc.icon === 'share'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M7.217 10.907a2.25 2.25 0 1 0 0 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186 9.566-5.314m-9.566 7.5 9.566 5.314m0 0a2.25 2.25 0 1 0 3.935 2.186 2.25 2.25 0 0 0-3.935-2.186Zm0-12.814a2.25 2.25 0 1 0 3.933-2.185 2.25 2.25 0 0 0-3.933 2.185Z" />
                </svg>
                <!-- terminal -->
                <svg v-else-if="uc.icon === 'terminal'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z" />
                </svg>
                <!-- gamepad -->
                <svg v-else-if="uc.icon === 'gamepad'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M14.25 6.087c0-.355.186-.676.401-.959.221-.29.349-.634.349-1.003 0-1.036-1.007-1.875-2.25-1.875s-2.25.84-2.25 1.875c0 .369.128.713.349 1.003.215.283.401.604.401.959v0a.64.64 0 0 1-.657.643 48.39 48.39 0 0 1-4.163-.3c.186 1.613.293 3.25.315 4.907a.656.656 0 0 1-.658.663v0c-.355 0-.676-.186-.959-.401a1.647 1.647 0 0 0-1.003-.349c-1.036 0-1.875 1.007-1.875 2.25s.84 2.25 1.875 2.25c.369 0 .713-.128 1.003-.349.283-.215.604-.401.959-.401v0c.31 0 .555.26.532.57a48.039 48.039 0 0 1-.642 5.056c1.518.19 3.058.309 4.616.354a.64.64 0 0 0 .657-.643v0c0-.355-.186-.676-.401-.959a1.647 1.647 0 0 1-.349-1.003c0-1.035 1.008-1.875 2.25-1.875 1.243 0 2.25.84 2.25 1.875 0 .369-.128.713-.349 1.003-.215.283-.4.604-.4.959v0c0 .333.277.599.61.58a48.1 48.1 0 0 0 5.427-.63 48.05 48.05 0 0 0 .582-4.717.532.532 0 0 0-.533-.57v0c-.355 0-.676.186-.959.401-.29.221-.634.349-1.003.349-1.035 0-1.875-1.007-1.875-2.25s.84-2.25 1.875-2.25c.37 0 .713.128 1.003.349.283.215.604.401.96.401v0a.656.656 0 0 0 .658-.663 48.422 48.422 0 0 0-.37-5.36c-1.886.342-3.81.574-5.766.689a.578.578 0 0 1-.61-.58v0Z" />
                </svg>
                <!-- code -->
                <svg v-else-if="uc.icon === 'code'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5" />
                </svg>
                <!-- cpu -->
                <svg v-else-if="uc.icon === 'cpu'" class="h-5 w-5" :style="`color: hsl(var(--${uc.color}));`" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 0 0 2.25-2.25V6.75a2.25 2.25 0 0 0-2.25-2.25H6.75A2.25 2.25 0 0 0 4.5 6.75v10.5a2.25 2.25 0 0 0 2.25 2.25Zm.75-12h9v9h-9v-9Z" />
                </svg>
              </div>
              <h3 class="text-lg font-display font-semibold">
                {{ t(`landing.useCases.${uc.key}.title`) }}
              </h3>
            </div>

            <!-- Description -->
            <p class="text-sm text-muted-foreground mb-4 flex-1">
              {{ t(`landing.useCases.${uc.key}.desc`) }}
            </p>

            <!-- Command -->
            <div class="relative group/cmd">
              <div class="bg-code rounded-lg p-3 font-mono text-xs overflow-x-auto border border-border">
                <span class="text-muted-foreground select-none">$ </span>
                <code class="text-primary">{{ t(`landing.useCases.${uc.key}.command`) }}</code>
              </div>
              <div class="absolute top-2 right-2 opacity-0 group-hover/cmd:opacity-100 transition-opacity">
                <button
                  class="p-1.5 rounded bg-surface/80 text-muted-foreground hover:text-foreground transition-colors"
                  @click="copyCommand(t(`landing.useCases.${uc.key}.command`), index)"
                >
                  <svg v-if="copiedIndex !== index" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                  </svg>
                  <svg v-else class="h-3.5 w-3.5 text-type-http" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
