<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const canvasRef = ref<HTMLCanvasElement | null>(null)

// --- Simplex 3D Noise ---
const G3 = [
  [1,1,0],[-1,1,0],[1,-1,0],[-1,-1,0],
  [1,0,1],[-1,0,1],[1,0,-1],[-1,0,-1],
  [0,1,1],[0,-1,1],[0,1,-1],[0,-1,-1],
]

function createNoise(seed: number) {
  const p = new Uint8Array(256)
  const perm = new Uint8Array(512)
  for (let i = 0; i < 256; i++) p[i] = i
  let s = seed * 2147483647 || 1
  for (let i = 255; i > 0; i--) {
    s = (s * 16807) % 2147483647
    const j = s % (i + 1);
    [p[i], p[j]] = [p[j], p[i]]
  }
  for (let i = 0; i < 512; i++) perm[i] = p[i & 255]

  return (xin: number, yin: number, zin: number): number => {
    const F3 = 1 / 3, G = 1 / 6
    const s = (xin + yin + zin) * F3
    const i = Math.floor(xin + s), j = Math.floor(yin + s), k = Math.floor(zin + s)
    const t = (i + j + k) * G
    const x0 = xin - (i - t), y0 = yin - (j - t), z0 = zin - (k - t)
    let i1: number, j1: number, k1: number, i2: number, j2: number, k2: number
    if (x0 >= y0) {
      if (y0 >= z0) { i1=1;j1=0;k1=0;i2=1;j2=1;k2=0 }
      else if (x0 >= z0) { i1=1;j1=0;k1=0;i2=1;j2=0;k2=1 }
      else { i1=0;j1=0;k1=1;i2=1;j2=0;k2=1 }
    } else {
      if (y0 < z0) { i1=0;j1=0;k1=1;i2=0;j2=1;k2=1 }
      else if (x0 < z0) { i1=0;j1=1;k1=0;i2=0;j2=1;k2=1 }
      else { i1=0;j1=1;k1=0;i2=1;j2=1;k2=0 }
    }
    const x1 = x0-i1+G, y1 = y0-j1+G, z1 = z0-k1+G
    const x2 = x0-i2+2*G, y2 = y0-j2+2*G, z2 = z0-k2+2*G
    const x3 = x0-1+3*G, y3 = y0-1+3*G, z3 = z0-1+3*G
    const ii = i & 255, jj = j & 255, kk = k & 255
    const gi0 = perm[ii + perm[jj + perm[kk]]] % 12
    const gi1 = perm[ii+i1 + perm[jj+j1 + perm[kk+k1]]] % 12
    const gi2 = perm[ii+i2 + perm[jj+j2 + perm[kk+k2]]] % 12
    const gi3 = perm[ii+1 + perm[jj+1 + perm[kk+1]]] % 12
    const d = (g: number[], x: number, y: number, z: number) => g[0]*x + g[1]*y + g[2]*z
    let t0 = 0.6-x0*x0-y0*y0-z0*z0
    const n0 = t0 < 0 ? 0 : (t0*=t0, t0*t0*d(G3[gi0],x0,y0,z0))
    let t1 = 0.6-x1*x1-y1*y1-z1*z1
    const n1 = t1 < 0 ? 0 : (t1*=t1, t1*t1*d(G3[gi1],x1,y1,z1))
    let t2 = 0.6-x2*x2-y2*y2-z2*z2
    const n2 = t2 < 0 ? 0 : (t2*=t2, t2*t2*d(G3[gi2],x2,y2,z2))
    let t3 = 0.6-x3*x3-y3*y3-z3*z3
    const n3 = t3 < 0 ? 0 : (t3*=t3, t3*t3*d(G3[gi3],x3,y3,z3))
    return 32 * (n0 + n1 + n2 + n3)
  }
}

