import { expect, test } from "./helpers/coverage";
import { mockWailsAPI } from "./helpers/mock-wails";

test.beforeEach(async ({ page }) => {
	await mockWailsAPI(page);
	await page.goto("/");
	// Wait for the app to mount properly before proceeding
	await expect(page.getByRole("button", { name: "Codex" })).toBeVisible({
		timeout: 15000,
	});
});

test.describe("履歴コーパス分析画面 E2E テスト", () => {
	test("Claude Code に切り替えるとコーパス分析タブが表示され、クリックして分析を実行できること", async ({
		page,
	}) => {
		// 初期状態（Codex）ではコーパス分析タブが表示されていないことを確認
		await expect(
			page.locator("button:has-text('コーパス分析')"),
		).not.toBeVisible();

		// Claude Code に切り替え
		await page.getByRole("button", { name: "Claude Code" }).click();

		// コーパス分析タブが表示されることを確認
		const analysisTab = page.locator("button:has-text('コーパス分析')");
		await expect(analysisTab).toBeVisible();

		// タブをクリック
		await analysisTab.click();

		// ツールバーと実行ボタンが表示されることを確認
		const analyzeBtn = page.getByRole("button", { name: "履歴分析を実行" });
		await expect(analyzeBtn).toBeVisible();
		await expect(page.locator("select")).toBeVisible();

		// 分析を実行
		await analyzeBtn.click();

		// 結果のダッシュボードが表示されることを確認
		await expect(page.locator("text=総セッションファイル数")).toBeVisible();
		await expect(page.locator("text=総トランスクリプト行数")).toBeVisible();
		await expect(page.locator("text=パースエラー発生数")).toBeVisible();

		// 表の要素が正しく表示されること
		await expect(page.locator("text=message.role")).toBeVisible();
		await expect(page.locator("text=message.content[].type")).toBeVisible();
		await expect(page.locator("text=bash")).toBeVisible();

		// プライバシー保護の文言とハッシュ表示
		await expect(page.locator("text=匿名性と安全性の保証")).toBeVisible();
		await expect(
			page.locator("text=hash-cmd-1-sha256-long-hex-string"),
		).toBeVisible();
	});

	test("エラー発生時にエラーメッセージが表示されること", async ({ page }) => {
		await page.getByRole("button", { name: "Claude Code" }).click();
		await page.locator("button:has-text('コーパス分析')").click();

		// モックのトリガー用に "trigger-error" オプションを動的に追加
		await page.locator("select").evaluate((el) => {
			const opt = document.createElement("option");
			opt.value = "trigger-error";
			opt.text = "Error Test";
			(el as HTMLSelectElement).appendChild(opt);
		});

		await page.locator("select").selectOption("trigger-error");
		await page.getByRole("button", { name: "履歴分析を実行" }).click();

		// エラーメッセージが表示されることを確認
		await expect(
			page.locator("text=Mocked Corpus Analyze Error"),
		).toBeVisible();
	});
});
