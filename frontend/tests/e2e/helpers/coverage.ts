import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { test as baseTest, expect } from "@playwright/test";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const NYC_OUTPUT_DIR = path.resolve(__dirname, "../../../.nyc_output");

export const test = baseTest.extend({
	page: async ({ page }, use) => {
		await use(page);

		if (process.env.VITE_COVERAGE === "true") {
			const coverage = await page.evaluate(() => {
				// biome-ignore lint/suspicious/noExplicitAny: window has no __coverage__ property by default
				return (window as any).__coverage__;
			});

			if (coverage) {
				if (!fs.existsSync(NYC_OUTPUT_DIR)) {
					fs.mkdirSync(NYC_OUTPUT_DIR, { recursive: true });
				}
				const randomId = Math.random().toString(36).substring(2, 15);
				const filename = path.join(NYC_OUTPUT_DIR, `coverage-${randomId}.json`);
				fs.writeFileSync(filename, JSON.stringify(coverage));
			}
		}
	},
});

export { expect };
