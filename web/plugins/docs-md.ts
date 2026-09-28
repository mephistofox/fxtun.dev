import fs from 'node:fs'
import path from 'node:path'
import MarkdownIt, { type Env } from 'markdown-it'
import type { Plugin } from 'vite'

// Mirrors DocPage in src/docs/types.ts: the plugin runs in Node and cannot import app code.
export interface RenderedDoc {
  title: string
  description: string
  html: string
  headings: { level: 2 | 3; id: string; text: string }[]
  minutes: number
}

const SHELL = new Set(['bash', 'sh', 'shell', 'console'])
const CALLOUTS: Record<string, { cls: string; label: string; icon: string }> = {
  TIP: { cls: 'tip', label: 'Совет', icon: '✦' },
  WARNING: { cls: 'warn', label: 'Внимание', icon: '!' },
  NOTE: { cls: 'note', label: 'Заметка', icon: 'i' },
}
const CALLOUT_MARK = /^\[!(TIP|WARNING|NOTE)\]\s*/

export function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

export function slugify(text: string): string {
  return text.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '')
}

export function parseFrontmatter(src: string, file: string): { data: Record<string, string>; body: string } {
  const m = /^---\r?\n([\s\S]*?)\r?\n---\r?\n/.exec(src)
  if (!m) throw new Error(`${file}: нет frontmatter`)
  const data: Record<string, string> = {}
  for (const line of m[1].split(/\r?\n/)) {
    const i = line.indexOf(':')
    if (i > 0) data[line.slice(0, i).trim()] = line.slice(i + 1).trim().replace(/^"(.*)"$/, '$1')
  }
  for (const key of ['title', 'description']) {
    if (!data[key]) throw new Error(`${file}: во frontmatter нет ${key}`)
  }
  return { data, body: src.slice(m[0].length) }
}

// Terminal look without a highlighter: a prompt before commands, dim comments,
// green output (lines written as "→ ..."), blue flags. A line continuing the
// previous one (which ends with "\") is part of the same command — no prompt.
function shellLine(line: string, continued: boolean): string {
  const safe = escapeHtml(line)
  if (line.startsWith('#')) return `<span class="t-cm">${safe}</span>`
  if (line.startsWith('→')) return `<span class="t-out">${safe}</span>`
  const flags = safe.replace(/(^|\s)(--?[a-zA-Z][\w-]*)/g, '$1<span class="t-flag">$2</span>')
  return continued || !line.trim() ? flags : `<span class="t-ps">$</span>${flags}`
}

function renderFence(content: string, lang: string): string {
  const lines = content.replace(/\n$/, '').split('\n')
  const shell = SHELL.has(lang)
  const body = shell
    ? lines.map((l, i) => shellLine(l, i > 0 && lines[i - 1].trimEnd().endsWith('\\'))).join('\n')
    : escapeHtml(lines.join('\n'))
  // Output lines are what the command prints — copying them would break the paste.
  const copy = shell ? lines.filter(l => !l.startsWith('→')).join('\n') : lines.join('\n')
  return (
    `<div class="term"><div class="term-bar"><i></i><i></i><i></i>` +
    `<span class="term-lang">${escapeHtml(lang || 'text')}</span>` +
    `<button type="button" class="term-copy" data-copy="${escapeHtml(copy)}">Копировать</button></div>` +
    `<pre><code>${body}</code></pre></div>\n`
  )
}

interface DocsEnv {
  file: string
  headings: RenderedDoc['headings']
}

