import { defineConfig } from "vitest/config";
import base from "./vitest.config";
export default defineConfig({
  ...base,
  define: { "import.meta.client": "false", "import.meta.server": "true" },
  test: { ...base.test, include: ["tests/**/*.server.spec.ts"] },
});
