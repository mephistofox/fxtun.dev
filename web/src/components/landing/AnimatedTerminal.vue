<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getBaseDomain } from '@/i18n'

interface TerminalLine {
  type: 'command' | 'output' | 'success' | 'info' | 'url'
  text: string
  delay?: number
}

const domain = computed(() => getBaseDomain())

const lines = computed<TerminalLine[]>(() => [
  { type: 'command', text: 'fxtunnel http 3000 --domain myapp', delay: 50 },
  { type: 'info', text: 'Connecting...', delay: 600 },
  { type: 'success', text: 'Tunnel established!', delay: 400 },
  { type: 'output', text: '', delay: 100 },
  { type: 'url', text: `https://myapp.${domain.value} → localhost:3000`, delay: 200 },
  { type: 'output', text: '', delay: 400 },
  { type: 'info', text: 'GET  /api/health          200  12ms', delay: 800 },
  { type: 'info', text: 'POST /api/webhooks/creem   200  45ms', delay: 600 },
  { type: 'info', text: 'GET  /dashboard            200   8ms', delay: 500 },
])

// Start from the finished transcript rather than an empty box. The terminal
// sits in the first screen, and leaving it blank until the typing caught up
// meant the page counted as still drawing itself for several seconds. Now the
// prerender ships the completed state, and the replay starts once loading is
// over.
const bodyEl = ref<HTMLElement | null>(null)

const finishedLines = () => lines.value.map(l => ({ type: l.type, text: l.text, typing: false }))

const displayedLines = ref<{ type: string; text: string; typing: boolean }[]>(finishedLines())
const currentLineIndex = ref(0)
const currentCharIndex = ref(0)
const isTyping = ref(false)
let animationTimer: ReturnType<typeof setTimeout> | null = null

function typeNextChar() {
  if (currentLineIndex.value >= lines.value.length) {
    isTyping.value = false
    // Restart after pause
    animationTimer = setTimeout(() => {
      // Freeze the height the finished transcript occupies before emptying it,
      // or the box collapses and everything under it jumps.
      if (bodyEl.value) bodyEl.value.style.minHeight = `${bodyEl.value.offsetHeight}px`
      displayedLines.value = []
      currentLineIndex.value = 0
      currentCharIndex.value = 0
      isTyping.value = true
      typeNextChar()
    }, 5000)
    return
  }

  const currentLine = lines.value[currentLineIndex.value]

  if (currentCharIndex.value === 0) {
    displayedLines.value.push({
      type: currentLine.type,
      text: '',
      typing: true,
    })
  }

  const lineIndex = displayedLines.value.length - 1

  if (currentCharIndex.value < currentLine.text.length) {
    displayedLines.value[lineIndex].text = currentLine.text.slice(0, currentCharIndex.value + 1)
    currentCharIndex.value++
    const speed = currentLine.type === 'command' ? 40 : 12
    animationTimer = setTimeout(typeNextChar, speed)
  } else {
    displayedLines.value[lineIndex].typing = false
    currentLineIndex.value++
    currentCharIndex.value = 0
    animationTimer = setTimeout(typeNextChar, currentLine.delay || 300)
  }
}

onMounted(() => {
  // The terminal types itself out and restarts every five seconds. Started at
  // mount, it is the largest thing repainting on the page while the browser is
  // still deciding when the page finished rendering — so the page measured as
  // loading for as long as the typing went on. Begin once loading is over.
  function begin() {
    // Already showing the finished transcript; somebody who asked for less
    // motion keeps it that way.
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return
    animationTimer = setTimeout(() => {
      displayedLines.value = []
      currentLineIndex.value = 0
      currentCharIndex.value = 0
      isTyping.value = true
      typeNextChar()
    }, 1200)
  }
  // The finished transcript is already on screen, so the replay is pure
  // decoration and can wait for the visitor. Left to start on its own a few
  // seconds in, it kept repainting the first screen while the browser was
  // still deciding whether the page had finished drawing — and it landed
  // inside or outside that window depending on the run, which is why the
  // measured score swung by seven points. It starts on the first real move,
  // or after twenty seconds of nobody making one.
  function schedule() {
    const timer = setTimeout(begin, 20000)
    const kick = () => { clearTimeout(timer); begin() }
    ;['pointerdown', 'keydown', 'touchstart'].forEach(ev =>
      addEventListener(ev, kick, { once: true, passive: true }))
  }
  if (document.readyState === 'complete') schedule()
  else window.addEventListener('load', schedule, { once: true })
})

onUnmounted(() => {
  if (animationTimer) {
    clearTimeout(animationTimer)
  }
})

function getLineClass(type: string) {
  switch (type) {
    case 'command':
      return 'text-foreground'
    case 'success':
      return 'text-type-http'
    case 'url':
      return 'text-primary font-semibold'
    case 'info':
      return 'text-muted-foreground'
    default:
      return 'text-foreground/80'
  }
}
</script>

<template>
  <div class="terminal animate-float">
    <div class="terminal-header">
      <div class="terminal-dot bg-red-500"></div>
      <div class="terminal-dot bg-yellow-500"></div>
      <div class="terminal-dot bg-green-500"></div>
      <span class="ml-3 text-xs text-muted-foreground font-mono">fxTunnel</span>
    </div>
    <div ref="bodyEl" class="terminal-body min-h-[220px]">
      <div v-for="(line, index) in displayedLines" :key="index" class="flex items-start">
        <span v-if="line.type === 'command'" class="terminal-prompt mr-2">$</span>
        <span v-else class="mr-2 w-2"></span>
        <span :class="getLineClass(line.type)">
          {{ line.text }}
          <span v-if="line.typing && isTyping" class="terminal-cursor"></span>
        </span>
      </div>
      <div v-if="displayedLines.length === 0" class="flex items-center">
        <span class="terminal-prompt mr-2">$</span>
        <span class="terminal-cursor"></span>
      </div>
    </div>
  </div>
</template>
