import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Sitemap from "vite-plugin-sitemap";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    Sitemap({
      hostname: "https://fxtun.dev",
      dynamicRoutes: ["/pricing", "/offer", "/terms", "/privacy", "/about", "/downloads", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"],
      exclude: ["/docs/offer", "/ru", "/ru/*", "/en", "/en/*", "/login", "/register"],
      generateRobotsTxt: false,
      robots: [{ userAgent: "*", allow: "/" }],
      changefreq: {
        "/": "weekly",
        "/pricing": "weekly",
        "/compare/*": "monthly",
        "/about": "monthly",
        "/downloads": "monthly",
        "/offer": "yearly",
        "/terms": "yearly",
        "/privacy": "yearly",
      },
      priority: {
        "/": 1.0,
        "/pricing": 0.9,
        "/compare/*": 0.8,
        "/about": 0.7,
        "/downloads": 0.8,
        "/offer": 0.3,
        "/terms": 0.3,
        "/privacy": 0.3,
      },
      lastmod: {
        "/": new Date("2026-03-28"),
        "/pricing": new Date("2026-03-27"),
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
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  ssgOptions: {
    script: "async",
    formatting: "minify",
    beastiesOptions: { fonts: false, preloadFonts: false },
    includedRoutes() {
      const pages = ["/", "/login", "/register", "/offer", "/terms", "/pricing", "/privacy", "/about", "/downloads", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"];
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
