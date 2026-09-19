import { createI18n } from 'vue-i18n'
import type { MessageCompiler } from 'vue-i18n'
import ru from './ru.json'

// Both locale files are about 1300 keys of marketing copy — a quarter of a
// megabyte of JSON. Bundling both meant every Russian visitor downloaded and
// parsed the English site as well, and vice versa. Only the Russian messages
// ship in the entry; the English ones are a chunk of their own, fetched when a
// route or a visitor actually asks for them. The two files are kept key-for-key
// identical, so nothing ever has to fall back across them.
type MessageSchema = typeof ru

// CSP-compatible message compiler: interprets {name} and {'literal'}
// without using new Function() (which requires unsafe-eval)
const cspMessageCompiler: MessageCompiler = (message) => {
  if (typeof message === 'function') return message as any

  const msg = String(message)

  // Parse message into parts: text, named interpolation {key}, literal {'|'}
  const parts: Array<{ t: 'x', v: string } | { t: 'n', k: string }> = []
  let lastIdx = 0
  const re = /\{'([^']*)'\}|\{([^}]+)\}/g
  let m: RegExpExecArray | null

  while ((m = re.exec(msg)) !== null) {
    if (m.index > lastIdx) parts.push({ t: 'x', v: msg.slice(lastIdx, m.index) })
    if (m[1] !== undefined) {
      parts.push({ t: 'x', v: m[1] }) // literal {'|'} → |
    } else {
      parts.push({ t: 'n', k: m[2] }) // named {key}
    }
    lastIdx = re.lastIndex
  }
  if (lastIdx < msg.length) parts.push({ t: 'x', v: msg.slice(lastIdx) })

  // No interpolation — return static string
  if (parts.every(p => p.t === 'x')) {
    const text = parts.map(p => p.v).join('')
    return () => text
  }

  return (ctx: any) =>
    parts
      .map(p => (p.t === 'n' ? (ctx.named(p.k) ?? `{${p.k}}`) : p.v))
      .join('')
}

export function getDomainLocale(): 'en' | 'ru' | null {
  if (import.meta.env.SSR) return null
  const host = window.location.hostname
  // fxtun.dev is consolidated into fxtun.ru — both domains serve the
  // Russian-first site (fxtun.dev 301-redirects to fxtun.ru at the edge).
  if (host === 'fxtun.ru' || host.endsWith('.fxtun.ru')) return 'ru'
  if (host === 'fxtun.dev' || host.endsWith('.fxtun.dev')) return 'ru'
  return null
}

function getDefaultLocale(): 'en' | 'ru' {
  if (import.meta.env.SSR) return 'ru'
  return getDomainLocale()
    ?? (localStorage.getItem('locale') as 'en' | 'ru' | null)
    ?? (['ru', 'uk', 'be'].includes(navigator.language.split('-')[0]) ? 'ru' : 'en')
}

const messages: Record<string, MessageSchema> = { ru }

export const i18n = createI18n<[MessageSchema], 'en' | 'ru'>({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'ru',
  messageCompiler: cspMessageCompiler,
  messages: messages as never,
})

// Load a locale's messages if they are not in memory yet. The router guard
// awaits this before the first render — during prerendering as well as in the
// browser — so an English page is never rendered or hydrated in Russian.
export async function ensureLocale(locale: 'en' | 'ru') {
  if (i18n.global.availableLocales.includes(locale)) return
  const loaded = await import('./en.json')
  i18n.global.setLocaleMessage(locale, loaded.default as never)
}

if (!import.meta.env.SSR) {
  const locale = getDefaultLocale()
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export function getBlogUrl(): string {
  if (import.meta.env.SSR) return '/blog'
  return `${window.location.protocol}//fxtun.ru/blog`
}

export async function setLocale(locale: 'en' | 'ru') {
  await ensureLocale(locale)
  // @ts-expect-error vue-i18n composition api
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export function getLocale(): 'en' | 'ru' {
  // @ts-expect-error vue-i18n composition api
  return i18n.global.locale.value as 'en' | 'ru'
}

export function getBaseDomain(): string {
  return getLocale() === 'ru' ? 'fxtun.ru' : 'fxtun.dev'
}
