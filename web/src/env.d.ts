/// <reference types="vite/client" />
/// <reference types="vite-ssg" />


interface Window {
  dataLayer: unknown[]
  gtag: (...args: unknown[]) => void
}
