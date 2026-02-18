import { useHead } from '@unhead/vue'
import { getDomainLocale, getLocale } from '../i18n'

function getBaseUrl(): string {
  // During SSG, use the current i18n locale (set by forcedLocale in router beforeEach).
  // This way /ru route gets fxtun.ru URLs, / route gets fxtun.dev URLs.
  if (import.meta.env.SSR) {
    return getLocale() === 'ru' ? 'https://fxtun.ru' : 'https://fxtun.dev'
  }
  const locale = getDomainLocale()
  return locale === 'ru' ? 'https://fxtun.ru' : 'https://fxtun.dev'
}

export function useOrganizationSchema() {
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: 'ld-organization',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Organization',
          name: 'fxtun',
          url: baseUrl,
          logo: `${baseUrl}/og-image.png`,
          sameAs: [
            'https://github.com/mephistofox/fxtun.dev',
          ],
          description:
            'fxtun is a free ngrok alternative — reverse tunneling service with HTTP, TCP & UDP support, desktop GUI, and no usage limits.',
        }),
      },
    ],
  })
}

export function useSoftwareApplicationSchema() {
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: 'ld-software-application',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'SoftwareApplication',
          name: 'fxtun',
          applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Windows, macOS, Linux',
          description:
            'Free ngrok alternative — reverse tunneling service that exposes localhost to the internet via HTTP, TCP, and UDP. Desktop GUI, CLI, custom subdomains, traffic inspector.',
          url: baseUrl,
          downloadUrl: `${baseUrl}/#download`,
          offers: [
            {
              '@type': 'Offer',
              price: '0',
              priceCurrency: 'USD',
              name: 'Free',
              description: '3 tunnels, any subdomain, no request limits, no session timeout',
            },
            {
              '@type': 'Offer',
              price: '5.00',
              priceCurrency: 'USD',
              name: 'Base',
              priceSpecification: {
                '@type': 'UnitPriceSpecification',
                price: '5.00',
                priceCurrency: 'USD',
                billingDuration: 'P1M',
              },
              description: '5 tunnels, 5 reserved subdomains, 1 custom domain, traffic inspector',
            },
            {
              '@type': 'Offer',
              price: '10.00',
              priceCurrency: 'USD',
              name: 'Pro',
              priceSpecification: {
                '@type': 'UnitPriceSpecification',
                price: '10.00',
                priceCurrency: 'USD',
                billingDuration: 'P1M',
              },
              description: '15 tunnels, 15 reserved subdomains, 5 custom domains, traffic inspector',
            },
          ],
          featureList: [
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
        }),
      },
    ],
  })
}

export function useWebSiteSchema() {
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: 'ld-website',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          name: 'fxtun',
          url: baseUrl,
          description:
            'Free ngrok alternative with no request limits and no session timeout. HTTP, TCP & UDP tunneling with desktop app.',
        }),
      },
    ],
  })
}

export function useWebPageSchema() {
  const baseUrl = getBaseUrl()
  useHead({
    script: [
      {
        id: 'ld-webpage',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'WebPage',
          name: 'fxtun — Free ngrok Alternative',
          url: baseUrl,
          description:
            'Free ngrok alternative with no request limits, no session timeout, and free custom subdomains. HTTP, TCP & UDP tunneling with desktop GUI.',
          speakable: {
            '@type': 'SpeakableSpecification',
            cssSelector: ['.hero-section', '#features', '#faq', '#comparison', '#pricing'],
          },
          mainEntity: {
            '@type': 'SoftwareApplication',
            name: 'fxtun',
          },
        }),
      },
    ],
  })
}

export function useFaqSchema(faqs: Array<{ question: string; answer: string }>) {
  useHead({
    script: [
      {
        id: 'ld-faq',
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'FAQPage',
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
