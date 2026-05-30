import { expect, test } from "@playwright/test";
import { mockWailsAPI } from "./helpers/mock-wails";

test.beforeEach(async ({ page }) => {
	// Wails APIのモックを注入
	await mockWailsAPI(page);

	page.on("console", (msg) => {
		console.log(`[BROWSER CONSOLE]: ${msg.type()}: ${msg.text()}`);
	});
	page.on("pageerror", (err) => {
		console.error(`[BROWSER ERROR]: ${err.message}`);
	});

	await page.goto("/");
});

test.describe("セッション詳細画面 E2E テスト", () => {
	test("セッション詳細画面への遷移と一覧画面への戻り動作が正しく機能すること", async ({
		page,
	}) => {
		// session-1の行をクリックして詳細画面へ遷移
		await page.locator("text=sess-001").click();

		// URLが詳細画面のパスになっていることを確認
		await expect(page).toHaveURL(/.*#\/sessions\/sess-001-uuid-long-name/);

		// 詳細画面にセッションIDが表示されていることを確認
		await expect(page.locator("text=sess-001-uuid-long-name")).toBeVisible();

		// 「← Back to List」ボタンをクリック
		await page.locator("button:has-text('← Back to List')").click();

		// URLが一覧画面に戻っていることを確認
		await expect(page).toHaveURL(/.*#\//);

		// 再びセッション一覧が表示されていることを確認
		await expect(page.locator("text=sess-001")).toBeVisible();
	});
});
