import { useHead, useSeoMeta } from '@unhead/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

interface SeoOptions {
  titleKey?: string
  descriptionKey?: string
  title?: string
  description?: string
  path?: string
  image?: string
  type?: 'website' | 'article'
}

export function useSeo(options: SeoOptions = {}) {
  const { t, locale } = useI18n()

  const title = computed(() => options.title || (options.titleKey ? t(options.titleKey) : 'fxTunnel'))
  const description = computed(() => options.description || (options.descriptionKey ? t(options.descriptionKey) : t('seo.defaultDescription')))
  const url = computed(() => `https://fxtun.dev${options.path || ''}`)
  const image = options.image || 'https://fxtun.dev/og-image.png'

  useSeoMeta({
    title,
    description,
    ogTitle: title,
    ogDescription: description,
    ogImage: image,
    ogUrl: url,
    ogType: options.type || 'website',
    ogSiteName: 'fxTunnel',
    twitterCard: 'summary_large_image',
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image,
  })

  useHead({
    htmlAttrs: { lang: locale },
  })
}
