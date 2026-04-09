/// <reference types="vite/client" />

declare module "vite-plugin-obfuscator" {
  import type { Plugin } from "vite";
  export function viteObfuscateFile(options?: Record<string, unknown>): Plugin;
}
