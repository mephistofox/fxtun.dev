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
    allowedHosts: ["test.mfdev.ru", "mfdev.ru", "unms.ru"],
    proxy: {
      "/api": {
        target: "https://mfdev.ru",
        changeOrigin: true,
        secure: true,
      },
    },
  },
});
