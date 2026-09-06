import { readdirSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig } from "vite-plus";
// Pure build-time license collector; no Cockpit bridge or runtime imports.
import { licenses } from "../cockpit/build/licenses.ts";

export default defineConfig({
  base: "/app/",
  server: { host: "127.0.0.1", strictPort: true },
  resolve: {
    alias: ["RedHatDisplay", "RedHatText", "RedHatMono"].flatMap((family) => {
      const dir = resolve(import.meta.dirname, "node_modules/@patternfly/patternfly/assets/fonts", family);
      return readdirSync(dir).map((file) => ({ find: `../../static/fonts/${file}`, replacement: resolve(dir, file) }));
    }),
  },
  build: { outDir: "dist", manifest: true, sourcemap: false, assetsInlineLimit: 0 },
  plugins: [{
    name: "soda-dashboard-licenses",
    generateBundle: {
      order: "post",
      handler(_options, bundle) {
        this.emitFile({
          type: "asset", fileName: "LICENSES.txt",
          source: licenses(Object.values(bundle).flatMap((item) => item.type === "chunk" ? Object.keys(item.modules) : []), resolve(import.meta.dirname, "../cockpit")),
        });
      },
    },
  }],
  test: { include: ["src/**/*.test.{ts,tsx}"], environment: "jsdom", restoreMocks: true },
});
