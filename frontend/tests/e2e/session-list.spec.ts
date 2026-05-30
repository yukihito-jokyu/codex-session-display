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

test.describe("セッション一覧画面 E2E テスト", () => {
	test("初期ロード時に2026年5月のセッション一覧が正しく表示されること", async ({
		page,
	}) => {
		// 月ナビゲーターの初期表示が「2026年 5月」であることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();

		// セッション1とセッション2が表示されていることを確認
		// SessionRowでは IDの先頭8文字が表示される (sess-001-uuid-long-name -> sess-001)
		await expect(page.locator("text=sess-001")).toBeVisible();
		await expect(page.locator("text=sess-002")).toBeVisible();
		// 4月のセッション3は表示されていないことを確認
		await expect(page.locator("text=sess-003")).not.toBeVisible();
	});

	test("月のナビゲーションが正しく機能すること", async ({ page }) => {
		// 初期表示は5月
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();

		// 前の月「◀」ボタンをクリック
		await page.locator("button:has-text('◀')").click();

		// 月ナビゲーターが「2026年 4月」に切り替わることを確認
		await expect(page.locator("span:has-text('2026年 4月')")).toBeVisible();

		// 4月のセッション3が表示され、5月のセッション1&2が非表示になることを確認
		await expect(page.locator("text=sess-003")).toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();
		await expect(page.locator("text=sess-002")).not.toBeVisible();

		// 次の月「▶」ボタンをクリック
		await page.locator("button:has-text('▶')").click();

		// 5月に戻ることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();
		await expect(page.locator("text=sess-001")).toBeVisible();
		await expect(page.locator("text=sess-002")).toBeVisible();
	});

	test("検索バーによるフィルタリングが正しく機能すること", async ({ page }) => {
		// 検索入力欄を取得して「openai」と入力
		const searchInput = page.locator("input[placeholder*='Search']");
		if ((await searchInput.count()) === 0) {
			// placeholderが異なる場合、汎用的にinputタグを取得
			await page.locator("input").first().fill("openai");
		} else {
			await searchInput.fill("openai");
		}

		// openai의モデルプロバイダを持つsession-2のみが表示され、session-1が消えることを確認
		await expect(page.locator("text=sess-002")).toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();
	});

	test("未解析セッションが正しく表示されること", async ({ page }) => {
		// 2026年5月の未解析セッション「sess-004」のIDが表示されていることを確認
		await expect(page.locator("text=sess-004")).toBeVisible();

		// 「未解析」バッジが表示されていることを確認
		await expect(page.locator("text=未解析")).toBeVisible();

		// プレースホルダーの「解析前」テキストが表示されていることを確認
		await expect(page.locator("text=解析前").first()).toBeVisible();
	});

	test("アコーディオン（年・月・日ノード）の開閉動作が正しく機能すること", async ({
		page,
	}) => {
		// 初期ロード時にはセッション1が表示されていることを確認
		await expect(page.locator("text=sess-001")).toBeVisible();

		// 「20日」のヘッダー（日ノード）をクリックして折りたたむ
		const dayHeader = page.locator("div[role='button']:has-text('20日')");
		await dayHeader.click();

		// セッション1が非表示になることを確認
		await expect(page.locator("text=sess-001")).not.toBeVisible();

		// 再び「20日」のヘッダーをクリックして展開
		await dayHeader.click();

		// セッション1が再び表示されることを確認
		await expect(page.locator("text=sess-001")).toBeVisible();

		// 「5月」のヘッダー（月ノード）をクリックして折りたたむ
		const monthHeader = page.locator("div[role='button']:has-text('5月')");
		await monthHeader.click();

		// 日ノード「20日」が非表示になることを確認
		await expect(page.locator("text=20日")).not.toBeVisible();
	});

	test("エラー発生時のエラー表示とクエリクリアによる復帰が正しく機能すること", async ({
		page,
	}) => {
		// 検索入力欄に「trigger-error」と入力してエラーを発生させる
		const searchInput = page.locator("input[placeholder*='Search']");
		if ((await searchInput.count()) === 0) {
			await page.locator("input").first().fill("trigger-error");
		} else {
			await searchInput.fill("trigger-error");
		}

		// エラーメッセージが表示されることを確認
		await expect(
			page.locator("text=Failed to fetch sessions: Mocked API Error"),
		).toBeVisible();
		await expect(page.locator("text=Retry")).toBeVisible();

		// 検索入力欄をクリアする
		if ((await searchInput.count()) === 0) {
			await page.locator("input").first().fill("");
		} else {
			await searchInput.fill("");
		}

		// エラーメッセージが消え、セッション1が表示される（復帰）ことを確認
		await expect(
			page.locator("text=Failed to fetch sessions:"),
		).not.toBeVisible();
		await expect(page.locator("text=sess-001")).toBeVisible();
	});

	test("検索結果が0件の場合に空表示が表示されること", async ({ page }) => {
		// 検索入力欄に存在しないセッションの条件を入力
		const searchInput = page.locator("input[placeholder*='Search']");
		if ((await searchInput.count()) === 0) {
			await page.locator("input").first().fill("nonexistent-query-string");
		} else {
			await searchInput.fill("nonexistent-query-string");
		}

		// 空表示のメッセージが表示されることを確認
		await expect(
			page.locator("text=No sessions found matching filters."),
		).toBeVisible();
		// フォルダのアイコン等が表示されていることを確認
		await expect(page.locator("text=📂")).toBeVisible();
	});

	test("月ナビゲーターの年またぎ境界値動作が正しく機能すること", async ({
		page,
	}) => {
		// 初期表示は「2026年 5月」
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();

		// 「◀」ボタンを4回クリックして「2026年 1月」へ移動
		const prevBtn = page.locator("button:has-text('◀')");
		for (let i = 0; i < 4; i++) {
			await prevBtn.click();
		}
		await expect(page.locator("span:has-text('2026年 1月')")).toBeVisible();

		// さらにもう1回「◀」ボタンをクリックして前年「2025年 12月」へ遷移
		await prevBtn.click();
		await expect(page.locator("span:has-text('2025年 12月')")).toBeVisible();

		// 「▶」ボタンをクリックして翌年「2026年 1月」に戻る
		const nextBtn = page.locator("button:has-text('▶')");
		await nextBtn.click();
		await expect(page.locator("span:has-text('2026年 1月')")).toBeVisible();
	});
});
