<script setup lang="ts">
/**
 * One view for every documentation section. The section text arrives as
 * finished HTML from plugins/docs-md.ts; this view adds the chrome around it —
 * section menu, on-page contents, previous/next and the copy buttons.
 *
 * The docs are Russian-only, so the interface strings live here rather than in
 * the i18n files.
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema } from '@/composables/useStructuredData'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import manifest from '@/docs/manifest.json'
import { docPath, type DocPage, type DocsEntry } from '@/docs/types'
import { getBlogUrl } from '@/i18n'
import { reachGoal } from '@/lib/metrika'

const props = defineProps<{ doc: DocPage; entry: DocsEntry }>()

const { t } = useI18n()
const router = useRouter()
const entries = manifest as DocsEntry[]
const path = docPath(props.entry.slug)
const blogUrl = getBlogUrl()

const crumbs = props.entry.slug
  ? [{ name: 'Документация', path: '/docs' }, { name: props.entry.nav, path }]
  : [{ name: 'Документация', path: '/docs' }]

useSeo({ title: props.doc.title, description: props.doc.description, type: 'article' })
useSubpageSchema({
  path,
  name: props.doc.title,
  description: props.doc.description,
  dateModified: props.entry.lastmod,
  breadcrumbs: crumbs,
})

const groups: { name: string; items: DocsEntry[] }[] = []
for (const e of entries) {
  let g = groups.find(x => x.name === e.group)
  if (!g) groups.push((g = { name: e.group, items: [] }))
  g.items.push(e)
}

const position = entries.findIndex(e => e.slug === props.entry.slug)
const prev = entries[position - 1]
const next = entries[position + 1]

const heading = props.entry.slug ? props.entry.nav : 'Документация fxTunnel'
const accent = `var(--${props.entry.tag ? `type-${props.entry.tag}` : 'primary'})`
const updated = new Date(props.entry.lastmod).toLocaleDateString('ru-RU', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})

const menuOpen = ref(false)
const activeId = ref(props.doc.headings[0]?.id ?? '')
const bodyEl = ref<HTMLElement>()
let observer: IntersectionObserver | undefined

onMounted(() => {
  observer = new IntersectionObserver(
    changes => changes.forEach(c => { if (c.isIntersecting) activeId.value = c.target.id }),
    { rootMargin: '-80px 0px -70% 0px' },
  )
  bodyEl.value?.querySelectorAll('h2[id], h3[id]').forEach(h => observer!.observe(h))
})
onBeforeUnmount(() => observer?.disconnect())

// One listener for the whole rendered text: copy buttons, and links to other
// docs pages that should switch pages through the router instead of reloading.
async function onBodyClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  const button = target.closest<HTMLButtonElement>('[data-copy]')
  if (button) {
    await navigator.clipboard.writeText(button.dataset.copy ?? '')
    reachGoal('copy_command')
    button.textContent = 'Скопировано ✓'
    button.classList.add('is-done')
    setTimeout(() => {
      button.textContent = 'Копировать'
      button.classList.remove('is-done')
    }, 2000)
    return
  }
  const link = target.closest<HTMLAnchorElement>('a[href^="/docs"]')
  if (link && e.button === 0 && !e.metaKey && !e.ctrlKey && !e.shiftKey && !e.altKey) {
    e.preventDefault()
    router.push(link.getAttribute('href')!)
  }
}
</script>

<template>
  <div class="page-shell">
    <nav class="page-nav">
      <div class="container mx-auto px-4 flex items-center h-16 gap-8">
        <RouterLink to="/" class="page-nav-brand">
          <div class="page-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="hidden lg:flex items-center gap-7">
          <RouterLink to="/features" class="page-nav-link">{{ t('landing.nav.features') }}</RouterLink>
          <RouterLink to="/pricing" class="page-nav-link">{{ t('landing.nav.pricing') }}</RouterLink>
          <RouterLink to="/downloads" class="page-nav-link">{{ t('landing.nav.download') }}</RouterLink>
          <RouterLink to="/docs" class="page-nav-link docs-nav-current">{{ t('landing.nav.docs') }}</RouterLink>
          <a :href="blogUrl" class="page-nav-link">{{ t('landing.nav.blog') }}</a>
        </div>
        <div class="flex items-center gap-4 ml-auto">
          <RouterLink to="/login" class="page-nav-link hidden sm:inline">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="page-nav-cta whitespace-nowrap">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <main class="pt-16">
      <div class="docs-shell">
        <aside class="docs-side" aria-label="Разделы документации">
          <div v-for="g in groups" :key="g.name" class="docs-side-group">
            <p class="docs-side-title">{{ g.name }}</p>
            <RouterLink
              v-for="e in g.items"
              :key="e.slug"
              :to="docPath(e.slug)"
              :class="['docs-side-link', { 'is-active': e.slug === entry.slug }]"
            >
              {{ e.nav }}
              <span v-if="e.tag" :class="['docs-tag', `docs-tag-${e.tag}`]">{{ e.tag.toUpperCase() }}</span>
            </RouterLink>
          </div>
        </aside>

        <article class="docs-main">
          <Breadcrumbs :items="crumbs" class="docs-crumbs" />

          <div class="docs-mobile">
            <button type="button" class="docs-mobile-toggle" :aria-expanded="menuOpen" @click="menuOpen = !menuOpen">
              <span>☰ Разделы</span>
              <span class="docs-mobile-current">{{ entry.nav }} ▾</span>
            </button>
            <div v-if="menuOpen" class="docs-mobile-panel">
              <div v-for="g in groups" :key="g.name" class="docs-side-group">
                <p class="docs-side-title">{{ g.name }}</p>
                <RouterLink
                  v-for="e in g.items"
                  :key="e.slug"
                  :to="docPath(e.slug)"
                  :class="['docs-side-link', { 'is-active': e.slug === entry.slug }]"
                >
                  {{ e.nav }}
                </RouterLink>
              </div>
            </div>
          </div>

          <header class="docs-hero" :style="{ '--doc-accent': accent }">
            <span class="docs-pill">● {{ entry.group }}</span>
            <h1 class="docs-title">{{ heading }}</h1>
            <p class="docs-lead">{{ doc.description }}</p>
            <div class="docs-meta">
              <span>{{ doc.minutes }} мин чтения</span>
              <span>Обновлено {{ updated }}</span>
            </div>
          </header>

          <div ref="bodyEl" class="docs-body" @click="onBodyClick" v-html="doc.html" />

          <nav class="docs-pager" aria-label="Соседние разделы">
            <RouterLink v-if="prev" :to="docPath(prev.slug)" class="docs-pager-link">
              <small>← Назад</small><span>{{ prev.nav }}</span>
            </RouterLink>
            <span v-else />
            <RouterLink v-if="next" :to="docPath(next.slug)" class="docs-pager-link docs-pager-next">
              <small>Далее →</small><span>{{ next.nav }}</span>
            </RouterLink>
          </nav>
        </article>

        <aside class="docs-toc" aria-label="На этой странице">
          <template v-if="doc.headings.length">
            <p class="docs-side-title">На этой странице</p>
            <a
              v-for="h in doc.headings"
              :key="h.id"
              :href="`#${h.id}`"
              :class="['docs-toc-link', { 'is-l3': h.level === 3, 'is-active': h.id === activeId }]"
            >{{ h.text }}</a>
          </template>
          <div class="docs-help">
            Не нашли ответ? Напишите нам в
            <a href="https://t.me/mephistofx" target="_blank" rel="noopener">Telegram</a>
          </div>
        </aside>
      </div>
    </main>

    <LandingFooter />
  </div>
</template>

<style scoped>
:global(html:has(.docs-shell)) {
  scroll-behavior: smooth;
}

.docs-nav-current {
  color: hsl(var(--primary));
}

.docs-shell {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 24px;
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr) 220px;
  gap: 48px;
}

/* ---- sidebar & contents ---- */
.docs-side,
.docs-toc {
  position: sticky;
  top: 64px;
  align-self: start;
  max-height: calc(100vh - 64px);
  overflow-y: auto;
}
.docs-side { padding: 32px 0; }
.docs-toc { padding: 40px 0; font-size: 13.5px; }

