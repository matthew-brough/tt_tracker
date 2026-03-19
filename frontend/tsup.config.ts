import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/main.ts"],
  outDir: "dist",
  format: "iife",
  minify: true,
  sourcemap: true,
  clean: true,
  outExtension: () => ({ js: ".js" }),
});
