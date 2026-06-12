import { expect, test } from "../e2e/helpers/coverage";

test("failed browser fixture", async ({ page }) => {
	await page.goto("data:text/html,<title>browser fixture</title>");
	await page.evaluate(() => {
		console.error("ブラウザconsoleマーカー");
		setTimeout(() => {
			throw new Error("ブラウザpage errorマーカー");
		}, 0);
	});
	await page.waitForTimeout(50);

	expect("actual", "ブラウザログ確認用の失敗").toBe("expected");
});