.docs-side-group { margin-bottom: 26px; }
.docs-side-title {
  margin: 0 0 8px 12px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: hsl(var(--muted-foreground));
}
.docs-toc .docs-side-title { margin-left: 0; }

.docs-side-link {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 12px;
  border-radius: 8px;
  border-left: 2px solid transparent;
  font-size: 14.5px;
  color: hsl(var(--muted-foreground));
  transition: color 0.15s, background 0.15s;
}
.docs-side-link:hover { color: hsl(var(--foreground)); background: hsl(var(--muted) / 0.6); }
.docs-side-link.is-active {
  color: hsl(var(--primary));
  font-weight: 500;
  background: linear-gradient(90deg, hsl(var(--primary) / 0.12), transparent);
  border-left-color: hsl(var(--primary));
  border-radius: 0 8px 8px 0;
}

.docs-tag {
  @apply font-mono;
  margin-left: auto;
  padding: 1px 6px;
  border-radius: 5px;
  font-size: 10px;
  font-weight: 600;
}
.docs-tag-http { color: hsl(var(--type-http)); background: hsl(var(--type-http) / 0.12); }
.docs-tag-tcp { color: hsl(var(--type-tcp)); background: hsl(var(--type-tcp) / 0.12); }
.docs-tag-udp { color: hsl(var(--type-udp)); background: hsl(var(--type-udp) / 0.12); }

