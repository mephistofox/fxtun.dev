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
      "mfdev.ru",
      "fxtun.ru",
      "fxtun.pro",
      "fxtun.dev",
      "test.mfdev.ru",
      "test.fxtun.ru",
      "test.fxtun.dev",
    ],
    proxy: {
      "/api": {
        target: "https://mfdev.ru",
        changeOrigin: true,
        secure: true,
      },
    },
  },
});
