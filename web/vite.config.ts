import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Sitemap from "vite-plugin-sitemap";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    Sitemap({
      hostname: "https://fxtun.dev",
      dynamicRoutes: ["/pricing", "/offer", "/terms", "/privacy"],
      exclude: ["/docs/offer", "/ru", "/ru/*", "/en", "/en/*", "/login", "/register"],
      generateRobotsTxt: false,
      changefreq: {
        "/": "weekly",
        "/pricing": "weekly",
        "/offer": "monthly",
        "/terms": "monthly",
        "/privacy": "monthly",
      },
      priority: {
        "/": 1.0,
        "/pricing": 0.9,
        "/offer": 0.5,
        "/terms": 0.5,
        "/privacy": 0.5,
      },
      robots: [{ userAgent: "*", allow: "/" }],
      readable: true,
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
    beastiesOptions: false,
    includedRoutes() {
      const pages = ["/", "/login", "/register", "/offer", "/terms", "/pricing", "/privacy"];
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
