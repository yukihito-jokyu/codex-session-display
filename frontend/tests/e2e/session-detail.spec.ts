import { expect, test } from "./helpers/coverage";
import { mockWailsAPI } from "./helpers/mock-wails";

test.beforeEach(async ({ page }) => {
	// Wails APIのモックを注入
	await mockWailsAPI(page);

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

	test("左側に会話タイムラインとターン・トークン情報が常時表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		await expect(timeline).toBeVisible();
		await expect(timeline.getByText("ターン外イベント")).toBeVisible();
		await expect(
			timeline.getByText("Conversation before the turn"),
		).toBeVisible();
		await expect(timeline.getByText("ターン 1")).toBeVisible();
		await expect(timeline.getByText("5秒")).toBeVisible();
		await expect(timeline.getByText("50K tokens")).toHaveCount(2);
		await expect(timeline.getByText("累計: 50K")).toBeVisible();
		await expect(
			timeline.getByText("Hello, agent, please help me."),
		).toBeVisible();
		await expect(timeline.getByText("Here is how to solve...")).toBeVisible();

		await expect(page.locator(".react-flow")).toBeVisible();
		await expect(page.locator("text=Session Analytics")).toBeVisible();
	});

	test("タイムラインの本文と要約を検索し、元のターン境界を維持すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		const search = timeline.getByLabel("タイムラインを検索");

		await search.fill("Conversation before the turn");
		await expect(timeline.getByText("ターン外イベント")).toBeVisible();
		await expect(
			timeline.getByText("Conversation before the turn"),
		).toBeVisible();
		await expect(timeline.getByText("ターン 1")).toHaveCount(0);

		await search.fill("focused test");
		await expect(timeline.getByText("ターン外イベント")).toHaveCount(0);
		await expect(timeline.getByText("ターン 1")).toBeVisible();
		await expect(
			timeline.getByRole("button", { name: /コマンド実行/ }),
		).toBeVisible();
	});

	test("検索キーワードに不一致の時に空状態を表示すること", async ({ page }) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		await timeline.getByLabel("タイムラインを検索").fill("nonexistent keyword");
		await expect(timeline.getByText("ターン 1")).toHaveCount(0);
		await expect(
			timeline.getByText("条件に一致するタイムライン項目はありません"),
		).toBeVisible();
	});

	test("左タイムラインが初期幅30%で表示され、幅変更用separatorを持つこと", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const viewportWidth = page.viewportSize()?.width;
		expect(viewportWidth).toBeDefined();

		const timeline = page.getByTestId("conversation-timeline");
		const timelineBox = await timeline.boundingBox();
		expect(timelineBox).not.toBeNull();
		expect(timelineBox?.width).toBeCloseTo((viewportWidth ?? 0) * 0.3, 0);

		const resizer = page.getByTestId("timeline-resizer");
		await expect(resizer).toHaveAttribute("role", "separator");
		await expect(resizer).toHaveAttribute("aria-orientation", "vertical");
		await expect(resizer).toHaveAttribute(
			"aria-label",
			"タイムラインの幅を変更",
		);
		await expect(resizer).toHaveAttribute("aria-valuemin", "320");
		await expect(resizer).toHaveAttribute(
			"aria-valuemax",
			String(Math.floor((viewportWidth ?? 0) * 0.5)),
		);
		await expect(resizer).toHaveAttribute(
			"aria-valuenow",
			String(Math.floor((viewportWidth ?? 0) * 0.3)),
		);
		await expect(resizer).toHaveAttribute("tabindex", "0");
	});

	test("左タイムラインの幅を矢印キーで変更し、上下限に収められること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const viewportWidth = page.viewportSize()?.width ?? 0;
		const timeline = page.getByTestId("conversation-timeline");
		const resizer = page.getByTestId("timeline-resizer");
		await resizer.focus();

		const initialWidth = (await timeline.boundingBox())?.width ?? 0;
		await resizer.press("ArrowRight");
		expect((await timeline.boundingBox())?.width).toBe(initialWidth + 16);
		await expect(resizer).toHaveAttribute(
			"aria-valuenow",
			String(initialWidth + 16),
		);

		for (let index = 0; index < 100; index += 1) {
			await resizer.press("ArrowLeft");
		}
		expect((await timeline.boundingBox())?.width).toBe(320);
		await expect(resizer).toHaveAttribute("aria-valuenow", "320");

		for (let index = 0; index < 100; index += 1) {
			await resizer.press("ArrowRight");
		}
		const maxWidth = Math.floor(viewportWidth * 0.5);
		expect((await timeline.boundingBox())?.width).toBe(maxWidth);
		await expect(resizer).toHaveAttribute("aria-valuenow", String(maxWidth));
	});

	test("左タイムラインの境界をドラッグして幅を変更し、上下限に収められること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const viewportWidth = page.viewportSize()?.width ?? 0;
		const timeline = page.getByTestId("conversation-timeline");
		const resizer = page.getByTestId("timeline-resizer");
		const resizerBox = await resizer.boundingBox();
		expect(resizerBox).not.toBeNull();

		const startX = (resizerBox?.x ?? 0) + (resizerBox?.width ?? 0) / 2;
		const startY = (resizerBox?.y ?? 0) + (resizerBox?.height ?? 0) / 2;
		const initialWidth = (await timeline.boundingBox())?.width ?? 0;

		await page.mouse.move(startX, startY);
		await page.mouse.down();
		await page.mouse.move(startX + 80, startY);
		await page.mouse.up();
		expect((await timeline.boundingBox())?.width).toBe(initialWidth + 80);

		await page.mouse.move(startX + 80, startY);
		await page.mouse.down();
		await page.mouse.move(0, startY);
		await page.mouse.up();
		expect((await timeline.boundingBox())?.width).toBe(320);

		const minResizerBox = await resizer.boundingBox();
		const minStartX = (minResizerBox?.x ?? 0) + (minResizerBox?.width ?? 0) / 2;
		await page.mouse.move(minStartX, startY);
		await page.mouse.down();
		await page.mouse.move(viewportWidth, startY);
		await page.mouse.up();
		expect((await timeline.boundingBox())?.width).toBe(
			Math.floor(viewportWidth * 0.5),
		);
	});

	test("変更した左タイムライン幅が別セッションと再読み込み後に復元されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		const resizer = page.getByTestId("timeline-resizer");
		await resizer.focus();
		await resizer.press("ArrowRight");
		await resizer.press("ArrowRight");
		const savedWidth = (await timeline.boundingBox())?.width ?? 0;

		await expect
			.poll(() =>
				page.evaluate(() =>
					localStorage.getItem("session-detail.timeline-width"),
				),
			)
			.toBe(String(savedWidth));

		await page.goto("/#/sessions/sess-002-uuid-long-name");
		await expect(page.getByTestId("conversation-timeline")).toHaveCSS(
			"width",
			`${savedWidth}px`,
		);

		await page.reload();
		await expect(page.getByTestId("conversation-timeline")).toHaveCSS(
			"width",
			`${savedWidth}px`,
		);
	});

	test("viewport縮小時に左タイムライン幅を再調整し、既存パネルを操作できること", async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1600, height: 900 });
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		const resizer = page.getByTestId("timeline-resizer");
		expect((await timeline.boundingBox())?.width).toBe(480);

		await page.setViewportSize({ width: 1400, height: 900 });
		expect((await timeline.boundingBox())?.width).toBe(480);
		await expect(resizer).toHaveAttribute("aria-valuemax", "700");

		await page.setViewportSize({ width: 1600, height: 900 });
		await resizer.focus();
		for (let index = 0; index < 100; index += 1) {
			await resizer.press("ArrowRight");
		}
		expect((await timeline.boundingBox())?.width).toBe(800);

		await page.setViewportSize({ width: 1400, height: 900 });
		await expect
			.poll(async () => (await timeline.boundingBox())?.width)
			.toBe(700);
		await expect(resizer).toHaveAttribute("aria-valuemax", "700");
		await expect(resizer).toHaveAttribute("aria-valuenow", "700");

		await expect(page.locator(".react-flow")).toBeVisible();
		await expect(page.getByText("Session Analytics")).toBeVisible();
		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.getByTestId("bottom-panel")).toBeVisible();
	});

	test("長いイベント内容を省略し、全文を下部詳細パネルで表示できること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		await timeline.getByRole("button", { name: /コマンド実行/ }).click();

		await expect(timeline.getByText("LONG_COMMAND_OUTPUT_END")).toHaveCount(0);
		await timeline.getByRole("button", { name: "全文を表示" }).click();

		const detail = page.getByTestId("bottom-panel-node-detail");
		await expect(page.getByText("Node Detail: コマンド実行")).toBeVisible();
		await expect(detail.getByText("LONG_COMMAND_OUTPUT_END")).toBeVisible();
	});

	test("短いイベント内容は省略せず、全文表示操作を出さないこと", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		const collab = timeline
			.locator("article")
			.filter({ hasText: "Subagent Activity" });
		await collab.getByRole("button", { name: /Subagent Activity/ }).click();

		await expect(collab.getByText("Subagent session initiated")).toBeVisible();
		await expect(
			collab.getByRole("button", { name: "全文を表示" }),
		).toHaveCount(0);
	});

	test("下部詳細パネルで全文を検索し、一致箇所を確認できること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		await timeline.getByRole("button", { name: /コマンド実行/ }).click();
		await timeline.getByRole("button", { name: "全文を表示" }).click();

		const detail = page.getByTestId("bottom-panel-node-detail");
		const search = detail.getByLabel("全文を検索");
		await search.fill("search_needle");
		await expect(detail.locator("mark")).toHaveCount(3);
		await expect(detail.getByText("3件", { exact: true })).toBeVisible();

		await search.fill("not_found_in_full_text");
		await expect(detail.locator("mark")).toHaveCount(0);
		await expect(detail.getByText("0件", { exact: true })).toBeVisible();
	});

	test("タイムライン全文表示後にキャンバスノードの詳細へ切り替えられること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timeline = page.getByTestId("conversation-timeline");
		await timeline.getByRole("button", { name: /コマンド実行/ }).click();
		await timeline.getByRole("button", { name: "全文を表示" }).click();
		await expect(page.getByText("Node Detail: コマンド実行")).toBeVisible();

		await page.locator(".react-flow__node-userMessage").click();
		await expect(page.getByText("Node Detail: User Message")).toBeVisible();
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
			page
				.getByTestId("bottom-panel-node-detail")
				.getByText("Hello, agent, please help me."),
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

		// User Message ノードに紐付く2件の消費量合計（20K + 30K）を確認
		const badge = page.locator(".react-flow__node-userMessage >> text=50K ×2");
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
		await expect(page.getByText("Session Analytics")).toBeVisible();

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

	test("左右分割表示の境界を矢印キーで変更でき、separator属性が更新されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		await page
			.getByRole("button", { name: "User Message token badge" })
			.click();

		const resizer = page.getByTestId("bottom-panel-resizer");
		await expect(resizer).toHaveAttribute("role", "separator");
		await expect(resizer).toHaveAttribute("aria-orientation", "vertical");
		await expect(resizer).toHaveAttribute(
			"aria-label",
			"Resize token detail panel",
		);
		await expect(resizer).toHaveAttribute("aria-valuemin", "25");
		await expect(resizer).toHaveAttribute("aria-valuemax", "75");
		await expect(resizer).toHaveAttribute("aria-valuenow", "52");
		await expect(resizer).toHaveAttribute("tabindex", "0");

		await resizer.focus();
		await page.keyboard.press("ArrowRight");
		await expect(resizer).toHaveAttribute("aria-valuenow", "55");

		await page.keyboard.press("ArrowLeft");
		await expect(resizer).toHaveAttribute("aria-valuenow", "52");
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

		const viewport = page.locator(".react-flow__viewport");
		await page.getByRole("button", { name: "zoom out" }).click();
		await page.waitForTimeout(300);
		const beforeTransform = await viewport.getAttribute("style");

		// Token Count Log テーブルの最初の行（classにtokenRowを含む行）をクリック
		await page.locator("tr[class*='tokenRow']").first().click();

		// 詳細パネルが表示されることを確認
		await expect(page.locator("text=Node Detail:")).toBeVisible();

		// 選択されたノード「User Message」の詳細が表示されていることを確認
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
		await expect
			.poll(async () => viewport.getAttribute("style"))
			.not.toBe(beforeTransform);
	});

	test("タイムライン項目を選択するとノード・統計・下部詳細が同じ表示単位として選択されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const timelineItem = page.getByTestId("timeline-item-timeline-turn-1-user");
		await timelineItem.click();

		await expect(timelineItem).toHaveAttribute("aria-selected", "true");
		await expect(
			page.locator('.react-flow__node[data-id="node-user-msg"]'),
		).toHaveClass(/selected/);
		await expect(page.locator('[data-testid="token-row-0"]')).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(page.locator('[data-testid="token-row-1"]')).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(page.getByTestId("last-token-point-0")).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(page.getByTestId("last-token-point-1")).toHaveAttribute(
			"data-selected",
			"true",
		);
		for (const series of ["input", "output", "reasoning", "cached"]) {
			await expect(
				page.getByTestId(`last-token-point-${series}-0`),
			).toHaveAttribute("data-selected", "true");
			await expect(
				page.getByTestId(`last-token-point-${series}-1`),
			).toHaveAttribute("data-selected", "true");
		}
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("統計行から選択すると対応タイムラインへスクロールして表示単位全体を選択すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		await page.evaluate(() => {
			const testWindow = window as Window & {
				__timelineScrollTarget?: string | null;
			};
			testWindow.__timelineScrollTarget = null;
			Element.prototype.scrollIntoView = function scrollIntoView() {
				testWindow.__timelineScrollTarget = this.getAttribute("data-testid");
			};
		});

		await page.getByTestId("token-row-0").click();

		await expect(
			page.getByTestId("timeline-item-timeline-turn-1-user"),
		).toHaveAttribute("aria-selected", "true");
		await expect(page.getByTestId("token-row-1")).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect
			.poll(() =>
				page.evaluate(
					() =>
						(
							window as Window & {
								__timelineScrollTarget?: string | null;
							}
						).__timelineScrollTarget,
				),
			)
			.toBe("timeline-item-timeline-turn-1-user");
	});

	test("統計から選択しても右パネルのスクロール位置を変更しないこと", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		const rightPanel = page
			.getByRole("heading", { name: "Session Analytics" })
			.locator("..");
		const tokenRow = page.getByTestId("token-row-0");
		await tokenRow.scrollIntoViewIfNeeded();
		const beforeScrollTop = await rightPanel.evaluate(
			(element) => element.scrollTop,
		);

		await tokenRow.click();

		await expect
			.poll(() => rightPanel.evaluate((element) => element.scrollTop))
			.toBe(beforeScrollTop);
	});

	test("キャンバスノードから選択すると対応タイムラインと全token_countを選択すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		await page.locator(".react-flow__node-userMessage").click();

		await expect(
			page.getByTestId("timeline-item-timeline-turn-1-user"),
		).toHaveAttribute("aria-selected", "true");
		await expect(page.getByTestId("token-row-0")).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(page.getByTestId("token-row-1")).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
	});

	test("選択中のタイムライン項目を再選択すると全領域の選択を解除すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();
		const timelineItem = page.getByTestId("timeline-item-timeline-turn-1-user");

		await timelineItem.click();
		await timelineItem.click();

		await expect(timelineItem).toHaveAttribute("aria-selected", "false");
		await expect(page.getByTestId("token-row-0")).toHaveAttribute(
			"data-selected",
			"false",
		);
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();
	});

	test("TOKEN COUNT LOG の同じ行を連続してクリックしても対象ノードへ再度ズームすること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const viewport = page.locator(".react-flow__viewport");
		const tokenRow = page.locator("tr[class*='tokenRow']").first();

		await tokenRow.click();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
		await page.waitForTimeout(900);

		await page.getByRole("button", { name: "zoom out" }).click();
		await page.waitForTimeout(300);
		const movedTransform = await viewport.getAttribute("style");

		await tokenRow.click();

		await expect
			.poll(async () => viewport.getAttribute("style"))
			.not.toBe(movedTransform);
	});

	test("LAST TOKEN CONSUMPTION PER INDEX チャートをクリックした際に、対応するノードが選択されて詳細パネルが表示されること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		// 最初は詳細パネルが表示されていないことを確認
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		const viewport = page.locator(".react-flow__viewport");
		await page.getByRole("button", { name: "zoom out" }).click();
		await page.waitForTimeout(300);
		const beforeTransform = await viewport.getAttribute("style");

		await page.locator("[data-testid='last-token-point-0']").click();

		// 詳細パネルが表示され、対応するノード（node-user-msg -> User Message）の情報が表示されることを確認
		await expect(page.locator("text=Node Detail:")).toBeVisible();
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
		await expect
			.poll(async () => viewport.getAttribute("style"))
			.not.toBe(beforeTransform);
	});

	test("LAST TOKEN CONSUMPTION PER INDEX の各系列をクリックした際に、対応するノードへ移動すること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const viewport = page.locator(".react-flow__viewport");
		await page.getByRole("button", { name: "zoom out" }).click();
		await page.waitForTimeout(300);
		const beforeTransform = await viewport.getAttribute("style");

		await page.getByTestId("last-token-point-input-0").click();

		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
		await expect
			.poll(async () => viewport.getAttribute("style"))
			.not.toBe(beforeTransform);
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

	test("LAST TOKEN CONSUMPTION PER INDEX にコンテキスト使用率と算出不能表示があること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const chart = page.getByTestId("last-token-chart");
		await expect(chart.locator(".recharts-yAxis")).toHaveCount(2);
		await expect(
			chart.getByText("Context Usage (%)", { exact: true }),
		).toBeVisible();
		await expect(page.getByTestId("last-token-point-context-0")).toBeVisible();
		await expect(page.getByTestId("last-token-point-context-3")).toHaveCount(0);

		await page.getByTestId("last-token-point-3").hover();
		await expect(page.getByTestId("last-token-tooltip-context")).toHaveText(
			"Context Usage (%): N/A",
		);
	});

	test("コンテキスト使用率はキャッシュ入力を差し引かず100%超過値のまま共通選択できること", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const contextPoint = page.getByTestId("last-token-point-context-0");
		await contextPoint.hover();
		await expect(page.getByTestId("last-token-tooltip-context")).toHaveText(
			"Context Usage (%): 150%",
		);

		await contextPoint.click();
		await expect(contextPoint).toHaveAttribute("data-selected", "true");
		await expect(page.getByTestId("token-row-0")).toHaveAttribute(
			"data-selected",
			"true",
		);
		await expect(
			page.getByTestId("timeline-item-timeline-turn-1-user"),
		).toHaveAttribute("aria-selected", "true");
		await expect(page.locator("text=Node Detail: User Message")).toBeVisible();
		await expect(page.getByTestId("bottom-panel-token-detail")).toBeVisible();
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

	test("紐付け先がないトークン項目は表とチャートで操作対象にならないこと", async ({
		page,
	}) => {
		await page.locator("text=sess-001").click();

		const missingNodeRow = page.locator("tbody tr").filter({
			has: page.locator("td:first-child", { hasText: /^4$/ }),
		});
		const unboundRow = page.locator("tbody tr").filter({
			has: page.locator("td:first-child", { hasText: /^5$/ }),
		});

		await expect(missingNodeRow).not.toHaveAttribute("role", "button");
		await expect(missingNodeRow).not.toHaveAttribute("tabindex", "0");
		await expect(unboundRow).not.toHaveAttribute("role", "button");
		await expect(unboundRow).not.toHaveAttribute("tabindex", "0");

		await missingNodeRow.click();
		await unboundRow.click();
		await expect(page.locator("text=Node Detail:")).not.toBeVisible();

		const validChartPoint = page.locator("[data-testid='last-token-point-0']");
		const missingNodeChartPoint = page.locator(
			"[data-testid='last-token-point-4']",
		);
		const unboundChartPoint = page.locator(
			"[data-testid='last-token-point-5']",
		);

		await expect(validChartPoint).toHaveAttribute("role", "button");
		await expect(missingNodeChartPoint).not.toHaveAttribute("role", "button");
		await expect(unboundChartPoint).not.toHaveAttribute("role", "button");
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

		// 50K *2 (node-user-msg, consumedTokens=50000, boundCount=2)
		const userBadge = page.locator(
			".react-flow__node-userMessage >> text=50K ×2",
		);
		await expect(userBadge).toBeVisible();

		// last_token_usage 欠落時は0として表示し、件数には含める
		const apiBadge = page.locator(".react-flow__node-userApiMessage >> text=0");
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

		// 右パネルのセッション累計は従来どおり statistics.total_tokens を表示する
		await expect(
			page.getByText("150,000", { exact: true }).first(),
		).toBeVisible();
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
		await page.goto("/#/sessions/sess-002-uuid-long-name");

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
		await page.goto("/#/sessions/sess-001-uuid-long-name");

		// labelを持たない generic ノードをクリック
		await page
			.locator(".react-flow__node-generic")
			.first()
			.click({ force: true });

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
		await page.goto("/#/sessions/sess-002-uuid-long-name");

		const tokenRow = page.locator("tbody tr").first();
		await expect(tokenRow).not.toHaveAttribute("role", "button");
		await expect(tokenRow).not.toHaveAttribute("tabindex", "0");
		await tokenRow.click();

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

	test("サブエージェント親子セッション間の遷移（キャンバス、タイムライン、戻る）が動作すること", async ({
		page,
	}) => {
		// 1. 親セッション詳細へ遷移
		await page.goto("/#/sessions/sess-001-uuid-long-name");
		await expect(
			page.locator("text=Session Detail: sess-001-uuid-long-name"),
		).toBeVisible();

		// 2. キャンバスの collabAgent ノードから「サブエージェントを表示」をクリックして子セッション詳細へ遷移
		const collabNode = page.locator(".react-flow__node-collabAgent");
		await expect(collabNode).toBeVisible();
		await collabNode
			.locator("button:has-text('サブエージェントを表示')")
			.click();

		// 子セッション詳細へ遷移したことを確認
		await expect(page).toHaveURL(/.*#\/sessions\/sess-002-uuid-long-name/);
		await expect(
			page.locator("text=Session Detail: sess-002-uuid-long-name"),
		).toBeVisible();

		// 3. 「← Back to Parent」ボタンで親セッション詳細に戻る
		const backToParentBtn = page.locator("button:has-text('← Back to Parent')");
		await expect(backToParentBtn).toBeVisible();
		await backToParentBtn.click();

		// 親セッション詳細に戻ったことを確認
		await expect(page).toHaveURL(/.*#\/sessions\/sess-001-uuid-long-name/);
		await expect(
			page.locator("text=Session Detail: sess-001-uuid-long-name"),
		).toBeVisible();

		// 4. タイムライン上の collab イベントから「サブエージェントを表示」をクリックして子セッション詳細へ遷移
		const timeline = page.getByTestId("conversation-timeline");
		const collabTimelineItem = timeline
			.locator("article")
			.filter({ hasText: "Subagent Activity" });
		await expect(collabTimelineItem).toBeVisible();
		// アコーディオンのトグルをクリックして展開
		await collabTimelineItem
			.getByRole("button", { name: "Subagent Activity" })
			.click();
		// 「サブエージェントを表示」をクリック
		await collabTimelineItem
			.locator("button:has-text('サブエージェントを表示')")
			.click();

		// 再び子セッション詳細へ遷移したことを確認
		await expect(page).toHaveURL(/.*#\/sessions\/sess-002-uuid-long-name/);
		await expect(
			page.locator("text=Session Detail: sess-002-uuid-long-name"),
		).toBeVisible();
	});
});
