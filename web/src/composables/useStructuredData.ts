import { useHead } from '@unhead/vue'
import { getDomainLocale, getLocale } from '../i18n'
import plansCache from '@/data/plans-cache.json'

function getEffectiveLocale(): 'en' | 'ru' {
  if (import.meta.env.SSR) return getLocale()
  return getDomainLocale() ?? getLocale()
}

function getBaseUrl(): string {
  return getEffectiveLocale() === 'ru' ? 'https://fxtun.ru' : 'https://fxtun.dev'
}

const descriptions = {
  en: {
    organization: 'fxTunnel is a free ngrok alternative — reverse tunneling service with HTTP, TCP & UDP support, desktop GUI, and no usage limits.',
    software: 'Free ngrok alternative — reverse tunneling service that exposes localhost to the internet via HTTP, TCP, and UDP. Desktop GUI, CLI, custom subdomains, traffic inspector.',
    website: 'Free ngrok alternative with no request limits and no session timeout. HTTP, TCP & UDP tunneling with desktop app.',
    webpage: 'Free ngrok alternative with no request limits, no session timeout, and free custom subdomains. HTTP, TCP & UDP tunneling with desktop GUI.',
    webpageName: 'fxTunnel — Free ngrok Alternative',
    features: [
      'HTTP tunneling with custom subdomains',
      'TCP port forwarding',
      'UDP port forwarding',
      'Desktop GUI client',
      'Traffic inspector',
      'No request or bandwidth limits',
      'No session timeout',
      'Self-hostable',
      'Custom domain support',
      'Automatic HTTPS',
    ],
  },
  ru: {
    organization: 'fxTunnel — бесплатная альтернатива ngrok. Сервис обратного туннелирования с поддержкой HTTP, TCP и UDP, десктопным GUI и без лимитов.',
    software: 'Бесплатная альтернатива ngrok — сервис обратного туннелирования для доступа к localhost через интернет по HTTP, TCP и UDP. Десктопное приложение, CLI, субдомены, инспектор трафика.',
    website: 'Бесплатная альтернатива ngrok без лимитов запросов и таймаута сессий. HTTP, TCP и UDP туннели с десктопным приложением.',
    webpage: 'Бесплатная альтернатива ngrok без лимитов запросов, таймаута сессий и с бесплатными субдоменами. HTTP, TCP и UDP туннели с десктопным GUI.',
    webpageName: 'fxTunnel — Бесплатная альтернатива ngrok',
    features: [
      'HTTP-туннели с субдоменами',
      'Проброс TCP-портов',
      'Проброс UDP-портов',
      'Десктопное GUI-приложение',
      'Инспектор трафика',
      'Без лимитов запросов и трафика',
      'Без таймаута сессий',
      'Можно развернуть на своём сервере',
      'Поддержка своих доменов',
      'Автоматический HTTPS',
    ],
  },
} as const

// Build schema offers from pre-fetched plans cache
function buildSchemaOffers(locale: 'en' | 'ru') {
  const currency = locale === 'ru' ? 'RUB' : 'USD'
  return plansCache.plans.map((plan) => {
    const price = locale === 'ru' ? String(plan.price_rub) : String(plan.price)
    const offer: Record<string, unknown> = {
      '@type': 'Offer',
      price,
      priceCurrency: currency,
      name: plan.name,
    }
    if (plan.price > 0) {
      offer.priceSpecification = {
        '@type': 'UnitPriceSpecification',
        price,
        priceCurrency: currency,
        billingDuration: 'P1M',
      }
    }
    return offer
  })
}

export function useOrganizationSchema() {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  const t = descriptions[locale]
  useHead({
    script: [
      {
        id: 'ld-organization',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Organization',
          '@id': `${baseUrl}/#organization`,
          name: 'fxTunnel',
          url: baseUrl,
          logo: `${baseUrl}/og-image.jpg`,
          sameAs: [
            'https://github.com/mephistofox/fxtun.dev',
          ],
          description: t.organization,
        }),
      },
    ],
  })
}

export function useSoftwareApplicationSchema() {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  const t = descriptions[locale]
  useHead({
    script: [
      {
        id: 'ld-software-application',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'SoftwareApplication',
          '@id': `${baseUrl}/#software`,
          name: 'fxTunnel',
          applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Windows, macOS, Linux',
          description: t.software,
          inLanguage: locale,
          url: baseUrl,
          downloadUrl: `${baseUrl}/downloads`,
          publisher: { '@id': `${baseUrl}/#organization` },
          offers: buildSchemaOffers(locale),
          featureList: t.features,
        }),
      },
    ],
  })
}

export function useWebSiteSchema() {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  const t = descriptions[locale]
  useHead({
    script: [
      {
        id: 'ld-website',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          '@id': `${baseUrl}/#website`,
          name: 'fxTunnel',
          url: baseUrl,
          description: t.website,
          inLanguage: locale,
          publisher: { '@id': `${baseUrl}/#organization` },
        }),
      },
    ],
  })
}

export function useWebPageSchema() {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  const t = descriptions[locale]
  useHead({
    script: [
      {
        id: 'ld-webpage',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'WebPage',
          '@id': `${baseUrl}/#webpage`,
          name: t.webpageName,
          url: baseUrl,
          description: t.webpage,
          inLanguage: locale,
          isPartOf: { '@id': `${baseUrl}/#website` },
          about: { '@id': `${baseUrl}/#software` },
          speakable: {
            '@type': 'SpeakableSpecification',
            cssSelector: ['.hero-section', '#features', '#faq', '#comparison', '#pricing'],
          },
          mainEntity: { '@id': `${baseUrl}/#software` },
        }),
      },
    ],
  })
}

export function useFaqSchema(faqs: Array<{ question: string; answer: string }>, idSuffix = '') {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: `ld-faq${idSuffix}`,
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'FAQPage',
          '@id': `${baseUrl}/#faq${idSuffix}`,
          inLanguage: locale,
          mainEntity: faqs.map((faq) => ({
            '@type': 'Question',
            name: faq.question,
            acceptedAnswer: {
              '@type': 'Answer',
              text: faq.answer,
            },
          })),
        }),
      },
    ],
  })
}

export function useSubpageSchema(options: {
  path: string
  name: string
  description: string
  pageType?: 'WebPage' | 'AboutPage'
  breadcrumbs?: Array<{ name: string; path: string }>
}) {
  const locale = getEffectiveLocale()
  const baseUrl = getBaseUrl()
  const pageUrl = `${baseUrl}${options.path}`

  const schemas: Array<{ id: string; type: string; innerHTML: string }> = []

  // WebPage / AboutPage
  schemas.push({
    id: 'ld-subpage',
    type: 'application/ld+json',
    innerHTML: JSON.stringify({
      '@context': 'https://schema.org',
      '@type': options.pageType || 'WebPage',
      '@id': `${pageUrl}/#webpage`,
      name: options.name,
      url: pageUrl,
      description: options.description,
      inLanguage: locale,
      isPartOf: { '@id': `${baseUrl}/#website` },
    }),
  })

  // BreadcrumbList
  if (options.breadcrumbs && options.breadcrumbs.length > 0) {
    const items = [
      { name: 'fxTunnel', path: '/' },
      ...options.breadcrumbs,
    ]
    schemas.push({
      id: 'ld-breadcrumb',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: items.map((item, index) => ({
          '@type': 'ListItem',
          position: index + 1,
          name: item.name,
          item: `${baseUrl}${item.path}`,
        })),
      }),
    })
  }

  useHead({ script: schemas })
}
