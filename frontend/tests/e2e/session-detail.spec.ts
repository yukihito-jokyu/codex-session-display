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

	test("React Flow キャンバスにノードとエッジが正しく描画され、コントロールUIが表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// キャンバス（react-flowコンテナ）が存在することを確認
		const reactFlow = page.locator(".react-flow");
		await expect(reactFlow).toBeVisible();

		// ズーム・パンなどのコントロールボタン（react-flow__controls）が表示されていることを確認
		const controls = page.locator(".react-flow__controls");
		await expect(controls).toBeVisible();

		// 登録したノードが描画されていることを確認
		await expect(page.locator("text=User Instructions").first()).toBeVisible();
		await expect(page.locator("text=User Message").first()).toBeVisible();
		await expect(page.locator("text=Orphan Complete").first()).toBeVisible();
	});

	test("ターン外ノードが破線ボーダーかつ[System]プレフィックス付きで表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// turnIndex === -1 のターン外ノードには [System] プレフィックスが付与されていることを確認
		const systemMeta = page.locator("text=[System] Session Meta");
		await expect(systemMeta).toBeVisible();

		// ターン外ノードクラスが適用されていることを確認
		const nodeElement = page.locator(".node-out-of-turn").first();
		await expect(nodeElement).toBeVisible();
	});

	test("警告アイコンを含む孤立イベントが警告色で表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 孤立イベントのラベル "Orphan Complete" と警告アイコン "⚠️" を確認
		await expect(page.locator("text=Orphan Complete").first()).toBeVisible();
		await expect(page.locator("text=⚠️").first()).toBeVisible();

		// 警告色のクラス `.node-warning` が適用されていることを確認
		const warningNode = page.locator(".node-warning");
		await expect(warningNode).toBeVisible();
	});

	test("ContextDocNodeの展開・折りたたみがクリックでトグル動作すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// デフォルト（折りたたみ状態）の表示（"▸ クリックして展開" など）を確認
		const textTrigger = page.locator("text=▸ クリックして展開");
		await expect(textTrigger).toBeVisible();

		// クリックして展開
		await textTrigger.click();

		// 展開されたインストラクションのテキスト全文（fullText）が表示されることを確認
		const expandedText = page
			.locator(
				"text=This is a detailed user instruction text that can be expanded.",
			)
			.first();
		await expect(expandedText).toBeVisible();

		// 折りたたみトグル（"▾ User Instructions"）をクリックして折りたたむ
		await page.locator("text=▾ User Instructions").first().click();

		// 再び "▸ クリックして展開" が表示されることを確認
		await expect(page.locator("text=▸ クリックして展開")).toBeVisible();
	});

	test("ノードを選択した際に画面下部に詳細パネルが表示され、閉じる動作が機能すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 最初は詳細パネルが表示されていないことを確認
		const panel = page.locator("text=Node Detail:");
		await expect(panel).not.toBeVisible();

		// User Message ノードをクリック
		await page.locator(".react-flow__node-userMessage").click();

		// 詳細パネルが表示され、選択したノードのラベルが含まれることを確認
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();

		// fullTextの表示を確認
		await expect(
			page.locator("text=Hello, agent, please help me."),
		).toBeVisible();

		// 閉じるボタン「✕」をクリック
		await page.locator("button:has-text('✕')").click();

		// 詳細パネルが非表示になることを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		// 再びノードをクリックして詳細パネルを表示
		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.locator("text=Node Detail:")).toBeVisible();

		// キャンバスの余白（.react-flow__pane）をクリックして閉じることを確認
		await page.locator(".react-flow__pane").click();
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();
	});

	test("詳細画面でのAPIエラー発生時にエラー画面が表示され、一覧に戻れること", async ({
		page,
	}) => {
		// エラーが発生するID（trigger-error）で詳細画面へ直接遷移
		await page.goto("/#/sessions/trigger-error");

		// エラーメッセージが表示されていることを確認
		await expect(
			page.locator(
				"text=Failed to fetch session detail: Mocked Detail API Error",
			),
		).toBeVisible();

		// 「Back to List」ボタンをクリック
		await page.locator("button:has-text('Back to List')").click();

		// 一覧画面にリダイレクトされ、セッション一覧が表示されていることを確認
		await expect(page).toHaveURL(/.*#\//);
		await expect(page.locator("text=sess-001")).toBeVisible();
	});

	test("トークンバッジが適切にフォーマットされてノード上に描画されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// User Message ノード内のトークンバッジ（150000 -> 150K, boundCount=2 -> ×2）を確認
		const badge = page.locator(".react-flow__node-userMessage >> text=150K ×2");
		await expect(badge).toBeVisible();
	});
});
