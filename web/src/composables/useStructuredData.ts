import { useHead } from '@unhead/vue'
import { getDomainLocale, getLocale } from '../i18n'

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
    offerFree: '3 tunnels, any subdomain, no request limits, no session timeout',
    offerBase: '5 tunnels, 5 reserved subdomains, 1 custom domain, traffic inspector',
    offerPro: '15 tunnels, 15 reserved subdomains, 5 custom domains, traffic inspector',
    offerBusiness: '50 tunnels, 50 reserved subdomains, 50 custom domains, traffic inspector',
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
    offerFree: '3 туннеля, любой субдомен, без лимитов запросов, без таймаута',
    offerBase: '5 туннелей, 5 зарезервированных субдоменов, 1 свой домен, инспектор трафика',
    offerPro: '15 туннелей, 15 зарезервированных субдоменов, 5 своих доменов, инспектор трафика',
    offerBusiness: '50 туннелей, 50 зарезервированных субдоменов, 50 своих доменов, инспектор трафика',
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

const pricing = {
  en: { currency: 'USD', free: '0', base: '5.00', pro: '10.00', business: '15.00' },
  ru: { currency: 'RUB', free: '0', base: '400', pro: '800', business: '1200' },
} as const

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
          logo: `${baseUrl}/og-image.png`,
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
  const p = pricing[locale]
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
          url: baseUrl,
          downloadUrl: `${baseUrl}/#download`,
          publisher: { '@id': `${baseUrl}/#organization` },
          offers: [
            {
              '@type': 'Offer',
              price: p.free,
              priceCurrency: p.currency,
              name: 'Free',
              description: t.offerFree,
            },
            {
              '@type': 'Offer',
              price: p.base,
              priceCurrency: p.currency,
              name: 'Base',
              priceSpecification: {
                '@type': 'UnitPriceSpecification',
                price: p.base,
                priceCurrency: p.currency,
                billingDuration: 'P1M',
              },
              description: t.offerBase,
            },
            {
              '@type': 'Offer',
              price: p.pro,
              priceCurrency: p.currency,
              name: 'Pro',
              priceSpecification: {
                '@type': 'UnitPriceSpecification',
                price: p.pro,
                priceCurrency: p.currency,
                billingDuration: 'P1M',
              },
              description: t.offerPro,
            },
            {
              '@type': 'Offer',
              price: p.business,
              priceCurrency: p.currency,
              name: 'Business',
              priceSpecification: {
                '@type': 'UnitPriceSpecification',
                price: p.business,
                priceCurrency: p.currency,
                billingDuration: 'P1M',
              },
              description: t.offerBusiness,
            },
          ],
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

export function useFaqSchema(faqs: Array<{ question: string; answer: string }>) {
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: 'ld-faq',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'FAQPage',
          '@id': `${baseUrl}/#faq`,
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
