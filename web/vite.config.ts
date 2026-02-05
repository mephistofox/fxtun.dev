import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "path";

export default defineConfig({
  plugins: [vue()],
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
      "fxtun.ru",
      "fxtun.pro",
      "mfdev.ru",
      "test.fxtun.dev",
      "test.fxtun.ru",
      "test.mfdev.ru",
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
