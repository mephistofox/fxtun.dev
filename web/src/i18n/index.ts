import { createI18n } from 'vue-i18n'
import en from './en.json'
import ru from './ru.json'

type MessageSchema = typeof en

export function getDomainLocale(): 'en' | 'ru' | null {
  if (import.meta.env.SSR) return null
  const host = window.location.hostname
  if (host === 'fxtun.ru' || host.endsWith('.fxtun.ru')) return 'ru'
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
  messages: {
    en,
    ru,
  },
})

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
