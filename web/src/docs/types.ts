// Shape produced by plugins/docs-md.ts (RenderedDoc) for every docs/*.md import.
export interface DocHeading {
  level: 2 | 3
  id: string
  text: string
}

export interface DocPage {
  title: string
  description: string
  html: string
  headings: DocHeading[]
  minutes: number
}

export interface DocsEntry {
  slug: string
  group: string
  nav: string
  lastmod: string
  tag?: 'http' | 'tcp' | 'udp'
}

export const docPath = (slug: string) => (slug ? `/docs/${slug}` : '/docs')