.docs-toc-link {
  display: block;
  padding: 5px 0 5px 14px;
  border-left: 1px solid hsl(var(--border));
  color: hsl(var(--muted-foreground));
  transition: color 0.15s;
}
.docs-toc-link.is-l3 { padding-left: 26px; font-size: 13px; }
.docs-toc-link:hover { color: hsl(var(--foreground)); }
.docs-toc-link.is-active {
  color: hsl(var(--primary));
  border-left: 2px solid hsl(var(--primary));
  padding-left: 13px;
}
.docs-toc-link.is-l3.is-active { padding-left: 25px; }

.docs-help {
  margin-top: 26px;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--card));
  font-size: 13px;
  color: hsl(var(--muted-foreground));
}
.docs-help a { color: hsl(var(--primary)); }

/* ---- main column ---- */
.docs-main { min-width: 0; max-width: 780px; padding: 16px 0 96px; }
.docs-crumbs { max-width: none; padding-left: 0; padding-right: 0; }

.docs-hero {
  position: relative;
  overflow: hidden;
  margin: 16px 0 36px;
  padding: 32px;
  border-radius: 20px;
  border: 1px solid hsl(var(--border));
  background:
    radial-gradient(600px 220px at 100% 0%, hsl(var(--doc-accent) / 0.16), transparent 70%),
    radial-gradient(500px 200px at 0% 100%, hsl(var(--primary) / 0.08), transparent 70%),
    hsl(var(--card));
}
.docs-hero::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  opacity: 0.5;
  background-image:
    linear-gradient(hsl(var(--border) / 0.5) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.5) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: linear-gradient(90deg, transparent 40%, #000);
}
.docs-pill {
  position: relative;
  z-index: 1;
  display: inline-flex;
  padding: 4px 11px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: hsl(var(--doc-accent));
  background: hsl(var(--doc-accent) / 0.12);
  border: 1px solid hsl(var(--doc-accent) / 0.25);
}
.docs-title {
  @apply font-display;
  position: relative;
  z-index: 1;
  margin: 14px 0 10px;
  font-size: clamp(30px, 4vw, 42px);
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.01em;
}
.docs-lead {
  position: relative;
  z-index: 1;
  max-width: 580px;
  font-size: 17px;
  color: hsl(var(--muted-foreground));
}
.docs-meta {
  position: relative;
  z-index: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  margin-top: 18px;
  font-size: 13px;
  color: hsl(var(--muted-foreground));
}
.docs-meta span::before {
  content: '●';
  margin-right: 7px;
  font-size: 8px;
  vertical-align: 2px;
  color: hsl(var(--primary));
}

