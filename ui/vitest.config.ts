import { defineConfig, configDefaults } from "vitest/config";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath } from "node:url";

export default defineConfig({
  // Single-file components. Quasar's build pipeline compiles the SPA, but
  // vitest runs off this config, and without the plugin a test cannot import a
  // .vue file at all.
  plugins: [vue()],
  resolve: {
    alias: {
      src: fileURLToPath(new URL("./src", import.meta.url)),
      app: fileURLToPath(new URL(".", import.meta.url)),
      // the wasm pkg has no main/exports entry, so node-style resolution fails;
      // tests vi.mock the module — this alias only satisfies the resolver.
      "image-wasm": fileURLToPath(new URL("../image-wasm/pkg/image_wasm.js", import.meta.url)),
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    include: ["tests/**/*.spec.ts"],
    // Playwright e2e specs (tests/e2e) use the Playwright runner, not vitest.
    exclude: [...configDefaults.exclude, "tests/e2e/**"],
  },
});
