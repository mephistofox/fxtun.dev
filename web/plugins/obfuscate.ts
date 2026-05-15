import type { Plugin } from "vite";
import { readFileSync, writeFileSync, readdirSync } from "fs";
import { join } from "path";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

export function obfuscateBundle(): Plugin {
  return {
    name: "vite-plugin-obfuscate-bundle",
    apply: "build",
    enforce: "post",
    closeBundle() {
      const JavaScriptObfuscator = require("javascript-obfuscator");

      const assetsDir = join(process.cwd(), "dist", "assets");
      let files: string[];
      try {
        files = readdirSync(assetsDir).filter((f: string) => f.endsWith(".js"));
      } catch {
        return;
      }

      console.log(`\nObfuscating ${files.length} JS files...`);

      for (const file of files) {
        const filePath = join(assetsDir, file);
        const code = readFileSync(filePath, "utf-8");
        const result = JavaScriptObfuscator.obfuscate(code, {
          compact: true,
          controlFlowFlattening: true,
          controlFlowFlatteningThreshold: 0.5,
          deadCodeInjection: false,
          debugProtection: false,
          identifierNamesGenerator: "hexadecimal",
          renameGlobals: false,
          selfDefending: false,
          stringArray: true,
          stringArrayCallsTransform: true,
          stringArrayEncoding: ["base64"],
          stringArrayThreshold: 0.5,
          splitStrings: true,
          splitStringsChunkLength: 10,
          transformObjectKeys: false,
          unicodeEscapeSequence: false,
          reservedNames: ["^__"],
        });
        writeFileSync(filePath, result.getObfuscatedCode());
        console.log(`  ✓ ${file}`);
      }

      console.log("Obfuscation complete.\n");
    },
  };
}