/* ---- rendered markdown ---- */
.docs-body { font-size: 16px; line-height: 1.7; }
.docs-body :deep(h2) {
  @apply font-display;
  margin: 52px 0 14px;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.3;
  scroll-margin-top: 88px;
}
.docs-body :deep(h3) { margin: 32px 0 8px; font-size: 18px; font-weight: 600; scroll-margin-top: 88px; }
.docs-body :deep(.doc-anchor) {
  margin-left: 10px;
  font-weight: 400;
  color: hsl(var(--primary));
  text-decoration: none;
  opacity: 0;
  transition: opacity 0.15s;
}
.docs-body :deep(h2:hover .doc-anchor),
.docs-body :deep(h3:hover .doc-anchor) { opacity: 1; }
.docs-body :deep(p) { margin: 0 0 14px; }
.docs-body :deep(ul) { margin: 0 0 14px; padding-left: 22px; list-style: disc; }
.docs-body :deep(ol) { margin: 0 0 14px; padding-left: 22px; list-style: decimal; }
.docs-body :deep(li) { margin: 4px 0; }
.docs-body :deep(li::marker) { color: hsl(var(--primary)); }
.docs-body :deep(a:not(.doc-anchor)) {
  color: hsl(var(--primary));
  text-decoration: underline;
  text-decoration-color: hsl(var(--primary) / 0.4);
  text-underline-offset: 3px;
}
.docs-body :deep(:not(pre) > code) {
  @apply font-mono;
  padding: 2px 7px;
  border-radius: 6px;
  font-size: 0.86em;
  color: hsl(var(--primary));
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
}
.docs-body :deep(blockquote) {
  margin: 18px 0;
  padding: 4px 0 4px 16px;
  border-left: 3px solid hsl(var(--border));
  color: hsl(var(--muted-foreground));
}
.docs-body :deep(hr) { margin: 36px 0; border: 0; border-top: 1px solid hsl(var(--border)); }

