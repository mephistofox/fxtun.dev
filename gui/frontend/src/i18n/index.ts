import { createI18n } from 'vue-i18n'
import en from './en.json'
import ru from './ru.json'

type SupportedLocale = 'en' | 'ru'

function getDefaultLocale(): SupportedLocale {
  const saved = localStorage.getItem('locale')
  if (saved === 'en' || saved === 'ru') {
    return saved
  }

  const browserLang = navigator.language.split('-')[0]
  // Russian, Ukrainian, Belarusian - default to Russian
  if (['ru', 'uk', 'be'].includes(browserLang)) {
    return 'ru'
  }

  return 'en'
}

export function setLocale(locale: SupportedLocale) {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export function getLocale(): SupportedLocale {
  return i18n.global.locale.value as SupportedLocale
}

export const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'en',
  messages: {
    en,
    ru,
  },
})

export default i18n