function marchContours(
  grid: Float32Array, cols: number, rows: number,
  threshold: number, cellW: number, cellH: number,
  ctx: CanvasRenderingContext2D,
) {
  for (let j = 0; j < rows - 1; j++) {
    const jOff = j * cols, jOff1 = jOff + cols
    for (let i = 0; i < cols - 1; i++) {
      const tl = grid[jOff + i], tr = grid[jOff + i + 1]
      const br = grid[jOff1 + i + 1], bl = grid[jOff1 + i]
      let ci = 0
      if (tl > threshold) ci |= 1
      if (tr > threshold) ci |= 2
      if (br > threshold) ci |= 4
      if (bl > threshold) ci |= 8
      if (ci === 0 || ci === 15) continue
      const x = i * cellW, y = j * cellH
      let dd: number
      dd = tr - tl; const tx = x + (dd === 0 ? 0.5 : (threshold - tl) / dd) * cellW
      const ty = y
      const rx = x + cellW
      dd = br - tr; const ry = y + (dd === 0 ? 0.5 : (threshold - tr) / dd) * cellH
      dd = br - bl; const bx = x + (dd === 0 ? 0.5 : (threshold - bl) / dd) * cellW
      const by = y + cellH
      const lx = x
      dd = bl - tl; const ly = y + (dd === 0 ? 0.5 : (threshold - tl) / dd) * cellH
      switch (ci) {
        case 1: case 14: ctx.moveTo(lx, ly); ctx.lineTo(tx, ty); break
        case 2: case 13: ctx.moveTo(tx, ty); ctx.lineTo(rx, ry); break
        case 3: case 12: ctx.moveTo(lx, ly); ctx.lineTo(rx, ry); break
        case 4: case 11: ctx.moveTo(rx, ry); ctx.lineTo(bx, by); break
        case 6: case 9:  ctx.moveTo(tx, ty); ctx.lineTo(bx, by); break
        case 7: case 8:  ctx.moveTo(lx, ly); ctx.lineTo(bx, by); break
        case 5:  ctx.moveTo(lx, ly); ctx.lineTo(tx, ty); ctx.moveTo(rx, ry); ctx.lineTo(bx, by); break
        case 10: ctx.moveTo(tx, ty); ctx.lineTo(rx, ry); ctx.moveTo(lx, ly); ctx.lineTo(bx, by); break
      }
    }
  }
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')!
  const noise = createNoise(42)

  let timer: ReturnType<typeof setInterval> | null = null
  let displayW = 0
  let displayH = 0
  let grid: Float32Array | null = null
  let gridCols = 0
  let gridRows = 0
  let time = 0

  const CELL = 14
  const FREQ = 0.012
  const OCTAVES = 2
  const THRESHOLDS = Array.from({ length: 9 }, (_, i) => -0.44 + i * 0.11)
  const TIME_STEP = 0.003 // increment per frame at ~30fps

  let lineColor = 'rgba(128,135,150,0.8)'
  function updateColor() {
    const raw = getComputedStyle(document.documentElement).getPropertyValue('--border').trim()
    if (raw) lineColor = `hsl(${raw} / 0.8)`
  }
  updateColor()

  const themeObserver = new MutationObserver(updateColor)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  function resize() {
    const rect = canvas!.parentElement!.getBoundingClientRect()
    displayW = rect.width
    displayH = rect.height
    canvas!.width = displayW
    canvas!.height = displayH
    gridCols = Math.ceil(displayW / CELL) + 1
    gridRows = Math.ceil(displayH / CELL) + 1
    grid = new Float32Array(gridCols * gridRows)
    draw() // immediate redraw on resize
  }
  resize()
  window.addEventListener('resize', resize)

  function draw() {
    if (!grid) return

    for (let j = 0; j < gridRows; j++) {
      for (let i = 0; i < gridCols; i++) {
        let v = 0, a = 1, t = 0, fx = i * FREQ, fy = j * FREQ
        for (let o = 0; o < OCTAVES; o++) {
          v += noise(fx, fy, time) * a
          t += a; fx *= 2; fy *= 2; a *= 0.5
        }
        grid[j * gridCols + i] = v / t
      }
    }

    ctx.clearRect(0, 0, displayW, displayH)
    ctx.strokeStyle = lineColor
    ctx.lineWidth = 0.8
    ctx.beginPath()
    for (const threshold of THRESHOLDS) {
      marchContours(grid, gridCols, gridRows, threshold, CELL, CELL, ctx)
    }
    ctx.stroke()

    time += TIME_STEP
  }

  // Pause when off-screen
  let visible = true
  const io = new IntersectionObserver(
    ([e]) => {
      visible = e.isIntersecting
      if (visible && !timer) {
        timer = setInterval(draw, 16) // ~60 FPS
      } else if (!visible && timer) {
        clearInterval(timer)
        timer = null
      }
    },
    { threshold: 0 },
  )
  io.observe(canvas)

  // Start animation at ~10 FPS (enough for slow background drift)
  timer = setInterval(draw, 33)

  onUnmounted(() => {
    if (timer) clearInterval(timer)
    window.removeEventListener('resize', resize)
    themeObserver.disconnect()
    io.disconnect()
  })
})
</script>

<template>
  <canvas
    ref="canvasRef"
    class="absolute inset-0 w-full h-full pointer-events-none"
    style="-webkit-mask-image: radial-gradient(ellipse 80% 60% at 50% 50%, black 20%, transparent 70%); mask-image: radial-gradient(ellipse 80% 60% at 50% 50%, black 20%, transparent 70%);"
  />
</template>
