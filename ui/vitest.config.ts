import { defineConfig, configDefaults } from "vitest/config";
import { fileURLToPath } from "node:url";

export default defineConfig({
  resolve: {
    alias: {
      src: fileURLToPath(new URL("./src", import.meta.url)),
      app: fileURLToPath(new URL(".", import.meta.url)),
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
