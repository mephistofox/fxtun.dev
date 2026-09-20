/**
 * Run work once the page has finished loading and the browser is idle.
 *
 * The landing page is prerendered: the plans, the downloads and the star count
 * are already in the HTML. Refreshing them at mount put three network requests
 * on the critical path — the GitHub one alone stretched it to 1.2 s — and each
 * answer repainted part of the page while the browser was still deciding when
 * rendering had finished. Same data, fetched a moment later.
 */
export function afterLoad(fn: () => void, timeout = 3000) {
  if (typeof window === 'undefined') return

  const run = () => {
    if ('requestIdleCallback' in window) window.requestIdleCallback(fn, { timeout })
    else setTimeout(fn, 800)
  }

  if (document.readyState === 'complete') run()
  else window.addEventListener('load', run, { once: true })
}
