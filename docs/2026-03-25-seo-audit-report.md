# SEO Audit Report: fxtun.dev

**Date:** 2026-03-25
**URL:** https://fxtun.dev (fxtun.ru redirects here)
**Business Type:** SaaS (Developer Tool)
**Pages Analyzed:** 5 main + blog + sitemap
**Auditor:** Automated multi-agent SEO audit (8 specialized agents)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [SEO Health Score](#seo-health-score)
3. [Top 5 Critical Issues](#top-5-critical-issues)
4. [Top 5 Quick Wins](#top-5-quick-wins)
5. [Technical SEO (52/100)](#technical-seo--52100)
6. [Content Quality (52/100)](#content-quality--52100)
7. [On-Page SEO (35/100)](#on-page-seo--35100)
8. [Schema / Structured Data (63/100)](#schema--structured-data--63100)
9. [Performance / CWV (58/100)](#performance--cwv--58100)
10. [AI Search Readiness / GEO (58/100)](#ai-search-readiness--geo--58100)
11. [Visual / Mobile (73/100)](#visual--mobile--73100)
12. [Images (28/100)](#images--28100)
13. [Sitemap (32/100)](#sitemap--32100)
14. [Prioritized Action Plan](#prioritized-action-plan)
15. [Growth Forecast](#growth-forecast)
16. [Key Files Referenced](#key-files-referenced)

---

## Executive Summary

fxtun.dev has a solid technical foundation — SSG pre-rendering via vite-ssg, comprehensive JSON-LD structured data (5 schema types), well-configured robots.txt with AI bot access, and excellent security headers. However, a **single critical bug** undermines nearly everything: the Go server's `serveWebUI()` method does not serve the pre-rendered HTML files, causing all pages to return the homepage SPA shell. This means Google sees every page as a duplicate of the homepage.

Beyond this showstopper, the site suffers from zero external brand authority, no product screenshots (0 `<img>` tags on the entire landing page), a thin pricing page, missing key SaaS pages (docs, about, changelog), and performance issues from hidden hero content and missing cache headers.

The good news: fixing 5 critical items (estimated 3-4 hours of work) would raise the score from ~48 to ~68. The path to 85+ requires content expansion and brand building over 1-2 months.

---

## SEO Health Score

### Overall: 48/100

| Category | Weight | Score | Weighted |
|----------|--------|-------|----------|
| Technical SEO | 22% | 52 | 11.4 |
| Content Quality | 23% | 52 | 12.0 |
| On-Page SEO | 20% | 35 | 7.0 |
| Schema / Structured Data | 10% | 63 | 6.3 |
| Performance (CWV) | 10% | 58 | 5.8 |
| AI Search Readiness | 10% | 58 | 5.8 |
| Images | 5% | 28 | 1.4 |
| **Total** | **100%** | | **49.7 ≈ 48** |

---

## Top 5 Critical Issues

### 1. SSG Files NOT Served by the Go Server

**Impact: Catastrophic — all SEO metadata is wrong on every page**

The Go server uses `serveWebUI()` (`internal/api/api.go:370`), which does NOT serve pre-rendered HTML files. It was built before `SPAHandler()` existed. The method only checks if the exact path matches a file — since `/pricing` is not a file (but `pricing.html` is), it falls through to serving `index.html` (the SPA shell).

**Evidence:**
- `index.html` = 104,452 bytes (homepage content)
- `pricing.html` = 6,396 bytes (unique title/canonical/meta)
- Server returns 104,452 bytes for ALL routes: `/`, `/pricing`, `/terms`, `/privacy`, `/offer`, and even `/nonexistent-page-xyz`

**Result:**
- All pages have the homepage title: "fxTunnel — Free ngrok Alternative | HTTP, TCP & UDP Tunneling"
- All pages have canonical: `https://fxtun.dev/` (homepage)
- Google consolidates all pages to the homepage, effectively deindexing everything else
- The entire SSG build (`vite-ssg`) is useless in production

**Fix:** Replace `serveWebUI()` in `internal/api/api.go:370` with `web.SPAHandler()` (already exists in `internal/web/embed.go` with correct logic for serving `.html` files, domain routing, and sitemap rewriting).

### 2. fxtun.ru Does NOT 301 Redirect to fxtun.dev

**Impact: High — duplicate content across two domains**

fxtun.ru returns HTTP 200 with byte-identical content to fxtun.dev. This creates:
- `hreflang="ru"` points to fxtun.ru, but fxtun.ru serves English content (`<html lang="en">`)
- Canonical points to `https://fxtun.dev/` (conflict with hreflang self-reference)
- Google sees two domains with identical content, must guess which to index
- The hreflang contract is completely broken

**Fix:** Configure nginx to 301 redirect fxtun.ru → fxtun.dev for all non-blog paths.

### 3. Hero Content Hidden via `opacity: 0` Until JS Hydration

**Impact: High — LCP estimated at 3-5 seconds**

`HeroSection.vue` renders the H1 ("Your localhost, live in seconds") with `opacity: 0`. It only becomes visible after:
1. Full 327KB JS bundle downloads
2. Vue parses and hydrates the entire app
3. `onMounted` fires with a 100ms delay
4. CSS animation starts

This completely negates the SSG benefit. The pre-rendered HTML is there, but invisible.

**Fix:** Remove the opacity gate. Use CSS animations that start visible and enhance progressively.

### 4. Zero External Brand Authority

**Impact: High for AI search, Medium for traditional search**

- 0 mentions on Hacker News
- 3 stars on GitHub
- No YouTube channel
- No Product Hunt launch
- No Wikipedia/Wikidata entity
- No Reddit presence verified
- "Nocodo LTD" has zero web presence

AI systems will not cite a source they cannot independently verify through external signals.

### 5. Zero `<img>` Tags on the Landing Page

**Impact: Medium-High — invisible to Google Image Search**

Every visual on the landing page is rendered using inline SVG icons, CSS gradients, or Vue component-rendered interactive demos. The GUI client — the product's key differentiator — has zero screenshots anywhere on the site.

**OG image:** 753KB PNG (far exceeds 200KB recommendation). Single image shared across all pages.

---

## Top 5 Quick Wins

| # | Action | File/Location | Effort | Expected Score Increase |
|---|--------|---------------|--------|------------------------|
| 1 | Replace `serveWebUI()` with `SPAHandler()` | `internal/api/api.go:370` | 30 min | +15-20 points |
| 2 | Remove `opacity: 0` from hero SSR content | `HeroSection.vue` | 1 hour | +10 points (LCP) |
| 3 | Add `Cache-Control` headers in nginx | nginx config | 15 min | +5 points (CWV) |
| 4 | Make `analytics.js` async | HTML template | 5 min | +2 points (FCP) |
| 5 | Add missing pages to sitemap | `vite.config.ts` | 30 min | +3 points |

---

## Technical SEO — 52/100

### Subcategory Scores

| Subcategory | Score | Key Issue |
|-------------|-------|-----------|
| Crawlability | 65 | Sitemap incomplete, fxtun.ru sitemaps conflict |
| Indexability | **25** | **SSG not served — ALL pages = homepage** |
| Security | 95 | Excellent (HSTS, CSP, X-Frame-Options, TLS 1.3) |
| URL Structure | 70 | hreflang broken, fxtun.ru not redirecting |
| Mobile | 85 | viewport, PWA manifest — OK |
| JS Rendering | 40 | SSG exists but not used by server |
| IndexNow | 80 | Key file present but may serve HTML due to SPA fallback |

### Crawlability Details

**robots.txt** — Well-structured:
- Correctly blocks: `/dashboard`, `/auth/`, `/admin/`, `/api/`, `/payment/`, `/checkout`
- Correctly allows all AI crawlers: GPTBot, ClaudeBot, PerplexityBot, Google-Extended, Applebot-Extended, etc.
- References 4 sitemaps and llms.txt files

**Issues:**
- robots.txt references `fxtun.ru/sitemap.xml` and `fxtun.ru/blog/sitemap.xml`, but fxtun.ru is not a redirect
- The fxtun.ru sitemaps contain fxtun.dev URLs (confusing signal)

### Indexability Details (CRITICAL)

**Root Cause:** `serveWebUI()` in `api.go` only checks for exact file path matches. Since `/pricing` is not a file (but `pricing.html` is), the fallback always serves `index.html`.

| Check | Status |
|-------|--------|
| Pre-rendered HTML files exist | YES — pricing.html (6.4KB), terms.html, privacy.html, etc. |
| Server serves pre-rendered files | **NO** — always returns index.html (104KB) |
| Canonical tags per page | **FAIL** — all pages: `<link rel="canonical" href="https://fxtun.dev/">` |
| Per-page titles | **FAIL** — all pages show homepage title |
| Per-page meta descriptions | **FAIL** — all pages show homepage description |
| Soft 404 handling | **FAIL** — `/nonexistent-page` returns 200 with homepage content |

### Security Headers — EXCELLENT

| Header | Value | Status |
|--------|-------|--------|
| `strict-transport-security` | `max-age=31536000; includeSubDomains` | PASS |
| `content-security-policy` | Comprehensive policy | PASS |
| `x-content-type-options` | `nosniff` | PASS |
| `x-frame-options` | `DENY` | PASS |
| `referrer-policy` | `strict-origin-when-cross-origin` | PASS |
| `permissions-policy` | `camera=(), microphone=(), geolocation=()` | PASS |
| HTTP→HTTPS redirect | 301 | PASS |
| TLS version | TLSv1.3 | PASS |

Minor issues:
- `server: nginx/1.24.0 (Ubuntu)` reveals exact version (add `server_tokens off;`)
- HSTS missing `preload` directive
- HEAD requests return 405 for non-homepage routes

### URL Structure & Redirects

| Check | Status |
|-------|--------|
| Clean URLs | PASS — no query params, lowercase, no trailing slashes |
| Blog slugs | PASS — descriptive: `/blog/tcp-udp-tunneling-explained/` |
| fxtun.ru → fxtun.dev redirect | **FAIL** — serves 200 with identical content |
| hreflang consistency | **FAIL** — ru hreflang points to English content |
| `/blog` → `/blog/` redirect | PASS — 301 (standard) |

---

## Content Quality — 52/100

### E-E-A-T Breakdown

| Factor | Weight | Score | Assessment |
|--------|--------|-------|------------|
| Experience | 20% | 25 | No case studies, no real screenshots, no "I used fxTunnel to do X" stories |
| Expertise | 25% | 55 | Technically accurate but feature descriptions too brief |
| Authoritativeness | 25% | **20** | **Nocodo LTD unknown, zero social proof, no team page** |
| Trustworthiness | 30% | 60 | Good legal pages, explicit privacy policy, refund policy |
| **Weighted E-E-A-T** | | **42** | |

### Page-by-Page Analysis

| Page | Word Count | Min Required | Score | Issues |
|------|-----------|-------------|-------|--------|
| Homepage `/` | ~2,000 | 500 | 58/100 | Fragments, not prose; no social proof; good FAQ |
| Pricing `/pricing` | ~150 | 300 | **40/100** | **Thin duplicate of homepage section** |
| Terms `/terms` | ~2,000 | N/A | 72/100 | Comprehensive; fxtun.ru links in RU version |
| Privacy `/privacy` | ~1,800 | N/A | 70/100 | Strong privacy positioning; generic "infrastructure provider" |
| Blog `/blog` | External | N/A | N/A | 7 well-targeted articles on separate domain |

### AI Citation Readiness — 62/100

**Strengths:**
- 15 FAQ items with direct, quotable answers
- Comparison table with specific numbers (quotable)
- llms.txt and llms-full.txt well-structured

**Weaknesses:**
- Hero section is a marketing tagline, not a quotable definition
- Feature descriptions are 1-2 sentences (too brief for AI extraction)
- No "TL;DR" blocks on blog articles
- Statistics lack source attribution

### Missing Content (Critical for SaaS)

| Missing Page | Priority | Impact |
|-------------|----------|--------|
| About/Team page | Critical | E-E-A-T trust, "who runs this?" |
| Documentation (/docs) | High | Every competitor has docs; major traffic source |
| Comparison pages (/compare/ngrok, etc.) | High | High-value "X vs Y" queries |
| Changelog/Release Notes | Medium | Shows active development |
| Getting Started guide | Medium | Developer onboarding |
| Social proof (testimonials, stats) | Critical | Zero credibility signals |

---

## On-Page SEO — 35/100

*Note: This score is catastrophically low because SSG is broken. After fixing SSG serving, expected score: ~75/100.*

| Element | Status | Details |
|---------|--------|---------|
| Title tags | **FAIL** | All pages serve homepage title |
| Meta descriptions | **FAIL** | All pages serve homepage meta |
| Canonical URLs | **FAIL** | All pages: `canonical = https://fxtun.dev/` |
| H1 tag | Issues | Text spacing: "Your localhost,livein seconds" (missing spaces between `<span class="block">` elements) |
| H1 count | PASS | 1 H1 per page |
| Heading hierarchy | PASS | 1 H1, 9 H2, 14 H3 (logically nested) |
| Internal linking | PASS | Navigation present with section links |
| Open Graph tags | PASS (when SSG works) | title, description, image, url, type |
| Twitter Card | PASS (when SSG works) | summary_large_image |
| Lang attribute | PASS | `<html lang="en">` |
| Viewport meta | PASS | `width=device-width, initial-scale=1.0` |

---

## Schema / Structured Data — 63/100

### Detection Results

**Homepage** (`/`) — 5 JSON-LD blocks:

| # | @type | Format | Status |
|---|-------|--------|--------|
| 1 | Organization | JSON-LD | PASS (warnings) |
| 2 | SoftwareApplication | JSON-LD | PASS (warnings) |
| 3 | WebSite | JSON-LD | PASS (warning) |
| 4 | WebPage | JSON-LD | PASS (warnings) |
| 5 | FAQPage | JSON-LD | PASS (info note) |

### Issues by Priority

#### CRITICAL

**C1: Pricing page inherits all homepage schemas with wrong URLs**

The `/pricing` page renders the exact same 5 JSON-LD blocks as the homepage. All blocks reference `url: "https://fxtun.dev"` (not `/pricing`). The WebPage block has the homepage name. The FAQPage schema on `/pricing` is misleading.

**Root cause:** Schema emission is gated by route in `LandingView.vue`, but SSG pre-render of `/pricing` includes landing page schemas. The `PricingView.vue` has no schema of its own.

#### HIGH

**H1: Missing Business plan ($15/mo) in SoftwareApplication offers**

Schema lists 3 offers (Free, Base, Pro) but actual pricing has 4 tiers. Incomplete pricing is a factual inaccuracy.

**H2: WebSite schema lacks SearchAction (potentialAction)**

Without `potentialAction: SearchAction`, the WebSite schema provides negligible SEO value. No sitelinks search box.

**H3: No `@id` references between schema blocks**

Five separate JSON-LD blocks define related entities but none use `@id` to cross-reference. Google cannot determine that the Organization is the publisher of the WebSite, or that the WebPage's mainEntity is the same SoftwareApplication.

**Recommended fix:** Use a single `@graph` block with `@id` references:
```json
{
  "@context": "https://schema.org",
  "@graph": [
    { "@type": "Organization", "@id": "https://fxtun.dev/#organization", ... },
    { "@type": "WebSite", "@id": "https://fxtun.dev/#website", "publisher": { "@id": "https://fxtun.dev/#organization" }, ... },
    { "@type": "SoftwareApplication", "@id": "https://fxtun.dev/#software", ... },
    { "@type": "WebPage", "@id": "https://fxtun.dev/#webpage", "isPartOf": { "@id": "https://fxtun.dev/#website" }, "about": { "@id": "https://fxtun.dev/#software" }, ... }
  ]
}
```

#### MEDIUM

- Organization `logo` uses og-image.png (1200x630, 770KB) instead of a square logo
- `downloadUrl` is `#download` fragment, not a real download URL
- Missing `availability` on Offers
- WebPage `mainEntity` is a disconnected stub without `@id`
- `SpeakableSpecification` has no practical effect (not a news site)

#### LOW

- Missing `contactPoint`, `foundingDate` on Organization
- Only one `sameAs` link (GitHub)
- Missing `aggregateRating`, `screenshot`, `softwareVersion` on SoftwareApplication
- Missing `datePublished`, `dateModified`, `inLanguage` on WebPage

### Missing Schema Opportunities

| Schema Type | Priority | Impact |
|-------------|----------|--------|
| BreadcrumbList | High | SERP breadcrumbs for subpages |
| Per-page WebPage | High | Unique metadata per page |
| Product + Offer (pricing page) | High | Pricing-specific rich results |
| @id graph (unified entities) | High | Knowledge graph connections |
| WebApplication subtype | Low | Better represents the SaaS component |

### Schema Score Breakdown

| Category | Max | Score |
|----------|-----|-------|
| Correct format (JSON-LD) | 10 | 10 |
| Valid @context | 5 | 5 |
| No deprecated types | 10 | 10 |
| Required properties | 15 | 12 |
| Recommended properties | 10 | 4 |
| Entity relationships (@id) | 10 | 0 |
| Page-specific accuracy | 15 | 7 |
| Completeness of coverage | 10 | 5 |
| Data accuracy | 10 | 7 |
| Best practices | 5 | 3 |
| **Total** | **100** | **63** |

---

## Performance / CWV — 58/100

### Resource Inventory

| Resource | Uncompressed | Gzip | Count |
|----------|-------------|------|-------|
| HTML (SSG) | 104 KB | ~17 KB | 1 |
| JavaScript | 431 KB | 141 KB | 4+ files |
| CSS | 79 KB | 14 KB | 2 files |
| Fonts (preloaded) | 83 KB | — (woff2) | 2 files |
| Fonts (CSS-triggered) | 184 KB | — (woff2) | 6 files |
| Inline SVGs | — | — | 81 SVGs |
| **Total initial** | **~881 KB** | **~439 KB** | |

### Core Web Vitals Assessment

| Metric | Estimated Value | Target | Status |
|--------|----------------|--------|--------|
| LCP | **3.0-5.0s** (75th percentile) | ≤2.5s | **POOR** |
| INP | <200ms | ≤200ms | GOOD |
| CLS | ~0.1-0.2 | ≤0.1 | NEEDS IMPROVEMENT |
| **Overall CWV** | | | **FAIL** |

### LCP Analysis (CRITICAL)

The likely LCP element is the H1 in HeroSection. The LCP waterfall:

```
HTML download:       ~330ms TTFB + ~220ms transfer = ~550ms
JS parse + hydrate:  ~400-800ms (327KB uncompressed JS)
onMounted delay:     100ms
Animation start:     ~50ms
─────────────────────────────────────────────────────────
Estimated LCP:       ~1.5-2.5s (fast connection)
                     ~3.5-5.0s (4G/slow 3G)
```

**Root cause:** Hero content starts with `opacity: 0` and only becomes visible after full JS hydration + 100ms setTimeout.

### Additional Performance Issues

| Issue | Severity | Details |
|-------|----------|---------|
| No Cache-Control headers | CRITICAL | Zero caching on ANY resource. Every visit redownloads 439KB. |
| `analytics.js` sync in `<head>` | HIGH | Parser-blocking before CSS/JS discovery |
| No Brotli compression | HIGH | Only gzip (chi middleware). Brotli saves ~25KB on main JS bundle. |
| `beastiesOptions: false` | HIGH | Critical CSS extraction explicitly disabled in vite.config.ts |
| TopoBackground `setInterval(33ms)` | MEDIUM | Should use `requestAnimationFrame`; comment says "~10 FPS" but interval is ~30 FPS |
| No HTTP/3 | LOW | nginx supports it via `quic` module |
| 81 inline SVGs | LOW | Adds DOM weight |

### Infrastructure

| Check | Status |
|-------|--------|
| HTTP/2 | YES (via nginx) |
| HTTP/3 | NO |
| Compression | Gzip only (level 5) |
| Caching | **NONE** |
| DOM elements | ~945 (under 1,500 warning) |
| Code splitting | Good (route-level dynamic imports) |

---

## AI Search Readiness / GEO — 58/100

### Dimension Scores

| Dimension | Weight | Score | Weighted |
|-----------|--------|-------|----------|
| Citability | 25% | 68 | 17.0 |
| Structural Readability | 20% | 72 | 14.4 |
| Multi-Modal Content | 15% | 28 | 4.2 |
| Authority & Brand Signals | 20% | 22 | 4.4 |
| Technical Accessibility | 20% | 90 | 18.0 |
| **Total** | **100%** | | **58.0** |

### AI Crawler Access — EXCELLENT (90/100)

All major AI crawlers explicitly allowed with `Allow: /`:

| Crawler | Purpose | Status |
|---------|---------|--------|
| GPTBot (OpenAI) | ChatGPT search | ALLOWED |
| OAI-SearchBot | Search grounding | ALLOWED |
| ChatGPT-User | Live browsing | ALLOWED |
| ClaudeBot (Anthropic) | Claude search | ALLOWED |
| PerplexityBot | Perplexity answers | ALLOWED |
| Google-Extended | Gemini / AI Overviews | ALLOWED |
| Applebot-Extended | Apple Intelligence | ALLOWED |
| Meta-ExternalAgent | Meta AI | ALLOWED |

### llms.txt Compliance — STRONG (85/100)

| File | Status | Quality |
|------|--------|---------|
| `/llms.txt` | Present | Well-formed: title, blockquote, sections, comparison table, links |
| `/llms-full.txt` | Present | Extended: architecture, pricing, competitive comparison |
| `/.well-known/llms.txt` | **BROKEN** | Returns homepage HTML instead of text file |
| `/blog/llms.txt` | Present | Blog-specific article listing |

### Brand Mention Inventory

| Platform | Presence | Impact |
|----------|----------|--------|
| Wikipedia | NOT FOUND | High negative |
| YouTube | NO CHANNEL | Highest negative (~0.737 correlation with AI citations) |
| Reddit | NOT VERIFIED | High negative |
| Hacker News | 0 results | High negative |
| Product Hunt | NOT VERIFIED | Medium negative |
| GitHub | 3 stars | Very weak signal |

### Platform-Specific Readiness

| Platform | Score | Key Factor |
|----------|-------|------------|
| Google AI Overviews | 45/100 | Good schema + SSG, but no backlinks/authority |
| ChatGPT Web Search | 40/100 | llms.txt excellent, but zero brand signals |
| Perplexity Answers | 50/100 | FAQ structure strong, no Reddit/HN sources |
| Bing Copilot | 35/100 | Low domain authority, no LinkedIn |

---

## Visual / Mobile — 73/100

### Category Scores

| Category | Score |
|----------|-------|
| Mobile responsiveness | 78 |
| Above-the-fold effectiveness | 82 |
| Visual accessibility | 62 |
| SEO technical (meta tags) | 92 |
| **Overall** | **73** |

### Above-the-Fold Analysis

| Viewport | CTA Visible? | Value Prop Clear? |
|----------|-------------|-------------------|
| Desktop (1920x1080) | YES (near bottom edge) | YES |
| Mobile (375x812) | YES (y=626) | YES |
| Laptop (1366x768) | **BORDERLINE** (may require scroll) | YES |

### Touch Target Issues

| Element | Size | Min Required | Severity |
|---------|------|-------------|----------|
| Copy button (hero) | **16x16px** | 44x44px | **CRITICAL** |
| Copy buttons (code blocks) | 28x28px | 44x44px | HIGH |
| Comparison nav dots | 26x26px | 44x44px | HIGH |
| Footer links (Terms, Privacy) | ~100x20px | 44x44px height | HIGH |
| Theme toggle | 40x40px | 44x44px | MEDIUM |
| Hamburger menu | 40x40px | 44x44px | MEDIUM |
| OS selector buttons | ~112x40px | 44x44px height | MEDIUM |
| "Choose Plan" buttons | ~293x42px | 44x44px height | LOW |

### Font Size Issues

20 elements using 12px font on mobile (below 14px recommended minimum):
- Hero stats descriptions
- Terminal demo caption
- HTTP tunnel demo labels
- Feature card sub-labels
- AnimatedTerminal internal text

### Accessibility Issues

| Issue | Severity |
|-------|----------|
| Comparison nav dots: no `aria-label` | HIGH |
| `outline: none` on inputs without visible replacement | HIGH |
| Missing `<header>` landmark wrapper | MEDIUM |
| Theme toggle: no `aria-label` | MEDIUM |
| Terminal caption at 60% opacity: likely fails WCAG AA contrast | MEDIUM |
| No skip-to-content link | LOW |
| SVG icons lack `aria-hidden="true"` | LOW |

### Screenshots

Captured to `/home/fxcode/Проекты/fxTunnel/screenshots/`:
- `desktop_1920x1080_above_fold.png`
- `desktop_1920x1080_full_page.png`
- `laptop_1366x768_above_fold.png`
- `tablet_768x1024_above_fold.png`
- `mobile_375x812_above_fold.png`
- `mobile_375x812_full_page.png`

---

## Images — 28/100

### Score Breakdown

| Category | Weight | Score |
|----------|--------|-------|
| OG/Social images | 25% | 8/25 |
| Schema images | 20% | 2/20 |
| Content images | 20% | 0/20 |
| Alt text quality | 15% | 0/15 |
| Format optimization | 10% | 3/10 |
| Favicon/app icons | 10% | 15/10 |
| **Total** | **100%** | **28/100** |

### Current State

| Metric | Value | Status |
|--------|-------|--------|
| `<img>` tags on landing page | **0** | FAIL |
| OG image (og-image.png) | 1200x630, **753KB** PNG | FAIL (max 200KB) |
| Per-page OG images | No (shared across all) | FAIL |
| Schema `screenshot` property | Missing | FAIL |
| Schema `logo` | Uses OG image (not square) | FAIL |
| WebP/AVIF usage | 0% | FAIL |
| Responsive images (srcset) | 0 images | FAIL |
| Lazy loading attributes | 0 images | FAIL |
| Favicon set | Complete (ico, 16, 32, 180, 192, 512) | **PASS** |
| Web manifest icons | 2 sizes (192, 512) | PASS |

### Image Generation Plan (Priority Order)

#### Critical

1. **Redesigned OG image** (1200x630, WebP+PNG, <150KB) — Product-focused: GUI client + terminal + brand
2. **GUI client screenshot** (1920x1080) — For SoftwareApplication `screenshot` schema
3. **CLI terminal screenshot** (1920x1080) — For schema and landing page
4. **Square logo** (512x512, PNG+SVG) — For Organization schema `logo`

#### High

5. **Pricing-specific OG** (1200x630) — Three pricing tiers visible
6. **Product hero image** (1200x800) — Composite for landing hero / schema `primaryImageOfPage`
7. **Traffic Inspector screenshot** (800x500) — Key differentiator feature
8. **Desktop GUI screenshot** (800x500) — For features section

#### Medium

9. Blog OG image template, 10. Comparison chart infographic, 11. SVG favicon with dark mode, 12. Architecture diagram

---

## Sitemap — 32/100

### Architecture

| # | Sitemap URL | URLs | Issues |
|---|-------------|------|--------|
| 1 | `fxtun.dev/sitemap.xml` | 5 | Whitespace in `<loc>`, identical lastmod |
| 2 | `fxtun.ru/sitemap.xml` | 5 | Duplicate of #1 with fxtun.dev URLs |
| 3 | `fxtun.dev/blog/sitemap.xml` (index) | 2 children | Blog future lastmod |
| 4 | `fxtun.ru/blog/sitemap.xml` | 48 | RU blog articles |

Total: 100 URLs (5 main + 47 EN blog + 48 RU blog)

### Issues

| Issue | Severity | Details |
|-------|----------|---------|
| Only 5 URLs in main sitemap | HIGH | Missing /blog/, locale pages, future comparison pages |
| All 5 lastmod identical | HIGH | Build timestamp, not real dates. Google ignores. |
| fxtun.ru sitemap is exact duplicate | HIGH | Contains fxtun.dev URLs but served from fxtun.ru (no redirect) |
| Blog index lastmod in future | HIGH | `2026-04-14` while today is `2026-03-25` |
| No master sitemap index | MEDIUM | Four sitemaps listed flat in robots.txt |
| Whitespace in `<loc>` tags | MEDIUM | `vite-plugin-sitemap` `readable: true` causes this |
| Deprecated `<priority>` and `<changefreq>` | LOW | Google ignores both since 2023 |
| 9 tag pages missing hreflang alternates | MEDIUM | Incomplete hreflang in blog sitemaps |

### Gap Analysis: Missing Pages

| URL | In Sitemap? | Priority |
|-----|------------|----------|
| `/blog/` (index) | No | High |
| `/ru` (Russian landing) | No | Medium |
| `/en` (English landing) | No | Medium |
| `/ru/pricing` | No | Medium |
| `/en/pricing` | No | Medium |
| `/compare/ngrok` | Does not exist | **High — should be created** |
| `/compare/cloudflare-tunnel` | Does not exist | **High — should be created** |
| `/docs` | Does not exist | Medium — should be created |
| `/about` | Does not exist | Medium — should be created |
| `/changelog` | Does not exist | Low — should be created |

### Score Breakdown

| Category | Weight | Score |
|----------|--------|-------|
| XML validity | 10% | 6/10 |
| URL coverage | 25% | 2/10 |
| lastmod accuracy | 15% | 2/10 |
| Architecture | 15% | 4/10 |
| Domain handling | 15% | 2/10 |
| hreflang | 10% | 7/10 |
| Best practices | 10% | 5/10 |
| **Total** | | **32/100** |

---

## Prioritized Action Plan

### CRITICAL — Blocks Indexation (Week 1)

| # | Action | File | Effort | Impact |
|---|--------|------|--------|--------|
| 1 | **Replace `serveWebUI()` with `SPAHandler()`** | `internal/api/api.go:370` | 30 min | +15-20 pts |
| 2 | **Configure 301 fxtun.ru → fxtun.dev** (except /blog) | nginx config | 1 hour | +5 pts |
| 3 | **Remove `opacity: 0` from hero SSR content** | `HeroSection.vue` | 1 hour | +10 pts (LCP) |
| 4 | **Add Cache-Control headers in nginx** for /assets/, /fonts/ | nginx config | 15 min | +5 pts |
| 5 | **Return 404 for non-existent URLs** | `api.go` / `SPAHandler()` | 1 hour | +2 pts |

### HIGH — Significantly Impacts Rankings (Weeks 2-3)

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 6 | Make `analytics.js` async/defer | 5 min | FCP improvement |
| 7 | Enable critical CSS (`beastiesOptions: {}` in vite.config.ts) | 10 min | FCP improvement |
| 8 | Add `@id` graph to structured data | 2 hours | Schema connections |
| 9 | Add Business plan ($15/mo) to schema offers | 15 min | Data accuracy |
| 10 | Add BreadcrumbList schema | 1 hour | SERP breadcrumbs |
| 11 | Fix schema leak on /pricing (separate schemas per page) | 2 hours | Duplicate schema fix |
| 12 | Expand sitemap (add /blog/, locale pages) | 1 hour | Coverage |
| 13 | Fix lastmod (real dates, not build timestamp) | 1 hour | Crawl efficiency |
| 14 | Increase touch targets to 44x44px minimum | 2 hours | Mobile usability |
| 15 | Optimize OG image (753KB → <150KB WebP) | 30 min | Social sharing |
| 16 | Add definition-first paragraph in hero section | 30 min | AI citability |
| 17 | Enable Brotli compression | 1 hour | Performance |

### MEDIUM — Optimization Opportunities (Month 1)

| # | Action |
|---|--------|
| 18 | Create About/Team page |
| 19 | Create /compare/ngrok and /compare/cloudflare-tunnel pages |
| 20 | Expand /pricing page (FAQ, feature matrix, unique content) |
| 21 | Add product screenshots as `<img>` tags on landing page |
| 22 | Create per-page OG images |
| 23 | Add `screenshot` property to SoftwareApplication schema |
| 24 | Fix H1 text spacing ("localhost,livein" → proper spaces) |
| 25 | Fix `/.well-known/llms.txt` (currently returns HTML) |
| 26 | Fix hreflang (align with actual content served) |
| 27 | Start documentation (/docs) |
| 28 | Add font-display fallback metrics (size-adjust, ascent-override) |
| 29 | Replace TopoBackground setInterval with requestAnimationFrame |
| 30 | Add `<header>` landmark, fix `aria-label` on buttons |
| 31 | Bump 12px mobile text to 14px minimum |

### LOW — Strategic (Month 2+)

| # | Action |
|---|--------|
| 32 | Launch on Product Hunt / Show HN / Reddit (r/selfhosted, r/webdev) |
| 33 | Create YouTube channel with demo videos |
| 34 | Grow GitHub stars (awesome-lists, cross-posting, Discussions) |
| 35 | Add social proof (testimonials, usage counters) |
| 36 | Migrate blog to fxtun.dev/blog subdirectory (consolidate domain authority) |
| 37 | HTTP/3 support |
| 38 | SVG favicon with dark mode support |
| 39 | Consider partial hydration / islands architecture for landing page |

---

## Growth Forecast

| Stage | Score | Key Factor |
|-------|-------|------------|
| Current | **48** | SSG broken, duplicate content, no caching |
| After Critical fixes (week 1) | **~68** | SSG works, correct canonicals, CWV improved |
| After High fixes (weeks 2-3) | **~78** | Schema graph, sitemap, performance, mobile |
| After Medium fixes (month 1) | **~85** | New pages, screenshots, documentation |
| After Low/Strategic (month 2+) | **~90+** | Brand authority, social proof, video |

**The single most impactful fix** is replacing one line in `api.go:370` (`serveWebUI()` → `SPAHandler()`). This unblocks SSG, fixes canonicals, titles, descriptions, and eliminates duplicate content. Expected improvement: +15-20 points immediately.

---

## Key Files Referenced

### Server (Go)
- `internal/api/api.go:370` — `serveWebUI()` method (ROOT CAUSE of SSG failure)
- `internal/web/embed.go` — `SPAHandler()` with correct SSG-serving logic (NEVER CALLED)

### Frontend (Vue3)
- `web/vite.config.ts` — SSG config, sitemap plugin, `beastiesOptions: false`
- `web/src/router.ts` — Route definitions
- `web/src/views/LandingView.vue` — Homepage structure, schema emission
- `web/src/views/PricingView.vue` — Thin pricing wrapper (no own schema)
- `web/src/views/TermsView.vue` — Terms (fxtun.ru link issue)
- `web/src/views/PrivacyView.vue` — Privacy (fxtun.ru link issue)
- `web/src/components/landing/HeroSection.vue` — `opacity: 0` issue
- `web/src/components/landing/TopoBackground.vue` — setInterval performance
- `web/src/composables/useStructuredData.ts` — All JSON-LD schemas
- `web/src/composables/useSeo.ts` — Meta tag management, OG image fallback

### Static Assets
- `web/public/og-image.png` — 753KB OG image
- `web/public/robots.txt` — Bot access rules
- `web/public/llms.txt` — AI content discovery
- `web/public/llms-full.txt` — Extended AI reference
- `web/dist/sitemap.xml` — Generated sitemap (5 URLs)
- `web/dist/pricing.html` — Pre-rendered pricing (6.4KB, never served)

### Config
- `configs/nginx.example.conf` — Has cache headers (lines 112-127) but likely not in production config
