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
  robots?: string
}

export function useSeo(options: SeoOptions = {}) {
  const { t, locale } = useI18n()
  const route = useRoute()

  // Effective locale: forcedLocale from route meta takes priority.
  // During SSG, default to 'en' for non-prefixed routes (used by fxtun.dev)
  // to prevent state leaks from shared i18n instance across renders.
  const effectiveLocale = (route.meta.forcedLocale as 'en' | 'ru') ??
    (import.meta.env.SSR ? 'en' : locale.value)

  // Always sync i18n state for current route (prevents SSG state leaks
  // where /ru route sets global locale to 'ru' and non-prefixed routes inherit it)
  locale.value = effectiveLocale
  // @ts-expect-error vue-i18n composition api
  i18n.global.locale.value = effectiveLocale

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
  const image = options.image || `https://${ogDomain}/og-image.jpg`

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

  // Show hreflang on all routes — every page needs bidirectional hreflang
  // for correct language targeting between fxtun.ru (ru) and fxtun.dev (en).
  const showHreflang = computed(() => true)

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

  const keywords = computed(() => te('seo.defaultKeywords'))

  useSeoMeta({
    title,
    description,
    keywords,
    ...(options.robots ? { robots: options.robots } : {}),
    ogTitle: title,
    ogDescription: description,
    ogImage: image,
    ogImageWidth: 1200,
    ogImageHeight: 630,
    ogImageType: 'image/jpeg',
    ogImageAlt: title,
    ogUrl: canonical,
    ogType: options.type || 'website',
    ogSiteName: 'fxtun',
    ogLocale: domainLocale === 'ru' ? 'ru_RU' : 'en_US',
    twitterCard: 'summary_large_image',
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image,
  })
}
