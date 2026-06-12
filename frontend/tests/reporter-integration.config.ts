import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: "./e2e",
	testMatch: "compact-reporter.spec.ts",
	workers: 1,
});
