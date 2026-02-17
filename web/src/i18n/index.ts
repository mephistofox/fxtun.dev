import { createI18n } from 'vue-i18n'
import type { MessageCompiler } from 'vue-i18n'
import en from './en.json'
import ru from './ru.json'

type MessageSchema = typeof en

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
  if (host === 'fxtun.ru' || host.endsWith('.fxtun.ru')) return 'ru'
  if (host === 'fxtun.dev' || host.endsWith('.fxtun.dev')) return 'en'
  return null
}

function getDefaultLocale(): 'en' | 'ru' {
  if (import.meta.env.SSR) return 'en'
  return getDomainLocale()
    ?? (localStorage.getItem('locale') as 'en' | 'ru' | null)
    ?? (['ru', 'uk', 'be'].includes(navigator.language.split('-')[0]) ? 'ru' : 'en')
}

export const i18n = createI18n<[MessageSchema], 'en' | 'ru'>({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'en',
  messageCompiler: cspMessageCompiler,
  messages: {
    en,
    ru,
  },
})

if (!import.meta.env.SSR) {
  const locale = getDefaultLocale()
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export function getBlogUrl(): string {
  if (import.meta.env.SSR) return '/blog'
  const locale = getLocale()
  const domain = locale === 'ru' ? 'fxtun.ru' : 'fxtun.dev'
  return `${window.location.protocol}//${domain}/blog`
}

export function setLocale(locale: 'en' | 'ru') {
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
