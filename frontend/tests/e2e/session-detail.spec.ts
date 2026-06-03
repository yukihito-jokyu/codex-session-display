import { expect, test } from "./helpers/coverage";
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
	test("初回表示時にフロントエンド準備完了をバックエンドへ通知すること", async ({
		page,
	}) => {
		const readyCalls = await page.evaluate(() => {
			return (
				(window as Window & { __frontendReadyCalls?: number })
					.__frontendReadyCalls || 0
			);
		});

		expect(readyCalls).toBeGreaterThanOrEqual(1);
	});

	test("open-session-file イベント受信時に対象セッション詳細へ遷移すること", async ({
		page,
	}) => {
		await page.evaluate(() => {
			(
				window as Window & { __emitWailsEvent?: (...args: unknown[]) => void }
			).__emitWailsEvent?.("open-session-file", "/path/to/session-2");
		});

		await expect(page).toHaveURL(/.*#\/sessions\/sess-002-uuid-long-name/);
		await expect(page.locator("text=sess-002-uuid-long-name")).toBeVisible();

		const resolveCalls = await page.evaluate(() => {
			return (
				(
					window as Window & {
						__resolveSessionIDCalls?: Array<{ filePath: string }>;
					}
				).__resolveSessionIDCalls || []
			);
		});

		expect(resolveCalls).toEqual([{ filePath: "/path/to/session-2" }]);
	});

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

	test("初期表示時から viewport 外のノードが DOM に描画されないこと", async ({
		page,
	}) => {
		await page.evaluate(() => {
			const testWindow = window as Window & {
				__farAwayNodeRendered?: boolean;
			};
			testWindow.__farAwayNodeRendered = false;

			const observer = new MutationObserver((mutations) => {
				for (const mutation of mutations) {
					for (const addedNode of mutation.addedNodes) {
						if (addedNode.textContent?.includes("Far Away Node")) {
							testWindow.__farAwayNodeRendered = true;
						}
					}
				}
			});
			observer.observe(document, { childList: true, subtree: true });
		});

		await page.goto("/#/sessions/performance-visibility");

		await expect(page.locator(".react-flow")).toBeVisible();
		await page.waitForTimeout(500);

		expect(
			await page.evaluate(() => {
				return (window as Window & { __farAwayNodeRendered?: boolean })
					.__farAwayNodeRendered;
			}),
		).toBe(false);
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
		const warningNode = page.locator(".node-warning").first();
		await expect(warningNode).toBeVisible();
	});

	test("ContextDocNodeの展開・折りたたみがクリックでトグル動作すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// デフォルト（折りたたみ状態）の表示（"▸ クリックして展開" など）を確認
		const contextNode = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "User Instructions" })
			.first();
		const textTrigger = contextNode.locator("text=▸ クリックして展開");
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
		await contextNode.locator("text=▾ User Instructions").click();

		// 再び "▸ クリックして展開" が表示されることを確認
		await expect(textTrigger).toBeVisible();
	});

	test("ContextDocNode のヘッダークリックで展開トグルできること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const contextNode = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "User Instructions" })
			.first();
		const header = contextNode.locator(
			"div[role='button']:has-text('User Instructions')",
		);

		await header.click();
		await expect(contextNode.locator("text=▾ User Instructions")).toBeVisible();

		await header.click();
		await expect(contextNode.locator("text=▸ クリックして展開")).toBeVisible();
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
		await page.locator(".react-flow__pane").click({
			position: { x: 8, y: 8 },
		});
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();
	});

	test("未選択時のキャンバス余白クリック後もノード選択が継続して利用できること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		await page.locator(".react-flow__pane").click({
			position: { x: 8, y: 8 },
		});
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("非contextDocノード選択中に別ノードをクリックすると一度選択解除され、再クリックで切り替わること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();

		await page.locator(".react-flow__node-agentMessage").click();
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		await page.locator(".react-flow__node-agentMessage").click();
		await expect(page.locator("text=Node Detail: Agent Message")).toBeVisible();
	});

	test("ノード選択中に React Flow のコントロール本体や SVG アイコンを押しても選択が維持されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();

		await page.evaluate(() => {
			const button = document.querySelector(".react-flow__controls-button");
			if (!(button instanceof HTMLButtonElement)) {
				throw new Error("React Flow control button not found");
			}
			button.dispatchEvent(new MouseEvent("click", { bubbles: true }));
		});
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();

		await page.evaluate(() => {
			const icon = document.querySelector(".react-flow__controls-button svg");
			if (!(icon instanceof SVGElement)) {
				throw new Error("React Flow control icon not found");
			}
			icon.dispatchEvent(new MouseEvent("click", { bubbles: true }));
		});
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
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

	test("詳細画面のエラー表示からログフォルダ展開とログパスコピーができること", async ({
		page,
	}) => {
		await page.goto("/#/sessions/trigger-error");

		await expect(
			page.locator(
				"text=Failed to fetch session detail: Mocked Detail API Error",
			),
		).toBeVisible();

		await page.getByRole("button", { name: "ログフォルダを開く" }).click();
		const openCalls = await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			return (window as any).__openLogDirectoryCalls || 0;
		});
		expect(openCalls).toBe(1);

		await page.getByRole("button", { name: "ログパスをコピー" }).click();
		const copiedTexts = await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			return (window as any).__copiedTexts || [];
		});
		expect(copiedTexts).toContain("/Users/test/.codex-display/logs/app.log");

		await expect(page.locator("text=ログパスをコピーしました")).toBeVisible();
	});

	test("トークンバッジが適切にフォーマットされてノード上に描画されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// User Message ノード内のトークンバッジ（1,000,000 -> 1M, boundCount=2 -> ×2）を確認
		const badge = page.locator(".react-flow__node-userMessage >> text=1M ×2");
		await expect(badge).toBeVisible();
	});

	test("トークンバッジクリックで下部パネルが左右分割表示に切り替わり、トークン詳細が表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		await page
			.getByRole("button", { name: "User Message token badge" })
			.click();

		await expect(page.getByTestId("bottom-panel")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-split")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
		await expect(page.locator("text=Token Detail")).toBeVisible();
		await expect(page.locator("text=2 entries")).toBeVisible();
		await expect(page.locator("text=Index #0")).toBeVisible();
		await expect(page.locator("text=Index #1")).toBeVisible();
	});

	test("左右分割表示の境界ドラッグでノード詳細ペインの幅が変わること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		await page
			.getByRole("button", { name: "User Message token badge" })
			.click();

		const leftPane = page.getByTestId("bottom-panel-node-detail");
		const resizer = page.getByTestId("bottom-panel-resizer");
		const before = await leftPane.boundingBox();
		if (!before) {
			throw new Error("left pane bounding box not found");
		}

		const handle = await resizer.boundingBox();
		if (!handle) {
			throw new Error("resizer bounding box not found");
		}

		await page.mouse.move(
			handle.x + handle.width / 2,
			handle.y + handle.height / 2,
		);
		await page.mouse.down();
		await page.mouse.move(
			handle.x + handle.width / 2 + 120,
			handle.y + handle.height / 2,
		);
		await page.mouse.up();

		const after = await leftPane.boundingBox();
		if (!after) {
			throw new Error("left pane bounding box not found after drag");
		}

		expect(after.width).toBeGreaterThan(before.width + 40);
	});

	test("右パネル（Session Analytics）に統計カード、グラフ、サマリー、トークンテーブルが正しく描画されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// タイトル "Session Analytics" が表示されていることを確認
		await expect(page.locator("text=Session Analytics")).toBeVisible();

		// 統計カード（ターン数, トータルログ等）が表示されていることを確認
		// Duration: 1m 15s (75000ms)
		await expect(page.locator("text=1m 15s")).toBeVisible();
		// Total Tokens: 150,000 (アサート対象を明確化)
		await expect(
			page.locator("span:has-text('Total Tokens') + span"),
		).toHaveText("150,000");

		// ターンサマリーカードが表示されていることを確認
		await expect(
			page.locator("[class*='summarySection'] >> text=Turn #1"),
		).toBeVisible();
		await expect(
			page.locator("[class*='summarySection'] >> text=Turn #2"),
		).toBeVisible();

		// トークンカウントログテーブルが表示されていることを確認
		await expect(page.locator("text=Token Count Log")).toBeVisible();
		// テーブルヘッダー
		await expect(page.locator("th:has-text('Cached')")).toBeVisible();
	});

	test("TOKEN COUNT LOG の行をクリックした際に、対応するノードが選択されて詳細パネルが表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 最初は詳細パネルが表示されていないことを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		// Token Count Log テーブルの最初の行（classにtokenRowを含む行）をクリック
		await page.locator("tr[class*='tokenRow']").first().click();

		// 詳細パネルが表示されることを確認
		await expect(page.locator("text=Node Detail:")).toBeVisible();

		// 選択されたノード「User Message」の詳細が表示されていることを確認
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("LAST TOKEN CONSUMPTION PER INDEX チャートをクリックした際に、対応するノードが選択されて詳細パネルが表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 最初は詳細パネルが表示されていないことを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		// チャートをクリック (ホバーしてからクリック)
		const chart = page
			.locator("[data-testid='last-token-chart'] .recharts-wrapper")
			.first();
		await chart.hover({ position: { x: 150, y: 100 }, force: true });
		await chart.click({ position: { x: 150, y: 100 }, force: true });

		// 詳細パネルが表示され、対応するノード（node-user-msg -> User Message）の情報が表示されることを確認
		await expect(page.locator("text=Node Detail:")).toBeVisible();
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("LAST TOKEN CONSUMPTION PER INDEX の6桁Y軸ラベルが見切れず表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const chart = page.locator("[data-testid='last-token-chart']").first();
		const sixDigitTick = chart.getByText("100000", { exact: true });

		await expect(sixDigitTick).toBeVisible();

		const chartBox = await chart.getByRole("application").boundingBox();
		const tickBox = await sixDigitTick.boundingBox();

		expect(chartBox).not.toBeNull();
		expect(tickBox).not.toBeNull();
		expect(tickBox?.x).toBeGreaterThanOrEqual(chartBox?.x ?? 0);
	});

	test("TOKEN CONSUMPTION PER TURN の6桁Y軸ラベルが見切れず表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const chart = page
			.getByRole("heading", { name: "Token Consumption per Turn" })
			.locator("xpath=following-sibling::div[1]");
		const sixDigitTick = chart.getByText(/^\d{6,}$/, { exact: true }).first();

		await expect(sixDigitTick).toBeVisible();

		const chartBox = await chart.getByRole("application").boundingBox();
		const tickBox = await sixDigitTick.boundingBox();

		expect(chartBox).not.toBeNull();
		expect(tickBox).not.toBeNull();
		expect(tickBox?.x).toBeGreaterThanOrEqual(chartBox?.x ?? 0);
	});

	test("TOKEN COUNT LOG の累積/ステップ(Last)表示切り替えが正しく動作すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// デフォルトは "Cumulative" (累積) モード
		// mock-wails.ts の2行目（インデックス 1）の Cumulative トータル値は 50,000
		const targetRow = page.locator("tr[class*='tokenRow']").nth(1);
		await expect(targetRow.locator("td").nth(1)).toHaveText("50,000");

		// "Step (Last)" ボタンをクリックして切り替える
		await page.locator("button:has-text('Step (Last)')").click();

		// インデックス 1 の Last トータル値は 30,000 に変化するはず
		await expect(targetRow.locator("td").nth(1)).toHaveText("30,000");

		// 再度 "Cumulative" ボタンをクリックして戻す
		await page.locator("button:has-text('Cumulative')").click();

		// 数値が 50,000 に戻ることを確認
		await expect(targetRow.locator("td").nth(1)).toHaveText("50,000");
	});

	test("右パネル内の各グラフ（Turn Duration, Token Consumption, Tool Calls）が正しく描画されていること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 各グラフセクションの見出しが表示されていることを確認
		await expect(page.locator("text=Turn Duration (seconds)")).toBeVisible();
		await expect(page.locator("text=Token Consumption per Turn")).toBeVisible();
		await expect(page.locator("text=Tool Calls per Turn")).toBeVisible();
	});

	test("ContextDocNodeのキーボード操作（Enter/Space）による展開・折りたたみ検証", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// デフォルトで「▸ クリックして展開」があることを確認
		const contextNode = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "User Instructions" })
			.first();
		const collapsedHeader = contextNode.locator(
			"div[role='button']:has-text('User Instructions')",
		);
		await collapsedHeader.focus();

		// Enterキーで展開する
		await page.keyboard.press("Enter");
		await expect(
			page.locator(
				"text=This is a detailed user instruction text that can be expanded.",
			),
		).toBeVisible();

		// 展開時の「▾ 折りたたむ」ボタンにフォーカスをあてて、Spaceキーで閉じる
		const collapseBtn = page.locator(
			".react-flow__node-contextDoc:has-text('User Instructions') div[role='button']:has-text('▾ 折りたたむ')",
		);
		await collapseBtn.focus();
		await page.keyboard.press("Space");
		await expect(contextNode.locator("text=▸ クリックして展開")).toBeVisible();

		// 「▸ クリックして展開」ボタンにフォーカスをあてて、Enterキーで展開する
		const expandBtn = page.locator(
			".react-flow__node-contextDoc:has-text('User Instructions') div[role='button']:has-text('▸ クリックして展開')",
		);
		await expandBtn.focus();
		await page.keyboard.press("Enter");
		await expect(contextNode.locator("text=▾ 折りたたむ")).toBeVisible();

		// ヘッダーでSpaceキーを押して閉じる
		await collapsedHeader.focus();
		await page.keyboard.press("Space");
		await expect(contextNode.locator("text=▸ クリックして展開")).toBeVisible();
	});

	test("Token Count Log のキーボード操作（Enter/Space）による行選択検証", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 最初は詳細パネルが表示されていないことを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		// 最初のアクティブな行にフォーカスをあてる
		const tokenRow = page.locator("tr[class*='tokenRow']").first();
		await tokenRow.focus();

		// Enterキーを押して選択する
		await page.keyboard.press("Enter");

		// 詳細パネルが表示され、「User Message」になっていることを確認
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();

		// 詳細パネルを閉じる
		await page.locator("button:has-text('✕')").click();
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		// 再度Spaceキーで選択する
		await tokenRow.focus();
		await page.keyboard.press("Space");
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("詳細画面でのリトライボタンクリック時に再読み込みが機能すること", async ({
		page,
	}) => {
		// エラーが発生するIDで遷移
		await page.goto("/#/sessions/trigger-error");
		await expect(
			page.locator(
				"text=Failed to fetch session detail: Mocked Detail API Error",
			),
		).toBeVisible();

		// getSessionDetailの呼び出し履歴をクリアする
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			(window as any).__getSessionDetailCalls = [];
		});

		// 「Retry」ボタンをクリック
		await page.locator("button:has-text('Retry')").click();

		// getSessionDetailが再実行され、1回の呼び出しが記録されていることを確認
		const callsCount = await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			return (window as any).__getSessionDetailCalls?.length || 0;
		});
		expect(callsCount).toBe(1);
	});

	test("メタデータテーブル内のオブジェクト/プリミティブ混在表示のアサーション", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// sessionMeta ノードをクリック
		const metaNode = page.locator(".react-flow__node-sessionMeta").first();
		await metaNode.click();

		// 詳細テーブルが表示されることを確認
		await expect(page.locator("text=Node Detail: Session Meta")).toBeVisible();

		// プリミティブな値 ("1.0.0") が正しく表示されていることを確認
		await expect(page.locator("td:has-text('1.0.0')")).toBeVisible();

		// オブジェクトな値 ({"debug":true}) がJSON文字列化されて表示されていることを確認
		await expect(page.locator("td:has-text('{\"debug\":true}')")).toBeVisible();
	});

	test("統計カードの時間表示（分・秒）のアサーション", async ({ page }) => {
		await page.locator("text=sess-001").click();

		// 期間が「1m 15s」（75000ms）として表示されていることを確認
		await expect(page.locator("text=1m 15s")).toBeVisible();
	});

	test("トークンバッジの全フォーマットのアサーション", async ({ page }) => {
		await page.locator("text=sess-001").click();

		// 2.5M (node-meta, 2500000 tokens)
		const metaBadge = page.locator(
			".react-flow__node-sessionMeta >> text=2.5M",
		);
		await expect(metaBadge).toBeVisible();

		// 1M *2 (node-user-msg, 1000000 tokens, boundCount=2)
		const userBadge = page.locator(
			".react-flow__node-userMessage >> text=1M ×2",
		);
		await expect(userBadge).toBeVisible();

		// 1K (node-user-api-msg, 1000 tokens)
		const apiBadge = page.locator(
			".react-flow__node-userApiMessage >> text=1K",
		);
		await expect(apiBadge).toBeVisible();

		// 1.5K (node-orphan-event, 1500 tokens)
		const orphanBadge = page.locator(
			".react-flow__node-taskEvent >> text=1.5K",
		);
		await expect(orphanBadge).toBeVisible();

		// 500 (node-agent-msg, 500 tokens)
		const agentBadge = page.locator(
			".react-flow__node-agentMessage >> text=500",
		);
		await expect(agentBadge).toBeVisible();
	});

	test("詳細フェッチ文字列エラー発生時の表示検証", async ({ page }) => {
		// 文字列エラーが発生するIDで詳細画面へ遷移
		await page.goto("/#/sessions/trigger-string-error");

		// エラーメッセージとして文字列自体が表示されていることを確認
		await expect(
			page.locator(
				"text=Failed to fetch session detail: Mocked Detail String Error",
			),
		).toBeVisible();
	});

	test("空のセッションデータが返された場合のフォールバック検証", async ({
		page,
	}) => {
		// nodes/edges 等が undefined を返す sess-002 へ遷移
		await page.locator("text=sess-002").click();

		// エラーにはならず、詳細画面が表示されていることを確認
		await expect(
			page.locator("text=Session Detail: sess-002-uuid-long-name"),
		).toBeVisible();

		// キャンバスが表示されていることを確認（空配列として渡されるためクラッシュしない）
		const reactFlow = page.locator(".react-flow");
		await expect(reactFlow).toBeVisible();
	});

	test("ContextDocNodeの「折りたたむ」リンククリックによる折りたたみ検証", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// クリックして展開
		const contextNode = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "User Instructions" })
			.first();
		await contextNode.locator("text=▸ クリックして展開").click();
		await expect(contextNode.locator("text=▾ 折りたたむ")).toBeVisible();

		// 「▾ 折りたたむ」ボタン（リンク）を直接クリックして閉じる
		await contextNode.locator("text=▾ 折りたたむ").click();
		await expect(contextNode.locator("text=▸ クリックして展開")).toBeVisible();
	});

	test("長文 ContextDocNode が 1K 表示となり、折りたたみ領域のキーボード操作で展開できること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const longContextCollapsed = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "Long Context" })
			.locator("div[role='button']:has-text('▸ クリックして展開')")
			.first();
		await expect(longContextCollapsed).toContainText("(1K)");

		await longContextCollapsed.focus();
		await page.keyboard.press("Space");

		await expect(
			page
				.locator(".react-flow__node-contextDoc")
				.filter({ hasText: "Long Context" })
				.locator("text=▾ 折りたたむ")
				.first(),
		).toBeVisible();
	});

	test("空の ContextDocNode は長さ表示なしで描画され、展開しても空表示のまま動作すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const emptyContextNode = page
			.locator(".react-flow__node-contextDoc")
			.filter({ hasText: "Empty Context" })
			.first();
		const collapsed = emptyContextNode
			.locator("div[role='button']")
			.filter({ hasText: "▸ クリックして展開" })
			.first();

		await expect(collapsed).toBeVisible();
		await expect(collapsed).not.toContainText("(");

		await collapsed.click();
		await expect(emptyContextNode.locator("text=▾ 折りたたむ")).toBeVisible();
		await expect(emptyContextNode).not.toContainText("undefined");
	});

	test("追加のノード詳細表示がない場合の検証", async ({ page }) => {
		await page.locator("text=sess-001").click();

		// fullTextとmetaを両方持たない Orphan Complete ノードをクリック
		await page.locator("text=Orphan Complete").first().click();

		// 詳細パネルが表示されていることを確認
		await expect(
			page.locator("text=Node Detail: Orphan Complete"),
		).toBeVisible();

		// 「No additional details available.」が表示されていることを確認
		await expect(
			page.locator("text=No additional details available."),
		).toBeVisible();
	});

	test("ノードラベルが空の場合にノードIDが詳細パネルのタイトルにフォールバック表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// labelを持たない generic ノードをクリック
		await page.locator(".react-flow__node-generic").first().click();

		// 詳細パネルにIDである「node-generic」が表示されていることを確認
		await expect(page.locator("text=Node Detail: node-generic")).toBeVisible();
		await expect(
			page.locator("text=No additional details available."),
		).toBeVisible();
	});

	test("ノードデータが存在しない詳細画面でToken Log行をクリックした際のガード処理検証", async ({
		page,
	}) => {
		// nodesがundefinedの sess-002 に遷移
		await page.locator("text=sess-002").click();

		// Token Count Log テーブルの行をクリック
		await page.locator("tr[class*='tokenRow']").first().click();

		// nodesが無いため、詳細パネル（Node Detail:）が表示されない（ガード処理が成功）ことを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();
	});

	test("セッションIDなしの詳細ルートではエラー表示にフォールバックすること", async ({
		page,
	}) => {
		await page.goto("/#/sessions");

		await expect(page.locator("text=Session ID is missing.")).toBeVisible();
		await page.locator("button:has-text('Back to List')").click();
		await expect(page).toHaveURL(/.*#\//);
	});

	test("token_counts がない詳細画面でも右パネルは表示され、Token Count Log は出ないこと", async ({
		page,
	}) => {
		await page.goto("/#/sessions/sess-003-uuid-long-name");

		await expect(page.locator("text=Session Analytics")).toBeVisible();
		await expect(page.locator("text=Turn #1")).toBeVisible();
		await expect(page.locator("text=Token Count Log")).not.toBeVisible();
	});

	test("turns が未定義の詳細画面でも右パネルがクラッシュしないこと", async ({
		page,
	}) => {
		await page.goto("/#/sessions/sess-no-turns");

		await expect(page.locator("text=Session Analytics")).toBeVisible();
		await expect(
			page.locator("text=Turn Duration (seconds)"),
		).not.toBeVisible();
		await expect(page.locator("text=Turn #1")).not.toBeVisible();
	});
});
