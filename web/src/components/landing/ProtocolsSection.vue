<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

type Protocol = 'http' | 'tcp' | 'udp'

const activeProtocol = ref<Protocol>('http')

const protocols = {
  http: {
    type: 'HTTP',
    color: 'type-http',
    titleKey: 'landing.protocols.http.title',
    descKey: 'landing.protocols.http.desc',
    features: [
      'landing.protocols.http.feature1',
      'landing.protocols.http.feature2',
      'landing.protocols.http.feature3',
    ],
    example: 'https://myapp.tunnel.example.com',
    command: 'fxtunnel http 3000 --subdomain myapp',
    useCases: ['Web apps', 'APIs', 'Webhooks'],
  },
  tcp: {
    type: 'TCP',
    color: 'type-tcp',
    titleKey: 'landing.protocols.tcp.title',
    descKey: 'landing.protocols.tcp.desc',
    features: [
      'landing.protocols.tcp.feature1',
      'landing.protocols.tcp.feature2',
      'landing.protocols.tcp.feature3',
    ],
    example: 'tcp://tunnel.example.com:54321',
    command: 'fxtunnel tcp 22 --remote-port 54321',
    useCases: ['SSH', 'Databases', 'Custom protocols'],
  },
  udp: {
    type: 'UDP',
    color: 'type-udp',
    titleKey: 'landing.protocols.udp.title',
    descKey: 'landing.protocols.udp.desc',
    features: [
      'landing.protocols.udp.feature1',
      'landing.protocols.udp.feature2',
      'landing.protocols.udp.feature3',
    ],
    example: 'udp://tunnel.example.com:54322',
    command: 'fxtunnel udp 53 --remote-port 54322',
    useCases: ['Game servers', 'VoIP', 'DNS'],
  },
}

const currentProtocol = computed(() => protocols[activeProtocol.value])

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)

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
    { threshold: 0.15 }
  )

  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }
})
</script>

<template>
  <section id="protocols" ref="sectionRef" class="py-16 md:py-32 bg-background relative overflow-hidden">
    <!-- Background pattern -->
    <div class="absolute inset-0 opacity-20">
      <div class="absolute inset-0 bg-grid-pattern bg-grid-60" style="mask-image: radial-gradient(ellipse 50% 60% at 50% 30%, black 20%, transparent 70%);" />
    </div>

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-16">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.protocols.label') || 'Multi-Protocol' }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.protocols.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.protocols.subtitle') }}
        </p>
      </div>

      <!-- Protocol selector tabs -->
      <div
        class="flex justify-center mb-12 reveal reveal-delay-3"
        :class="{ 'visible': isVisible }"
      >
        <div class="inline-flex p-1.5 rounded-2xl bg-surface border border-border">
          <button
            v-for="(proto, key) in protocols"
            :key="key"
            @click="activeProtocol = key as Protocol"
            class="relative px-6 py-3 rounded-xl font-medium transition-all duration-300"
            :class="[
              activeProtocol === key
                ? `bg-${proto.color}/15 text-${proto.color} shadow-sm`
                : 'text-muted-foreground hover:text-foreground'
            ]"
          >
            <span class="flex items-center gap-2">
              <span
                class="w-2 h-2 rounded-full transition-all duration-300"
                :class="activeProtocol === key ? `bg-${proto.color}` : 'bg-muted-foreground/30'"
              />
              {{ proto.type }}
            </span>
          </button>
        </div>
      </div>

      <!-- Protocol content -->
      <div
        class="max-w-5xl mx-auto reveal reveal-delay-4"
        :class="{ 'visible': isVisible }"
      >
        <div class="grid lg:grid-cols-2 gap-8 items-center">
          <!-- Left: Info -->
          <div class="space-y-6">
            <!-- Title & Description -->
            <div>
              <h3 class="text-2xl font-display font-semibold mb-4">
                {{ t(currentProtocol.titleKey) }}
              </h3>
              <p class="text-muted-foreground leading-relaxed break-words">
                {{ t(currentProtocol.descKey) }}
              </p>
            </div>

            <!-- Features -->
            <ul class="space-y-3">
              <li
                v-for="(feature, index) in currentProtocol.features"
                :key="index"
                class="flex items-start gap-3"
              >
                <div :class="`w-5 h-5 rounded-full bg-${currentProtocol.color}/20 flex items-center justify-center flex-shrink-0 mt-0.5`">
                  <svg :class="`h-3 w-3 text-${currentProtocol.color}`" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                  </svg>
                </div>
                <span class="text-foreground/90">{{ t(feature) }}</span>
              </li>
            </ul>

            <!-- Use cases -->
            <div class="flex flex-wrap gap-2 pt-2">
              <span
                v-for="useCase in currentProtocol.useCases"
                :key="useCase"
                class="px-3 py-1 text-xs font-medium rounded-full bg-surface border border-border text-muted-foreground"
              >
                {{ useCase }}
              </span>
            </div>
          </div>

          <!-- Right: Visual -->
          <div class="glass-card-glow p-4 sm:p-6 lg:p-8 overflow-hidden">
            <!-- URL preview -->
            <div class="mb-6">
              <p class="text-xs text-muted-foreground mb-2 uppercase tracking-wider">Public endpoint</p>
              <div :class="`flex items-center gap-2 p-3 rounded-lg bg-${currentProtocol.color}/10 border border-${currentProtocol.color}/30 overflow-hidden`">
                <div :class="`pulse-indicator flex-shrink-0`" :style="`background: hsl(var(--${currentProtocol.color}))`" />
                <code :class="`font-mono text-xs sm:text-sm text-${currentProtocol.color} truncate`">
                  {{ currentProtocol.example }}
                </code>
              </div>
            </div>

            <!-- Command -->
            <div>
              <p class="text-xs text-muted-foreground mb-2 uppercase tracking-wider">Command</p>
              <div class="relative group">
                <div class="bg-[hsl(220,20%,6%)] rounded-lg p-4 font-mono text-xs sm:text-sm border border-border overflow-x-auto">
                  <span class="text-primary">$</span>
                  <span class="text-foreground/90 ml-2">{{ currentProtocol.command }}</span>
                </div>
                <button class="absolute top-3 right-3 p-2 rounded-md bg-surface/50 text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity">
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Data flow visualization -->
            <div class="mt-6 pt-6 border-t border-border">
              <div class="flex items-center justify-between text-xs text-muted-foreground">
                <span>Local</span>
                <span>Public</span>
              </div>
              <div class="mt-2 relative h-1.5 rounded-full bg-border/50 overflow-hidden">
                <!-- Static track -->
                <div class="absolute inset-0 rounded-full" :class="`bg-${currentProtocol.color}/20`" />
                <!-- Animated particles -->
                <div class="data-flow-particles" :style="`--flow-color: var(--${currentProtocol.color})`">
                  <span /><span /><span /><span /><span />
                </div>
              </div>
              <div class="flex items-center justify-between mt-2 text-sm">
                <span class="font-mono text-foreground/70">localhost:3000</span>
                <span :class="`font-mono text-${currentProtocol.color} text-xs sm:text-sm truncate`">{{ currentProtocol.example.split('://')[1] }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
