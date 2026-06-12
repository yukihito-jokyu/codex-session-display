import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: ".",
	reporter: "../../tests/reporters/compact-reporter.ts",
	workers: 1,
});
