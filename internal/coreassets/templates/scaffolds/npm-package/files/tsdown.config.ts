import { defineConfig } from "tsdown";

export default defineConfig({
    entry: ["src/index.ts"],
    dts: true,
    // Emit .js/.d.ts (not tsdown's default .mjs/.d.mts) to match package.json exports.
    outExtensions: () => ({ js: ".js", dts: ".d.ts" }),
});