/* terminal — dark in both themes, like the one on the home page */
.docs-body :deep(.term) {
  margin: 18px 0 22px;
  overflow: hidden;
  border-radius: 14px;
  background: hsl(220 20% 6%);
  border: 1px solid hsl(220 15% 15%);
  box-shadow: 0 10px 40px -18px hsl(0 0% 0% / 0.7);
  transition: border-color 0.2s;
}
.docs-body :deep(.term:hover) { border-color: hsl(75 100% 50% / 0.35); }
.docs-body :deep(.term-bar) {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 10px 14px;
  border-bottom: 1px solid hsl(220 15% 15%);
}
.docs-body :deep(.term-bar i) { width: 11px; height: 11px; border-radius: 50%; }
.docs-body :deep(.term-bar i:nth-child(1)) { background: #ff5f57; }
.docs-body :deep(.term-bar i:nth-child(2)) { background: #febc2e; }
.docs-body :deep(.term-bar i:nth-child(3)) { background: #28c840; }
.docs-body :deep(.term-lang) {
  @apply font-mono;
  margin-left: 8px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: hsl(220 10% 55%);
}
.docs-body :deep(.term-copy) {
  margin-left: auto;
  padding: 4px 10px;
  border-radius: 7px;
  font-size: 12px;
  color: hsl(220 10% 60%);
  background: hsl(220 15% 12%);
  border: 1px solid hsl(220 15% 18%);
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.docs-body :deep(.term-copy:hover) { color: hsl(75 100% 50%); border-color: hsl(75 100% 50% / 0.4); }
.docs-body :deep(.term-copy.is-done) { color: hsl(160 84% 45%); border-color: hsl(160 84% 45% / 0.4); }
.docs-body :deep(.term pre) { margin: 0; padding: 16px 18px; overflow-x: auto; }
.docs-body :deep(.term code) { @apply font-mono; font-size: 14px; line-height: 1.75; color: hsl(0 0% 92%); }
.docs-body :deep(.t-ps) { margin-right: 10px; color: hsl(75 100% 50%); user-select: none; }
.docs-body :deep(.t-flag) { color: hsl(217 91% 70%); }
.docs-body :deep(.t-cm) { color: hsl(220 10% 45%); }
.docs-body :deep(.t-out) { color: hsl(160 84% 50%); }

/* callouts */
.docs-body :deep(.callout) {
  display: flex;
  gap: 14px;
  margin: 22px 0;
  padding: 16px 18px;
  border-radius: 14px;
  font-size: 15px;
  --callout: var(--primary);
  background: hsl(var(--callout) / 0.06);
  border: 1px solid hsl(var(--callout) / 0.22);
}
.docs-body :deep(.callout-warn) { --callout: 38 85% 55%; }
.docs-body :deep(.callout-note) { --callout: var(--type-tcp); }
.docs-body :deep(.callout-ico) {
  flex: none;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  font-size: 14px;
  color: hsl(var(--callout));
  background: hsl(var(--callout) / 0.15);
}
.docs-body :deep(.callout-body b) {
  display: block;
  margin-bottom: 2px;
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: hsl(var(--callout));
}
.docs-body :deep(.callout-body p) { margin: 0; }

/* tables */
.docs-body :deep(.doc-table) {
  margin: 18px 0;
  overflow-x: auto;
  border-radius: 14px;
  border: 1px solid hsl(var(--border));
}
.docs-body :deep(table) { width: 100%; border-collapse: collapse; font-size: 14px; }
.docs-body :deep(th) {
  padding: 12px 16px;
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted) / 0.5);
}
.docs-body :deep(td) { padding: 12px 16px; vertical-align: top; border-top: 1px solid hsl(var(--border)); }
.docs-body :deep(tr:hover td) { background: hsl(var(--primary) / 0.04); }
.docs-body :deep(td:first-child code) { white-space: nowrap; }

/* ---- previous / next ---- */
.docs-pager { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-top: 64px; }
.docs-pager-link {
  padding: 18px 20px;
  border-radius: 14px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--card));
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.2s;
}
.docs-pager-link:hover {
  border-color: hsl(var(--primary) / 0.5);
  box-shadow: 0 0 0 4px hsl(var(--primary) / 0.07);
  transform: translateY(-2px);
}
.docs-pager-link small { display: block; margin-bottom: 4px; font-size: 12px; color: hsl(var(--muted-foreground)); }
.docs-pager-link span { font-size: 16px; font-weight: 600; }
.docs-pager-next { text-align: right; }

/* ---- mobile section menu ---- */
.docs-mobile { display: none; }

@media (max-width: 1279px) {
  .docs-shell { grid-template-columns: 230px minmax(0, 1fr); }
  .docs-toc { display: none; }
}

@media (max-width: 900px) {
  .docs-shell { grid-template-columns: minmax(0, 1fr); padding: 0 16px; }
  .docs-side { display: none; }
  .docs-mobile { display: block; margin-top: 12px; }
  .docs-mobile-toggle {
    display: flex;
    width: 100%;
    align-items: center;
    justify-content: space-between;
    padding: 11px 14px;
    border-radius: 12px;
    border: 1px solid hsl(var(--border));
    background: hsl(var(--card));
    font-size: 14px;
  }
  .docs-mobile-current { color: hsl(var(--primary)); }
  .docs-mobile-panel {
    margin-top: 8px;
    padding: 16px 8px 4px;
    border-radius: 12px;
    border: 1px solid hsl(var(--border));
    background: hsl(var(--card));
  }
  .docs-hero { padding: 22px; }
  .docs-pager { grid-template-columns: 1fr; }
  .docs-pager-next { text-align: left; }
  .docs-body :deep(.term code) { font-size: 13px; }
}
</style>
