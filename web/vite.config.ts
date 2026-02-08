import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Sitemap from "vite-plugin-sitemap";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    Sitemap({
      hostname: "https://fxtun.dev",
      dynamicRoutes: ["/login", "/register", "/offer"],
      exclude: ["/docs/offer"],
      generateRobotsTxt: false,
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
