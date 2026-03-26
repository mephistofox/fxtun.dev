import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Sitemap from "vite-plugin-sitemap";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    Sitemap({
      hostname: "https://fxtun.dev",
      dynamicRoutes: ["/pricing", "/offer", "/privacy", "/about", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"],
      exclude: ["/docs/offer", "/ru", "/ru/*", "/en", "/en/*", "/login", "/register", "/terms"],
      generateRobotsTxt: false,
      robots: [{ userAgent: "*", allow: "/" }],
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
      const pages = ["/", "/login", "/register", "/offer", "/terms", "/pricing", "/privacy", "/about", "/compare/ngrok", "/compare/cloudflare", "/compare/tuna", "/compare/xtunnel"];
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