function createMd() {
  const md = new MarkdownIt({ html: false })

  md.renderer.rules.fence = (tokens, idx) => {
    const t = tokens[idx]
    return renderFence(t.content, t.info.trim().split(/\s+/)[0] ?? '')
  }
  md.renderer.rules.table_open = () => '<div class="doc-table"><table>\n'
  md.renderer.rules.table_close = () => '</table></div>\n'
  md.renderer.rules.blockquote_open = (tokens, idx, options, _env, self) => {
    const kind = tokens[idx].meta?.callout as string | undefined
    if (!kind) return self.renderToken(tokens, idx, options)
    const c = CALLOUTS[kind]
    return `<div class="callout callout-${c.cls}"><div class="callout-ico" aria-hidden="true">${c.icon}</div><div class="callout-body"><b>${c.label}</b>\n`
  }
  md.renderer.rules.blockquote_close = (tokens, idx, options, _env, self) =>
    tokens[idx].meta?.callout ? '</div></div>\n' : self.renderToken(tokens, idx, options)
  md.renderer.rules.heading_close = (tokens, idx, options, _env, self) => {
    const id = tokens[idx].meta?.anchor as string | undefined
    const anchor = id ? `<a class="doc-anchor" href="#${id}" aria-hidden="true" tabindex="-1">#</a>` : ''
    return anchor + self.renderToken(tokens, idx, options)
  }

  // Runs after inline parsing: heading ids and the TOC, and GitHub-style
  // "> [!TIP]" blockquotes turned into callouts.
  md.core.ruler.push('fxtun_docs', state => {
    // markdown-it types env as an index-signature bag; ours is the concrete shape we put in it.
    const env = state.env as unknown as DocsEnv
    const tokens = state.tokens
    const seen = new Map<string, number>()
    const callouts: (string | null)[] = []
    for (let i = 0; i < tokens.length; i++) {
      const tok = tokens[i]
      if (tok.type === 'heading_open') {
        if (tok.tag === 'h1') throw new Error(`${env.file}: заголовок первого уровня в тексте — он берётся из манифеста`)
        if (tok.tag !== 'h2' && tok.tag !== 'h3') continue
        const text = (tokens[i + 1].children ?? [])
          .filter(c => c.type === 'text' || c.type === 'code_inline')
          .map(c => c.content)
          .join('')
        const base = slugify(text) || 'section'
        const n = (seen.get(base) ?? 0) + 1
        seen.set(base, n)
        const id = n === 1 ? base : `${base}-${n}`
        tok.attrSet('id', id)
        tokens[i + 2].meta = { anchor: id }
        env.headings.push({ level: tok.tag === 'h2' ? 2 : 3, id, text })
      } else if (tok.type === 'blockquote_open') {
        const inline = tokens[i + 2]
        const m = inline?.type === 'inline' ? CALLOUT_MARK.exec(inline.content) : null
        callouts.push(m ? m[1] : null)
        if (!m) continue
        tok.meta = { callout: m[1] }
        const kids = inline.children ?? []
        kids[0].content = kids[0].content.replace(CALLOUT_MARK, '')
        if (!kids[0].content && kids[1]?.type === 'softbreak') kids.splice(0, 2)
      } else if (tok.type === 'blockquote_close') {
        const kind = callouts.pop()
        if (kind) tok.meta = { callout: kind }
      }
    }
  })
  return md
}

const md = createMd()

export function renderDoc(src: string, file: string): RenderedDoc {
  const { data, body } = parseFrontmatter(src, file)
  const env: DocsEnv = { file, headings: [] }
  const html = md.render(body, env as unknown as Env)
  const words = body.replace(/```[\s\S]*?```/g, ' ').split(/\s+/).filter(Boolean).length
  return {
    title: data.title,
    description: data.description,
    html,
    headings: env.headings,
    minutes: Math.max(1, Math.round(words / 180)),
  }
}

export interface ManifestEntry {
  slug: string
  group: string
  nav: string
  lastmod: string
  tag?: string
}

// web/public/docs/offer.* is the static offer page; "index" names the overview file.
const RESERVED = new Set(['index', 'offer'])

export const docFile = (slug: string) => `${slug || 'index'}.md`

// The manifest is the only list of pages: routes, prerender and sitemap all read
// it, so a page it misses or a file it names but lacks must stop the build.
export function validateManifest(dir: string): ManifestEntry[] {
  const entries = JSON.parse(fs.readFileSync(path.join(dir, 'manifest.json'), 'utf8')) as ManifestEntry[]
  const problems: string[] = []
  const slugs = new Set<string>()
  for (const e of entries) {
    const name = e.slug || '(обзор)'
    if (!/^[a-z0-9-]*$/.test(e.slug)) problems.push(`${name}: slug только из a-z, 0-9 и -`)
    if (RESERVED.has(e.slug)) problems.push(`${name}: slug занят`)
    if (slugs.has(e.slug)) problems.push(`${name}: slug повторяется`)
    slugs.add(e.slug)
    for (const key of ['group', 'nav'] as const) if (!e[key]) problems.push(`${name}: нет ${key}`)
    if (!/^\d{4}-\d{2}-\d{2}$/.test(e.lastmod ?? '')) problems.push(`${name}: lastmod не в формате ГГГГ-ММ-ДД`)
    if (e.tag && !['http', 'tcp', 'udp'].includes(e.tag)) problems.push(`${name}: tag только http, tcp или udp`)
    if (!fs.existsSync(path.join(dir, docFile(e.slug)))) problems.push(`${name}: нет файла ${docFile(e.slug)}`)
  }
  for (const f of fs.readdirSync(dir).filter(f => f.endsWith('.md'))) {
    if (!entries.some(e => docFile(e.slug) === f)) problems.push(`${f}: нет в manifest.json`)
  }
  if (problems.length) throw new Error(`docs/manifest.json:\n  - ${problems.join('\n  - ')}`)
  return entries
}

export function docsMarkdown(dir: string): Plugin {
  const root = path.resolve(dir)
  return {
    name: 'fxtun-docs-md',
    enforce: 'pre',
    buildStart() {
      validateManifest(root)
    },
    transform(src, id) {
      const file = id.split('?')[0]
      if (!file.endsWith('.md') || path.dirname(file) !== root) return null
      return { code: `export default ${JSON.stringify(renderDoc(src, path.basename(file)))}`, map: null }
    },
  }
}
