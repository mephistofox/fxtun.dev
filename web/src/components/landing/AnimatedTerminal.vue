<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

interface TerminalLine {
  type: 'command' | 'output' | 'success' | 'info'
  text: string
  delay?: number
}

const lines: TerminalLine[] = [
  { type: 'command', text: 'fxtunnel http 3000 --subdomain myapp', delay: 50 },
  { type: 'info', text: 'Connecting to fxtunnel server...', delay: 800 },
  { type: 'success', text: 'Tunnel established!', delay: 600 },
  { type: 'output', text: '', delay: 100 },
  { type: 'output', text: 'HTTP: https://myapp.mfdev.ru', delay: 200 },
  { type: 'output', text: 'Forwarding to localhost:3000', delay: 150 },
  { type: 'output', text: '', delay: 100 },
  { type: 'info', text: 'Ready to receive connections', delay: 300 },
]

const displayedLines = ref<{ type: string; text: string; typing: boolean }[]>([])
const currentLineIndex = ref(0)
const currentCharIndex = ref(0)
const isTyping = ref(true)
let animationTimer: ReturnType<typeof setTimeout> | null = null

function typeNextChar() {
  if (currentLineIndex.value >= lines.length) {
    isTyping.value = false
    // Restart after pause
    animationTimer = setTimeout(() => {
      displayedLines.value = []
      currentLineIndex.value = 0
      currentCharIndex.value = 0
      isTyping.value = true
      typeNextChar()
    }, 4000)
    return
  }

  const currentLine = lines[currentLineIndex.value]

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
    const speed = currentLine.type === 'command' ? 40 : 15
    animationTimer = setTimeout(typeNextChar, speed)
  } else {
    displayedLines.value[lineIndex].typing = false
    currentLineIndex.value++
    currentCharIndex.value = 0
    animationTimer = setTimeout(typeNextChar, currentLine.delay || 300)
  }
}

onMounted(() => {
  animationTimer = setTimeout(typeNextChar, 500)
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
    <div class="terminal-body min-h-[200px]">
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
