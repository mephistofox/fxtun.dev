import { createI18n } from 'vue-i18n'
import en from './en.json'
import ru from './ru.json'

type MessageSchema = typeof en

function getDefaultLocale(): 'en' | 'ru' {
  const saved = localStorage.getItem('locale')
  if (saved === 'en' || saved === 'ru') {
    return saved
  }

  const browserLang = navigator.language.split('-')[0]
  if (['ru', 'uk', 'be'].includes(browserLang)) {
    return 'ru'
  }

  return 'en'
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
