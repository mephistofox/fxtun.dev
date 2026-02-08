import { useHead } from '@unhead/vue'

export function useOrganizationSchema() {
  useHead({
    script: [
      {
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Organization',
          name: 'fxTunnel',
          url: 'https://fxtun.dev',
          logo: 'https://fxtun.dev/logo.png',
        }),
      },
    ],
  })
}

export function useSoftwareApplicationSchema() {
  useHead({
    script: [
      {
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'SoftwareApplication',
          name: 'fxTunnel',
          applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Windows, macOS, Linux',
          description:
            'Secure localhost tunneling service supporting HTTP, TCP, and UDP protocols with desktop GUI client',
          url: 'https://fxtun.dev',
          downloadUrl: 'https://fxtun.dev/#download',
          offers: [
            {
              '@type': 'Offer',
              price: '0',
              priceCurrency: 'USD',
              name: 'Free',
              description: '3 tunnels, any subdomain, no limits',
            },
            {
              '@type': 'Offer',
              price: '5.00',
              priceCurrency: 'USD',
              name: 'Base',
              description: '5 tunnels, 5 reserved subdomains, traffic inspector',
            },
            {
              '@type': 'Offer',
              price: '10.00',
              priceCurrency: 'USD',
              name: 'Pro',
              description: '15 tunnels, 15 reserved subdomains, 5 custom domains',
            },
          ],
          featureList: [
            'HTTP tunneling with custom subdomains',
            'TCP port forwarding',
            'UDP port forwarding',
            'Desktop GUI client',
            'Traffic inspector',
            'Self-hostable',
            'No bandwidth limits',
            'No session timeout',
          ],
        }),
      },
    ],
  })
}

export function useWebSiteSchema() {
  useHead({
    script: [
      {
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          name: 'fxTunnel',
          url: 'https://fxtun.dev',
          description:
            'Secure localhost tunneling service supporting HTTP, TCP, and UDP protocols with desktop GUI client',
        }),
      },
    ],
  })
}

export function useFaqSchema(faqs: Array<{ question: string; answer: string }>) {
  useHead({
    script: [
      {
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
