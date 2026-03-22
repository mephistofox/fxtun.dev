import { useHead, useSeoMeta } from '@unhead/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { i18n, getDomainLocale } from '../i18n'

interface SeoOptions {
  titleKey?: string
  descriptionKey?: string
  title?: string
  description?: string
  image?: string
  type?: 'website' | 'article'
}

export function useSeo(options: SeoOptions = {}) {
  const { t, locale } = useI18n()
  const route = useRoute()

  // Effective locale: forcedLocale from route meta takes priority
  const effectiveLocale = (route.meta.forcedLocale as 'en' | 'ru') ?? locale.value

  // Apply forcedLocale globally (for component body rendering)
  if (route.meta.forcedLocale) {
    locale.value = effectiveLocale
    // @ts-expect-error vue-i18n composition api
    i18n.global.locale.value = effectiveLocale
  }

  // Helper: translate with explicit locale (bypasses reactive locale for SSG)
  // @ts-expect-error vue-i18n message schema
  const te = (key: string) => i18n.global.messages.value[effectiveLocale]?.[key.split('.')[0]]
    ? t(key, [], { locale: effectiveLocale })
    : t(key)

  const title = computed(() => options.title || (options.titleKey ? te(options.titleKey) : 'fxtun'))
  const description = computed(() => options.description || (options.descriptionKey ? te(options.descriptionKey) : te('seo.defaultDescription')))
  // During SSG getDomainLocale() returns null (no window), so use effectiveLocale
  // which is set from route meta forcedLocale (e.g. 'ru' for /ru/* routes).
  const domainLocale = import.meta.env.SSR ? effectiveLocale : getDomainLocale()
  const ogDomain = domainLocale === 'ru' ? 'fxtun.ru' : 'fxtun.dev'
  const image = options.image || `https://${ogDomain}/og-image.png`

  const isLangPrefix = computed(() =>
    route.path.startsWith('/ru') || route.path.startsWith('/en')
  )

  const cleanPath = computed(() => {
    if (route.path.startsWith('/ru')) return route.path.slice(3) || '/'
    if (route.path.startsWith('/en')) return route.path.slice(3) || '/'
    return route.path
  })

  const enCanonical = computed(() => `https://fxtun.dev${cleanPath.value}`)
  const ruCanonical = computed(() => `https://fxtun.ru${cleanPath.value}`)

  const canonical = computed(() => {
    if (isLangPrefix.value) {
      return route.path.startsWith('/ru') ? ruCanonical.value : enCanonical.value
    }
    const domain = domainLocale === 'ru' ? 'fxtun.ru' : 'fxtun.dev'
    return `https://${domain}${cleanPath.value}`
  })

  // Show hreflang on non-prefixed routes AND on /ru, /en root landing pages.
  // /ru is served as fxtun.ru root via domain-based pre-rendering, so it needs hreflang.
  const showHreflang = computed(() =>
    !isLangPrefix.value || route.path === '/ru' || route.path === '/en'
  )

  useHead({
    htmlAttrs: { lang: effectiveLocale },
    link: computed(() => [
      { rel: 'canonical', href: canonical.value },
      ...(showHreflang.value ? [
        { rel: 'alternate', hreflang: 'en', href: enCanonical.value },
        { rel: 'alternate', hreflang: 'ru', href: ruCanonical.value },
        { rel: 'alternate', hreflang: 'x-default', href: enCanonical.value },
      ] : []),
    ]),
  })

  useSeoMeta({
    title,
    description,
    ogTitle: title,
    ogDescription: description,
    ogImage: image,
    ogUrl: canonical,
    ogType: options.type || 'website',
    ogSiteName: 'fxtun',
    twitterCard: 'summary_large_image',
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image,
  })
}
