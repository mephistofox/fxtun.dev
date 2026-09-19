import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Sitemap from "vite-plugin-sitemap";
import { ViteMinifyPlugin } from "vite-plugin-minify";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    ViteMinifyPlugin({
      collapseWhitespace: true,
      removeComments: true,
      removeRedundantAttributes: true,
      removeScriptTypeAttributes: true,
      removeStyleLinkTypeAttributes: true,
      minifyCSS: true,
      minifyJS: true,
    }),
    Sitemap({
      hostname: "https://fxtun.ru",
      dynamicRoutes: ["/pricing", "/offer", "/terms", "/privacy", "/about", "/downloads", "/abuse", "/aup", "/disclaimer", "/ngrok-alternative", "/features", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"],
      exclude: ["/docs/offer", "/ru", "/ru/*", "/en", "/en/*", "/login", "/register"],
      generateRobotsTxt: false,
      robots: [{ userAgent: "*", allow: "/" }],
      changefreq: {
        "/": "weekly",
        "/pricing": "weekly",
        "/ngrok-alternative": "monthly",
        "/features": "monthly",
        "/compare/*": "monthly",
        "/about": "monthly",
        "/downloads": "monthly",
        "/offer": "yearly",
        "/terms": "yearly",
        "/privacy": "yearly",
        "/abuse": "yearly",
        "/aup": "yearly",
        "/disclaimer": "yearly",
      },
      priority: {
        "/": 1.0,
        "/pricing": 0.9,
        "/ngrok-alternative": 0.8,
        "/features": 0.8,
        "/compare/*": 0.8,
        "/about": 0.7,
        "/downloads": 0.8,
        "/offer": 0.3,
        "/terms": 0.3,
        "/privacy": 0.3,
        "/abuse": 0.2,
        "/aup": 0.2,
        "/disclaimer": 0.2,
      },
      lastmod: {
        "/": new Date("2026-03-28"),
        "/pricing": new Date("2026-03-27"),
        "/ngrok-alternative": new Date("2026-07-15"),
        "/features": new Date("2026-07-15"),
        "/about": new Date("2026-03-20"),
        "/downloads": new Date("2026-03-28"),
        "/compare/ngrok": new Date("2026-03-27"),
        "/compare/cloudflare": new Date("2026-03-27"),
        "/compare/tuna": new Date("2026-03-27"),
        "/compare/xtunnel": new Date("2026-03-27"),
        "/offer": new Date("2026-02-15"),
        "/terms": new Date("2026-02-15"),
        "/privacy": new Date("2026-02-15"),
      },
    }),
  ],
  define: {
    __VUE_I18N_FULL_INSTALL__: true,
    __VUE_I18N_LEGACY_API__: false,
    __INTLIFY_JIT_COMPILATION__: true,
    __INTLIFY_DROP_MESSAGE_COMPILER__: false,
    __INTLIFY_PROD_DEVTOOLS__: false,
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
    __VUE_PROD_DEVTOOLS__: false,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    minify: "terser",
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
        passes: 2,
        pure_funcs: ["console.log", "console.info", "console.debug"],
      },
      mangle: {
        toplevel: true,
      },
      format: {
        comments: false,
      },
    },
    cssMinify: "lightningcss",
  },
  ssgOptions: {
    script: "async",
    formatting: "minify",
    // reduceInlineStyles: beasties folds any <style> already in index.html into
    // its critical CSS and drops the selectors that do not match the static
    // markup. That killed the html.dark background rule — the class only
    // appears at runtime — and dark-mode visitors got a white flash.
    // allowRules: the .dark block holds every colour variable for the dark
    // theme, and body reads --background from it. Beasties drops it because the
    // prerendered markup never carries the class — the inline script adds it —
    // so dark visitors got a white page until the external CSS landed.
    beastiesOptions: {
      fonts: false,
      preloadFonts: false,
      reduceInlineStyles: false,
      allowRules: [/^\.dark$/],
      // The whole stylesheet is 76 KiB — 13 KiB over the wire — so inline it
      // rather than pay a blocking round trip for it. Beasties was already
      // inlining a 17 KiB critical subset and then blocking on the rest;
      // this drops the request and, unlike loading it asynchronously, there
      // is nothing left to swap in later and shift the layout.
      inlineThreshold: 120000,
      // Not preload: 'swap'. Measured on PageSpeed: it cuts FCP from 3.3 s to
      // 0.9 s, but the swap reflows the page and CLS goes 0 -> 0.185, past the
      // 0.1 threshold, taking the score 68 -> 64. The blocking stylesheet is
      // what keeps the layout still; worth revisiting only once the critical
      // CSS covers enough of the page that the swap changes nothing.
    },
    includedRoutes() {
      const pages = ["/", "/login", "/register", "/offer", "/terms", "/pricing", "/privacy", "/about", "/downloads", "/abuse", "/aup", "/disclaimer", "/ngrok-alternative", "/features", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"];
      const ruPages = pages.map((p) => `/ru${p === "/" ? "" : p}`);
      const enPages = pages.map((p) => `/en${p === "/" ? "" : p}`);
      return [...pages, ...ruPages, ...enPages];
    },
  },
  server: {
    allowedHosts: [
      "fxtun.dev",
      "test.fxtun.dev",
    ],
    proxy: {
      "/api": {
        target: "https://fxtun.dev",
        changeOrigin: true,
        secure: true,
      },
    },
  },
});
