const fs = require("node:fs");
const path = require("node:path");
const globals = require("globals");
const shared = require("../../packages/config/eslint.config.js");

// Nuxt declares these imports during prepare. Keep lint in step with the app's
// real imports rather than maintaining a second list by hand.
const nuxtGlobals = {};
for (const name of ["imports", "nitro-imports", "shared-imports"]) {
  const file = path.join(__dirname, ".nuxt", "types", `${name}.d.ts`);
  if (!fs.existsSync(file))
    throw new Error("Run nuxt prepare before linting the dashboard.");
  for (const match of fs
    .readFileSync(file, "utf8")
    .matchAll(/\bconst\s+([\w$]+):/g))
    nuxtGlobals[match[1]] = "readonly";
}
module.exports = [
  ...shared,
  {
    files: ["**/*.{js,mjs,cjs,ts,tsx,vue}"],
    languageOptions: {
      globals: { ...globals.browser, ...globals.node, ...nuxtGlobals },
    },
  },
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parserOptions: {
        project: [path.join(__dirname, "tsconfig.eslint.json")],
      },
    },
    // TypeScript resolves types and Nuxt imports; the base JS rule cannot.
    rules: {
      "no-undef": "off",
      // Recommend these stylistic changes without changing intentional falsy fallbacks.
      "@typescript-eslint/prefer-nullish-coalescing": "warn",
      "@typescript-eslint/prefer-optional-chain": "warn",
    },
  },
  {
    files: ["**/*.vue"],
    processor: "vue/vue",
    // Prettier owns whitespace and tag layout. These rules conflict with its
    // formatting, while the remaining Vue and JavaScript checks still apply.
    rules: {
      "no-undef": "off",
      "vue/max-attributes-per-line": "off",
      "vue/html-self-closing": "off",
    },
  },
];
