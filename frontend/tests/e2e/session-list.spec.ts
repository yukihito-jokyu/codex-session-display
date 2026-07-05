import { expect, test } from "./helpers/coverage";
import { mockWailsAPI } from "./helpers/mock-wails";

test.beforeEach(async ({ page }) => {
	// Wails APIのモックを注入
	await mockWailsAPI(page);

	await page.goto("/");
});

test.describe("セッション一覧画面 E2E テスト", () => {
	test("初期ロード時に2026年5月のセッション一覧が正しく表示されること", async ({
		page,
	}) => {
		// 月ナビゲーターの初期表示が「2026年 5月」であることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();

		// セッション1が表示されていることを確認
		await expect(page.locator("text=sess-001")).toBeVisible();
		// 子セッションであるセッション2は初期状態で非表示であることを確認
		await expect(page.locator("text=sess-002")).not.toBeVisible();
		// 4月のセッション3は表示されていないことを確認
		await expect(page.locator("text=sess-003")).not.toBeVisible();
	});

	test("Codex / Claude Code を切り替えて一覧表示できること", async ({
		page,
	}) => {
		await expect(page.getByRole("button", { name: "Codex" })).toBeVisible();
		await expect(
			page.getByRole("button", { name: "Claude Code" }),
		).toBeVisible();
		await expect(page.locator("text=~/.codex/sessions")).toBeVisible();

		await page.getByRole("button", { name: "Claude Code" }).click();

		await expect(page.locator("text=~/.claude/projects")).toBeVisible();
		await expect(page.locator("text=claude-s")).toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();

		const calls = await page.evaluate(() => {
			return (
				window as unknown as {
					__listSessionsCalls: Array<{ provider: string }>;
				}
			).__listSessionsCalls;
		});
		expect(calls.some((call) => call.provider === "claude")).toBe(true);

		await page.getByRole("button", { name: "Codex" }).click();
		await expect(page.locator("text=sess-001")).toBeVisible();
	});

	test("サブエージェントセッションが親セッションの下にネストして表示され、展開・折りたたみ可能であること", async ({
		page,
	}) => {
		// 初期状態では親セッションのみが表示され、子セッションは非表示
		await expect(page.locator("text=sess-001")).toBeVisible();
		await expect(page.locator("text=sess-002")).not.toBeVisible();

		// 展開ボタン（▶）が存在することを確認
		const expandBtn = page.locator('button[aria-label="Expand subagents"]');
		await expect(expandBtn).toBeVisible();

		// 展開ボタンをクリック
		await expandBtn.click();

		// 子セッションが表示されることを確認
		await expect(page.locator("text=sess-002")).toBeVisible();

		// 折りたたみボタン（▼）をクリック
		const collapseBtn = page.locator('button[aria-label="Collapse subagents"]');
		await expect(collapseBtn).toBeVisible();
		await collapseBtn.click();

		// 再び非表示になることを確認
		await expect(page.locator("text=sess-002")).not.toBeVisible();
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
		await expect(page.locator("text=sess-002")).not.toBeVisible();
	});

	test("検索バーによるフィルタリングが正しく機能すること", async ({ page }) => {
		// 検索入力欄を取得して「openai」と入力
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("openai");

		// openai의モデルプロバイダを持つsession-2のみが表示され、session-1が消えることを確認
		await expect(page.locator("text=sess-002")).toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();
	});

	test("未解析セッションが正しく表示されること", async ({ page }) => {
		// 2026年5月の未解析セッション「sess-004」のIDが表示されていることを確認
		await expect(page.locator("text=sess-004")).toBeVisible();

		// 「未解析」バッジが表示されていることを確認
		await expect(page.locator("span:has-text('未解析')")).toBeVisible();

		// プレースホルダーの「解析前」テキストが表示されていることを確認
		await expect(page.locator("text=解析前").first()).toBeVisible();
	});

	test("アコーディオン（日ノード）の開閉動作が正しく機能すること", async ({
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
	});

	test("エラー発生時のエラー表示とクエリクリアによる復帰が正しく機能すること", async ({
		page,
	}) => {
		// 検索入力欄に「trigger-error」と入力してエラーを発生させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("trigger-error");

		// エラーメッセージが表示されることを確認
		await expect(
			page.locator("text=Failed to fetch sessions: Mocked API Error"),
		).toBeVisible();
		await expect(page.locator("text=Retry")).toBeVisible();

		// 検索入力欄をクリアする
		await searchInput.fill("");

		// エラーメッセージが消え、セッション1が表示される（復帰）ことを確認
		await expect(
			page.locator("text=Failed to fetch sessions:"),
		).not.toBeVisible();
		await expect(page.locator("text=sess-001")).toBeVisible();
	});

	test("検索結果が0件の場合に空表示が表示されること", async ({ page }) => {
		// 検索入力欄に存在しないセッションの条件を入力
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("nonexistent-query-string");

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

	test("セッション一覧が空の場合に現在の年月がデフォルト値になること", async ({
		page,
	}) => {
		// Wails APIのモックを空配列で再初期化
		await mockWailsAPI(page, []);
		// goto "/" to trigger initial load (year=0, month=0)
		await page.goto("/");

		// 月ナビゲーターが現在の年月になっていることを確認
		const now = new Date();
		const expectedLabel = `${now.getFullYear()}年 ${now.getMonth() + 1}月`;
		await expect(
			page.locator(`span:has-text('${expectedLabel}')`),
		).toBeVisible();
	});

	test("タイムスタンプ例外発生時のフォールバック表示検証", async ({ page }) => {
		// getHoursが1999年の場合にエラーをスローするようスクリプト注入
		await page.addInitScript(() => {
			const originalGetHours = Date.prototype.getHours;
			Date.prototype.getHours = function () {
				if (this.getFullYear() === 1999) {
					throw new Error("mock timestamp error");
				}
				return originalGetHours.call(this);
			};
		});

		// timestamp-error を検索して1999年のセッションを表示させる
		await page.goto("/");
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("timestamp-error");

		// 例外をキャッチした結果、元のISO文字列 (1999-05-20T10:00:00Z) がそのまま表示されていることを確認
		await expect(page.locator("text=1999-05-20T10:00:00Z")).toBeVisible();
	});

	test("日ノード、セッション行のキーボード操作（Enter/Space）検証", async ({
		page,
	}) => {
		// 初期状態ではセッションが表示されている
		await expect(page.locator("text=sess-001")).toBeVisible();

		// 日ノード「20日」のヘッダーにフォーカスを当てて、Enterキーを押して閉じる
		const dayHeader = page.locator("div[role='button']:has-text('20日')");
		await dayHeader.focus();
		await page.keyboard.press("Enter");
		await expect(page.locator("text=sess-001")).not.toBeVisible();

		// 再度スペースキーを押して展開する
		await page.keyboard.press("Space");
		await expect(page.locator("text=sess-001")).toBeVisible();

		// セッション行にフォーカスを当てて、Spaceキーを押して詳細画面に遷移することを確認 (Space key / line 68 coverage)
		const sessionRow = page
			.locator("div[role='button']:has-text('sess-001')")
			.first();
		await sessionRow.focus();
		await page.keyboard.press("Space");
		await expect(page).toHaveURL(/.*#\/sessions\/sess-001-uuid-long-name/);
	});

	test("エラー画面でのリトライボタンクリック時に再読み込みが機能すること", async ({
		page,
	}) => {
		// 検索入力欄に「trigger-error」を入力してエラーを発生させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("trigger-error");
		await expect(
			page.locator("text=Failed to fetch sessions: Mocked API Error"),
		).toBeVisible();

		// listSessionsの呼び出し履歴をクリアする
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			(window as any).__listSessionsCalls = [];
		});

		// 「Retry」ボタンをクリック
		await page.locator("button:has-text('Retry')").click();

		// listSessionsが再実行され、1回の呼び出しが記録されていることを確認
		const callsCount = await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			return (window as any).__listSessionsCalls?.length || 0;
		});
		expect(callsCount).toBe(1);
	});

	test("デバウンス検索中の画面遷移によるアンマウント時にクリーンアップされること", async ({
		page,
	}) => {
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		// 文字を入力してデバウンスのsetTimeoutを開始
		await searchInput.fill("sess");

		// すぐにセッション行をクリックして詳細画面に遷移（一覧画面とToolbarをアンマウント）
		await page.locator("text=sess-001").click();
		await expect(page).toHaveURL(/.*#\/sessions\/sess-001-uuid-long-name/);
	});

	test("複数日付にまたがるセッション表示とソートの検証", async ({ page }) => {
		// multi-date を検索して全セッションを表示させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("multi-date");

		// 複数日、および同一日の複数セッションが表示されていることを検証
		await expect(
			page.locator("div[class*='dayNode'] >> text=20日"),
		).toBeVisible();
		await expect(
			page.locator("div[class*='dayNode'] >> text=19日"),
		).toBeVisible();
	});

	test("解析済セッションの空フィールドフォールバック表示検証", async ({
		page,
	}) => {
		// null-fields セッションを表示させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("sess-007-null-fields");

		// null-fields セッションの行が表示されていることを確認
		await expect(page.locator("text=sess-007")).toBeVisible();

		// 各種フィールドが空のためフォールバック文字「—」が表示されていることを確認
		const fallbackCount = await page.locator("text=—").count();
		expect(fallbackCount).toBeGreaterThanOrEqual(2);
	});

	test("リストフェッチ文字列エラー発生時の表示検証", async ({ page }) => {
		// trigger-string-error を検索して文字列例外を発生させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("trigger-string-error");

		// エラーメッセージとして文字列自体が表示されていることを確認
		await expect(
			page.locator("text=Failed to fetch sessions: Mocked List String Error"),
		).toBeVisible();
	});

	test("APIからundefinedが返された場合のフォールバック検証", async ({
		page,
	}) => {
		// return-undefined を検索して undefined を返させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("return-undefined");

		// エラーにはならず、空リストメッセージが表示されていることを確認
		await expect(
			page.locator("text=No sessions found matching filters."),
		).toBeVisible();
	});

	test("ローディング中にナビゲーションを強制クリックした際のガード処理検証", async ({
		page,
	}) => {
		// 初期ロードをハングさせるフラグを設定
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			(window as any).__hangInitialLoad = true;
		});

		// ページロード
		await page.goto("/");

		// ローディング中であることを確認
		await expect(page.locator("text=Scanning session logs...")).toBeVisible();

		// ナビゲーションの「◀」と「▶」ボタンを `{ force: true }` でクリックして、
		// currentYear/currentMonthがnullの状態のガード句を通す
		const prevBtn = page.locator("button:has-text('◀')");
		const nextBtn = page.locator("button:has-text('▶')");

		await prevBtn.click({ force: true });
		await nextBtn.click({ force: true });

		// エラーなくローディング状態が維持されていることを確認
		await expect(page.locator("text=Scanning session logs...")).toBeVisible();
	});

	test("未定義タイムスタンプと無効な日付形式のフォールバック表示検証", async ({
		page,
	}) => {
		// test-timestamp-fallbacks を検索してモックデータを表示させる
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("test-timestamp-fallbacks");

		// undefined タイムスタンプが「—」と表示されていることを確認
		// invalid タイムスタンプが「invalid-date-format-string」のまま表示されていることを確認
		await expect(page.locator("text=sess-008")).toBeVisible();
		await expect(page.locator("text=sess-009")).toBeVisible();
		await expect(
			page.locator("div[role='button']:has-text('sess-008') >> text=—"),
		).toBeVisible();
		await expect(
			page.locator(
				"div[role='button']:has-text('sess-009') >> text=invalid-date-format-string",
			),
		).toBeVisible();
	});

	test("グルーピング不能な日付しかない場合は空表示にフォールバックすること", async ({
		page,
	}) => {
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("all-invalid-grouping-dates");

		await expect(
			page.locator("text=No sessions found matching filters."),
		).toBeVisible();
	});

	test("ファイルサイズ 0 バイトのセッションが 0 B と表示されること", async ({
		page,
	}) => {
		const searchInput = page.getByPlaceholder(
			"Filter by ID, CWD, branch, provider...",
		);
		await searchInput.fill("zero-size");

		await expect(page.locator("text=sess-010")).toBeVisible();
		await expect(
			page.locator("div[role='button']:has-text('sess-010') >> text=0 B"),
		).toBeVisible();
	});

	test("初期表示エラー時のリトライ処理検証", async ({ page }) => {
		// 初期ロード時にエラーを発生させるフラグを設定
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			(window as any).__triggerInitialError = true;
		});

		// ページロード
		await page.goto("/");

		// 初期ロード時のエラーメッセージとリトライボタンが表示されていることを確認
		await expect(
			page.locator("text=Failed to fetch sessions: Initial Load Error"),
		).toBeVisible();

		// listSessionsの呼び出し履歴をクリア
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			(window as any).__listSessionsCalls = [];
		});

		// 「Retry」ボタンをクリック
		await page.locator("button:has-text('Retry')").click();

		// 再試行が走り、1回の呼び出しが記録されていることを確認
		const callsCount = await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock field on window
			return (window as any).__listSessionsCalls?.length || 0;
		});
		expect(callsCount).toBe(1);
	});

	test("アコーディオン展開時に、未解析セッションのパースが自動的に開始され、ステータスと情報が更新されること", async ({
		page,
	}) => {
		// 初期状態では展開されてパースされてしまうため、一度折りたたんだ状態で起動する
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
		});

		// このテスト用の成功モックを設定
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const App = (window as any).go.main.App;
			App.GetSessionDetail = async (id: string) => {
				if (id === "sess-004-unparsed-session") {
					// 「解析中...」の表示時間を確保するため意図的に少し遅延させる
					await new Promise((resolve) => setTimeout(resolve, 1000));

					// dummySessions 内のステータスを更新
					// biome-ignore lint/suspicious/noExplicitAny: mock
					const session = ((window as any).__dummySessions as any[])?.find(
						(s) => s.id === id,
					);
					if (session) {
						session.parsed = true;
						session.cwd = "/Users/test/projects/unparsed-app";
						session.branch = "feature/unparsed";
						session.model_provider = "anthropic";
						session.cli_version = "1.0.0";
					}

					return {
						id: id,
						cache_schema_version: 3,
						parsed_at: "2026-05-20T14:00:00Z",
						nodes: [
							{
								id: "node-meta",
								type: "sessionMeta",
								position: { x: 0, y: 0 },
								data: {
									category: "meta",
									label: "Session Meta",
									icon: "⚙️",
									summary: "CLI Version: 1.0.0",
									turnIndex: -1,
									meta: {
										version: "1.0.0",
										cwd: "/Users/test/projects/unparsed-app",
										git_branch: "feature/unparsed",
										model_provider: "anthropic",
										cli_version: "1.0.0",
									},
								},
							},
						],
						edges: [],
						statistics: {
							duration_ms: 1000,
							total_tokens: 0,
							tool_call_count: 0,
							token_count_count: 0,
							context_window_size: 0,
							turn_count: 0,
							turns: [],
						},
						token_counts: [],
						timeline: [],
					};
				}
				throw new Error("Not mocked for this ID");
			};
		});

		await page.goto("/");

		// 「20日」のヘッダーをクリックして展開する
		const dayHeader = page.locator("div[role='button']:has-text('20日')");
		await dayHeader.click();

		// ローディングインジケータ「解析中...」または「解析中」バッジが表示されることを確認 (strict mode 回避のため first() を使用)
		const parsingRow = page
			.locator("div[role='button']:has-text('sess-004')")
			.first();
		await expect(parsingRow.locator("text=解析中...").first()).toBeVisible();
		await expect(parsingRow.locator("text=解析中").first()).toBeVisible();

		// パース完了（200ms遅延後）を待ち、CWDやブランチ名が反映されることを確認
		await expect(parsingRow.locator("text=feature/unparsed")).toBeVisible();
		await expect(
			parsingRow.locator('text="/Users/test/projects/unparsed-app"'),
		).toBeVisible();
		await expect(
			parsingRow.locator("text=解析中...").first(),
		).not.toBeVisible();
		await expect(parsingRow.locator("text=未解析").first()).not.toBeVisible();
	});

	test("パースの同時実行数が最大3件に制御されること", async ({ page }) => {
		// 5件の未解析セッションを同じ日に用意
		const testSessions = [
			{
				id: "sess-p1",
				file_path: "/path/to/p1",
				cwd: undefined,
				cli_version: undefined,
				originator: "user-1",
				model_provider: undefined,
				branch: undefined,
				source: "cli",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-p2",
				file_path: "/path/to/p2",
				cwd: undefined,
				cli_version: undefined,
				originator: "user-1",
				model_provider: undefined,
				branch: undefined,
				source: "cli",
				timestamp: "2026-05-20T10:01:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-p3",
				file_path: "/path/to/p3",
				cwd: undefined,
				cli_version: undefined,
				originator: "user-1",
				model_provider: undefined,
				branch: undefined,
				source: "cli",
				timestamp: "2026-05-20T10:02:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-p4",
				file_path: "/path/to/p4",
				cwd: undefined,
				cli_version: undefined,
				originator: "user-1",
				model_provider: undefined,
				branch: undefined,
				source: "cli",
				timestamp: "2026-05-20T10:03:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-p5",
				file_path: "/path/to/p5",
				cwd: undefined,
				cli_version: undefined,
				originator: "user-1",
				model_provider: undefined,
				branch: undefined,
				source: "cli",
				timestamp: "2026-05-20T10:04:00Z",
				file_size: 100,
				parsed: false,
			},
		];

		// Wails APIモックを注入し、GetSessionDetail呼び出しの同時実行数を追跡する仕組みを構築
		await mockWailsAPI(page, testSessions);
		await page.addInitScript(() => {
			let activeCount = 0;
			let maxConcurrentSeen = 0;

			// GetSessionDetail をオーバーライドして同時実行数を計測
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const originalGetSessionDetail = (window as any).go.main.App
				.GetSessionDetail;
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).go.main.App.GetSessionDetail = async (id: string) => {
				activeCount++;
				if (activeCount > maxConcurrentSeen) {
					maxConcurrentSeen = activeCount;
				}
				// biome-ignore lint/suspicious/noExplicitAny: mock
				(window as any).__maxConcurrentSeen = maxConcurrentSeen;

				// 各パース呼び出しに遅延を持たせる
				await new Promise((resolve) => setTimeout(resolve, 100));
				activeCount--;

				return originalGetSessionDetail(id);
			};
		});

		// 画面へ遷移（20日ノードが自動展開され、全パースがトリガーされる）
		await page.goto("/");

		// 全て完了するまで待つ（100ms * 2回分以上の時間＝最大500ms程度）
		await expect(
			page.locator("text=sess-p1 >> text=解析中...").first(),
		).not.toBeVisible();
		await expect(
			page.locator("text=sess-p5 >> text=解析中...").first(),
		).not.toBeVisible();

		// 最大同時実行数が3以下であることを検証
		const maxConcurrent = await page.evaluate(
			// biome-ignore lint/suspicious/noExplicitAny: mock
			() => (window as any).__maxConcurrentSeen || 0,
		);
		expect(maxConcurrent).toBeGreaterThan(0);
		expect(maxConcurrent).toBeLessThanOrEqual(3);
	});

	test("アップデートがある場合にヘッダーにバッジが表示され、クリックで確認ダイアログが開き、アップデート適用と進捗推移が正しく行われること", async ({
		page,
	}) => {
		// アップデートあり状態のモックを注入
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__mockUpdateResult = {
				hasUpdate: true,
				current: "1.0.0",
				latest: "1.1.0",
				releaseUrl: "https://github.com/owner/repo/releases/tag/v1.1.0",
				downloadUrl:
					"https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.zip",
			};
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__triggerMockProgress = false; // 自動送信は無効化
		});

		// ページ再ロード
		await page.goto("/");

		// ヘッダー（Toolbar）に「🚀 新バージョンがあります (v1.1.0)」が表示されていることを確認
		const updateBadge = page.locator(
			"button:has-text('🚀 新バージョンがあります')",
		);
		await expect(updateBadge).toBeVisible();

		// バッジをクリック
		await updateBadge.click();

		// 確認ダイアログ（モーダル）が表示されていることを確認
		await expect(page.locator("text=新しいバージョンがあります")).toBeVisible();
		const dialog = page.locator("div[role='dialog']");
		await expect(dialog.locator("text=v1.0.0")).toBeVisible();
		await expect(dialog.locator("text=v1.1.0")).toBeVisible();

		// 「アップデート」ボタンをクリックしてアップデートを開始
		const updateBtn = page.locator("button:has-text('アップデート')");
		await expect(updateBtn).toBeVisible();
		await updateBtn.click();

		// 適用中タイトルが表示されていることを確認
		await expect(page.locator("text=アップデートを適用中")).toBeVisible();

		// 1. ダウンロード初期状態（progress: 0）
		await expect(
			page.locator("text=最新バージョンをダウンロードしています... 0%"),
		).toBeVisible();

		// 2. downloading (progress: 50.0) を手動で送信
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__emitWailsEvent("update-progress", {
				status: "downloading",
				progress: 50.0,
			});
		});
		await expect(
			page.locator("text=最新バージョンをダウンロードしています... 50%"),
		).toBeVisible();

		// 3. download_complete (progress: 100.0) を手動で送信
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__emitWailsEvent("update-progress", {
				status: "download_complete",
				progress: 100.0,
			});
		});
		await expect(
			page.locator("text=最新バージョンをダウンロードしています... 100%"),
		).toBeVisible();

		// 4. extracting を手動で送信
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__emitWailsEvent("update-progress", {
				status: "extracting",
				progress: 100.0,
			});
		});
		await expect(
			page.locator("text=パッケージを展開しています..."),
		).toBeVisible();

		// 5. restarting を手動で送信
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__emitWailsEvent("update-progress", {
				status: "restarting",
				progress: 100.0,
			});
		});
		await expect(
			page.locator("text=システムを再起動しています..."),
		).toBeVisible();
	});

	test("タブUIによる表示切り替えが正しく機能すること", async ({ page }) => {
		// 初期状態（履歴ツリー）で月ナビゲーターが表示されていることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();
		// 初期状態ではカレンダーUIは表示されていないことを確認
		await expect(page.locator("input[type='date']")).not.toBeVisible();

		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		// 月ナビゲーターが非表示になっていることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).not.toBeVisible();
		// カレンダーUIが表示されていることを確認
		await expect(page.locator("input[type='date']")).toBeVisible();

		// 「履歴ツリー」タブを再びクリック
		await page.locator("button:has-text('履歴ツリー')").click();

		// 再び月ナビゲーターが表示され、カレンダーUIが非表示になることを確認
		await expect(page.locator("span:has-text('2026年 5月')")).toBeVisible();
		await expect(page.locator("input[type='date']")).not.toBeVisible();
	});

	test("カレンダーの日付変更によるセッションのフィルタリングと年月変更連動が機能すること", async ({
		page,
	}) => {
		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		const dateInput = page.locator("input[type='date']");
		await expect(dateInput).toBeVisible();

		// 日付を 2026-05-20 に変更
		await dateInput.fill("2026-05-20");

		// 2026年5月20日のセッション1が表示されていることを確認
		await expect(page.locator("text=sess-001")).toBeVisible();
		// 19日のセッション6が表示されていないことを確認
		await expect(page.locator("text=sess-006")).not.toBeVisible();

		// 日付を 2026-05-19 に変更
		await dateInput.fill("2026-05-19");

		// 19日のセッション6が表示され、20日のセッション1が非表示になることを確認
		await expect(page.locator("text=sess-006")).toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();

		// 異なる月である 2026-04-10 に日付を変更
		await dateInput.fill("2026-04-10");

		// 2026年4月のセッション3が表示されることを確認
		await expect(page.locator("text=sess-003")).toBeVisible();
		// 5月のセッションは表示されないことを確認
		await expect(page.locator("text=sess-006")).not.toBeVisible();
		await expect(page.locator("text=sess-001")).not.toBeVisible();
	});

	test("詳細画面へ遷移して戻った際、タブとカレンダーの日付状態が維持されていること", async ({
		page,
	}) => {
		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		const dateInput = page.locator("input[type='date']");
		// 日付を 2026-05-19 に変更
		await dateInput.fill("2026-05-19");
		await expect(page.locator("text=sess-006")).toBeVisible();

		// セッション行をクリックして詳細画面に遷移
		await page.locator("text=sess-006").click();
		await expect(page).toHaveURL(/.*#\/sessions\/sess-006/);

		// 一覧に戻るボタンをクリックして戻る
		await page.locator("button:has-text('Back to List')").click();

		// 再び一覧画面で「ディレクトリ分類」タブがアクティブであることを確認
		await expect(page.locator("input[type='date']")).toBeVisible();
		// 選択していた日付が 2026-05-19 のままであることを確認
		await expect(dateInput).toHaveValue("2026-05-19");
		// その日のセッションが表示されていることを確認
		await expect(page.locator("text=sess-006")).toBeVisible();
	});

	test("セッション行に統計バーが正しく表示されること (トレーサー弾)", async ({
		page,
	}) => {
		// sess-001 に対応する行の下部の統計バーが表示されていることを検証
		// 表示例: トークン: 12,345 (入力: 8,123 / 出力: 3,022 / 推論: 1,200) | ターン数: 5 | ステップ数: 12
		const statsBar = page
			.locator("div[role='button']:has-text('sess-001')")
			.first();
		await expect(statsBar).toBeVisible();

		// トークン情報が含まれていること
		await expect(statsBar.locator("text=12,345")).toBeVisible();
		await expect(statsBar.locator("text=8,123")).toBeVisible();
		await expect(statsBar.locator("text=3,022")).toBeVisible();
		await expect(statsBar.locator("text=1,200")).toBeVisible();

		// ターン数とステップ数が含まれていること
		await expect(statsBar.locator("text=ターン数: 5")).toBeVisible();
		await expect(statsBar.locator("text=ステップ数: 12")).toBeVisible();
	});

	test("未解析および解析中のセッション行で統計バーが適切なプレースホルダーを示すこと", async ({
		page,
	}) => {
		// 1. 未解析セッション sess-004
		const unparsedRow = page
			.locator("div[role='button']:has-text('sess-004')")
			.first();
		await expect(unparsedRow).toBeVisible();
		// 統計バーの部分に「解析前」と表示されていることを検証
		const unparsedStats = unparsedRow.locator("div[class*='statsBar']");
		await expect(unparsedStats.locator("text=解析前")).toBeVisible();

		// 2. 解析中のセッションの挙動を検証するため、アコーディオン展開テストと同様のモックと遅延を設定
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
		});

		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const App = (window as any).go.main.App;
			App.GetSessionDetail = async (id: string) => {
				if (id === "sess-004-unparsed-session") {
					// 「解析中...」の表示時間を確保するため意図的に少し遅延させる
					await new Promise((resolve) => setTimeout(resolve, 1000));

					// dummySessions 内のステータスと統計情報を更新
					// biome-ignore lint/suspicious/noExplicitAny: mock
					const session = ((window as any).__dummySessions as any[])?.find(
						(s) => s.id === id,
					);
					if (session) {
						session.parsed = true;
						session.total_tokens = 100;
						session.input_tokens = 60;
						session.output_tokens = 30;
						session.reasoning_tokens = 10;
						session.turn_count = 1;
						session.step_count = 2;
					}

					return {
						id: id,
						cache_schema_version: 3,
						parsed_at: "2026-05-20T14:00:00Z",
						nodes: [],
						edges: [],
						statistics: {
							duration_ms: 1000,
							total_tokens: 100,
							tool_call_count: 2,
							token_count_count: 0,
							context_window_size: 0,
							turn_count: 1,
							turns: [
								{
									index: 0,
									consumed_tokens: {
										total_tokens: 100,
										input_tokens: 60,
										output_tokens: 30,
										reasoning_output_tokens: 10,
									},
								},
							],
						},
						token_counts: [],
						timeline: [],
					};
				}
				throw new Error("Not mocked for this ID");
			};
		});

		await page.goto("/");

		// 日ノード「20日」を展開
		const dayHeader = page.locator("div[role='button']:has-text('20日')");
		await dayHeader.click();

		// 「解析中...」が表示されることを検証
		const parsingRow = page
			.locator("div[role='button']:has-text('sess-004')")
			.first();
		const parsingStats = parsingRow.locator("div[class*='statsBar']");
		await expect(parsingStats.locator("text=解析中...")).toBeVisible();

		// 少し待って（1000msの遅延後）、統計情報が表示に切り替わることを検証
		await expect(parsingStats.locator("text=解析中...")).not.toBeVisible();
		await expect(parsingStats.getByText("100", { exact: true })).toBeVisible();
		await expect(parsingStats.getByText("60", { exact: true })).toBeVisible();
		await expect(parsingStats.getByText("30", { exact: true })).toBeVisible();
		await expect(parsingStats.getByText("10", { exact: true })).toBeVisible();
		await expect(parsingStats.locator("text=ターン数: 1")).toBeVisible();
	});

	test("未解析セッションの優先パース（キューの先頭へ追加）が正しく機能すること", async ({
		page,
	}) => {
		// 5件の未解析セッションを用意
		const testSessions = [
			{
				id: "sess-q1",
				file_path: "/path/to/q1",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-q2",
				file_path: "/path/to/q2",
				timestamp: "2026-05-20T10:01:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-q3",
				file_path: "/path/to/q3",
				timestamp: "2026-05-20T10:02:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-q4",
				file_path: "/path/to/q4",
				timestamp: "2026-05-20T10:03:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-q5",
				file_path: "/path/to/q5",
				timestamp: "2026-05-20T10:04:00Z",
				file_size: 100,
				parsed: false,
			},
		];

		await mockWailsAPI(page, testSessions);

		// GetSessionDetailをフックして、呼び出し順を記録しつつ遅延させる
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__detailCalls = [];
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const originalGetSessionDetail = (window as any).go.main.App
				.GetSessionDetail;
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).go.main.App.GetSessionDetail = async (id: string) => {
				// biome-ignore lint/suspicious/noExplicitAny: mock
				(window as any).__detailCalls.push(id);
				// 意図的に少し遅延させてキューが溜まるようにする
				await new Promise((resolve) => setTimeout(resolve, 200));
				return originalGetSessionDetail(id);
			};
		});

		// ページロード
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
		});

		await page.goto("/");

		// キューへの投入順を制御する
		await page.evaluate(() => {
			// biome-ignore lint/suspicious/noExplicitAny: test helper
			const parse = (window as any).parseSessions;
			// 1, 2, 3 をキューに投入 (これらは maxConcurrency=3 なので即座に実行状態へ)
			parse(["sess-q1", "sess-q2", "sess-q3"]);
			// 4 を優先度なしでキューに投入 (これはキューの末尾に追加される)
			parse(["sess-q4"], false);
			// 5 を優先度ありでキューに投入 (優先キューイングが効けば、4より前に割り込む)
			parse(["sess-q5"], true);
		});

		// 全てのセッションがパース完了するのを待つ（遅延が200msなので十分な時間を待つ）
		await page.waitForTimeout(1500);

		// biome-ignore lint/suspicious/noExplicitAny: mock
		const calls = await page.evaluate(() => (window as any).__detailCalls);

		// 期待される呼び出し順:
		// 最初の3つは sess-q1, sess-q2, sess-q3
		// その後、優先キューイングが機能していれば、sess-q5 が先に呼ばれ、最後に sess-q4 が呼ばれる。
		expect(calls.slice(0, 3)).toContain("sess-q1");
		expect(calls.slice(0, 3)).toContain("sess-q2");
		expect(calls.slice(0, 3)).toContain("sess-q3");
		expect(calls[3]).toBe("sess-q5");
		expect(calls[4]).toBe("sess-q4");
	});

	test("ディレクトリ分類タブにおいて、選択した日付に未解析セッションがあれば自動的にパースが開始されること", async ({
		page,
	}) => {
		// 1件の未解析セッションを用意
		const testSessions = [
			{
				id: "sess-012",
				file_path: "/path/to/auto-unparsed",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: false,
			},
		];

		await mockWailsAPI(page, testSessions);

		// 履歴ツリーで自動展開されるのを防ぐ、かつ選択日付を 2026-05-20 に設定する
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
			window.sessionStorage.setItem("session_list_selected_date", "2026-05-20");
		});

		// GetSessionDetail をモック
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__detailCalledFor = [];
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const originalGetSessionDetail = (window as any).go.main.App
				.GetSessionDetail;
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).go.main.App.GetSessionDetail = async (id: string) => {
				// biome-ignore lint/suspicious/noExplicitAny: mock
				(window as any).__detailCalledFor.push(id);

				if (id === "sess-012") {
					// biome-ignore lint/suspicious/noExplicitAny: mock
					const session = ((window as any).__dummySessions as any[])?.find(
						(s) => s.id === id,
					);
					if (session) {
						session.parsed = true;
						session.cwd = "/Users/test/projects/auto-unparsed-app";
						session.branch = "feature/auto-parsed";
					}

					return {
						id: id,
						cache_schema_version: 3,
						parsed_at: "2026-05-20T14:00:00Z",
						nodes: [
							{
								id: "node-meta",
								type: "sessionMeta",
								position: { x: 0, y: 0 },
								data: {
									category: "meta",
									label: "Session Meta",
									icon: "⚙️",
									turnIndex: -1,
									meta: {
										cwd: "/Users/test/projects/auto-unparsed-app",
										git_branch: "feature/auto-parsed",
									},
								},
							},
						],
						statistics: {
							duration_ms: 1000,
							total_tokens: 0,
							tool_call_count: 0,
							turn_count: 0,
							turns: [],
						},
					};
				}

				return originalGetSessionDetail(id);
			};
		});

		// ページロード
		await page.goto("/");

		// 履歴ツリータブでは自動解析が走らないことを確認
		const callsBefore = await page.evaluate(
			// biome-ignore lint/suspicious/noExplicitAny: mock
			() => (window as any).__detailCalledFor,
		);
		expect(callsBefore).not.toContain("sess-012");

		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		// 自動解析が走り、セッションがパース完了することを確認
		// 解析が走ると CWD やブランチ名が反映されるはず
		const sessionRow = page
			.locator("div[role='button']:has-text('sess-012')")
			.first();
		await expect(sessionRow.locator("text=feature/auto-parsed")).toBeVisible({
			timeout: 5000,
		});
		await expect(
			sessionRow.locator('text="/Users/test/projects/auto-unparsed-app"'),
		).toBeVisible();

		const callsAfter = await page.evaluate(
			// biome-ignore lint/suspicious/noExplicitAny: mock
			() => (window as any).__detailCalledFor,
		);
		expect(callsAfter).toContain("sess-012");
	});

	test("ディレクトリ分類タブにおいて、セッションが cwd ごとにグループ化されてアコーディオン表示されること", async ({
		page,
	}) => {
		// グループ化検証用のテストセッション
		const testSessions = [
			{
				id: "sess-g1",
				file_path: "/path/to/g1",
				cwd: "/Users/test/projects/react-app",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: true,
			},
			{
				id: "sess-g2",
				file_path: "/path/to/g2",
				cwd: "/Users/test/projects/go-app",
				timestamp: "2026-05-20T11:00:00Z",
				file_size: 100,
				parsed: true,
			},
			{
				id: "sess-g3",
				file_path: "/path/to/g3",
				cwd: undefined, // 未解析
				timestamp: "2026-05-20T12:00:00Z",
				file_size: 100,
				parsed: false,
			},
		];

		await mockWailsAPI(page, testSessions);

		// 履歴ツリーで自動展開されるのを防ぐ、かつ選択日付を 2026-05-20 に設定する
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
			window.sessionStorage.setItem("session_list_selected_date", "2026-05-20");
		});

		await page.goto("/");

		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		// グループヘッダーが表示されていることを確認
		await expect(
			page
				.locator("span[class*='directoryTitle']")
				.getByText("/Users/test/projects/react-app"),
		).toBeVisible();
		await expect(
			page
				.locator("span[class*='directoryTitle']")
				.getByText("/Users/test/projects/go-app"),
		).toBeVisible();
		await expect(
			page
				.locator("span[class*='directoryTitle']")
				.getByText("未解析のセッション"),
		).toBeVisible();
	});

	test("ディレクトリ分類タブにおいて、アコーディオンの開閉操作および sessionStorage への永続化が正しく機能すること", async ({
		page,
	}) => {
		const testSessions = [
			{
				id: "sess-g1",
				file_path: "/path/to/g1",
				cwd: "/Users/test/projects/react-app",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: true,
			},
			{
				id: "sess-g2",
				file_path: "/path/to/g2",
				cwd: "/Users/test/projects/go-app",
				timestamp: "2026-05-20T11:00:00Z",
				file_size: 100,
				parsed: true,
			},
		];

		await mockWailsAPI(page, testSessions);

		// 履歴ツリーの自動展開防止と、初期日付設定
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
			window.sessionStorage.setItem("session_list_selected_date", "2026-05-20");
		});

		await page.goto("/");

		// ディレクトリ分類タブへ切り替え
		await page.locator("button:has-text('ディレクトリ分類')").click();

		// 初期状態でアコーディオンがすべて展開されており、sess-g1とsess-g2が表示されていることを確認
		const rowG1 = page
			.locator("div[role='button']:has-text('sess-g1')")
			.first();
		const rowG2 = page
			.locator("div[role='button']:has-text('sess-g2')")
			.first();
		await expect(rowG1).toBeVisible();
		await expect(rowG2).toBeVisible();

		// 「/Users/test/projects/react-app」のヘッダーをクリックして閉じる
		const headerG1 = page.locator(
			"button:has-text('/Users/test/projects/react-app')",
		);
		await headerG1.click();

		// sess-g1 が非表示になり、sess-g2 が表示されたままであることを確認
		await expect(rowG1).not.toBeVisible();
		await expect(rowG2).toBeVisible();

		// sessionStorage を確認
		const collapsedDirs = await page.evaluate(() => {
			return window.sessionStorage.getItem("session_list_collapsed_dirs");
		});
		expect(collapsedDirs).toContain("/Users/test/projects/react-app");

		// 詳細画面へ遷移する
		await rowG2.click();
		await expect(page).toHaveURL(/.*#\/sessions\/sess-g2/);

		// 一覧画面へ戻る
		await page.locator("button:has-text('Back to List')").click();

		// ディレクトリ分類タブが復元されていることを確認
		await expect(
			page.locator("button:has-text('ディレクトリ分類')"),
		).toHaveClass(/activeTab/);

		// 状態が維持され、sess-g1は折りたたまれたまま非表示、sess-g2は表示されていることを確認
		const rowG1New = page
			.locator("div[role='button']:has-text('sess-g1')")
			.first();
		const rowG2New = page
			.locator("div[role='button']:has-text('sess-g2')")
			.first();
		await expect(rowG1New).not.toBeVisible();
		await expect(rowG2New).toBeVisible();

		// 再びヘッダーをクリックして展開する
		const headerG1New = page.locator(
			"button:has-text('/Users/test/projects/react-app')",
		);
		await headerG1New.click();

		// 再表示されることを確認
		await expect(rowG1New).toBeVisible();
	});

	test("ファイル追加検知イベントによってセッション一覧が自動リロードされ、新規の未解析セッションが自動パースされること", async ({
		page,
	}) => {
		// 1. 新しい未解析セッションデータを準備
		const newSession = {
			id: "sess-015-new-unparsed",
			file_path: "/path/to/session-15",
			cwd: undefined,
			cli_version: undefined,
			originator: "user-1",
			model_provider: undefined,
			branch: undefined,
			source: "cli",
			timestamp: "2026-05-20T08:00:00Z", // 2026年5月20日（既存の表示対象の日付）
			file_size: 256,
			file_modified_at: "2026-05-20T08:00:00Z",
			parsed: false,
		};

		// このセッション用の GetSessionDetail の詳細モックを設定
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const App = (window as any).go.main.App;
			const originalGetSessionDetail = App.GetSessionDetail;

			App.GetSessionDetail = async (id: string) => {
				if (id === "sess-015-new-unparsed") {
					// ダミーのリスト状態を更新（パース完了にする）
					// biome-ignore lint/suspicious/noExplicitAny: mock
					const session = ((window as any).__dummySessions as any[])?.find(
						(s) => s.id === id,
					);
					if (session) {
						session.parsed = true;
						session.cwd = "/Users/test/projects/new-detected-app";
						session.branch = "feature/new-detected";
						session.model_provider = "openai";
					}

					return {
						id: id,
						cache_schema_version: 3,
						parsed_at: "2026-05-20T08:10:00Z",
						nodes: [
							{
								id: "node-meta",
								type: "sessionMeta",
								position: { x: 0, y: 0 },
								data: {
									category: "meta",
									label: "Session Meta",
									icon: "⚙️",
									turnIndex: -1,
									meta: {
										cwd: "/Users/test/projects/new-detected-app",
										git_branch: "feature/new-detected",
										model_provider: "openai",
									},
								},
							},
						],
						statistics: {
							duration_ms: 100,
							total_tokens: 0,
							tool_call_count: 0,
							turn_count: 0,
							turns: [],
						},
					};
				}
				return originalGetSessionDetail(id);
			};
		});

		// ページを表示
		await page.goto("/");

		// 最初は sess-015-new-unparsed が存在しないことを確認
		await expect(page.locator("text=sess-015")).not.toBeVisible();

		// dummySessionsに新セッションを追加し、イベントを発生させる
		await page.evaluate((newSess) => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const dummy = (window as any).__dummySessions;
			dummy.push(newSess);
			// イベント発火
			// biome-ignore lint/suspicious/noExplicitAny: mock
			(window as any).__emitWailsEvent(
				"session-dir-changed",
				newSess.file_path,
			);
		}, newSession);

		// sess-015-new-unparsed が出現し、自動パースされてCWDが反映されることを検証
		const newRow = page
			.locator("div[role='button']:has-text('sess-015')")
			.first();
		await expect(newRow).toBeVisible();

		// 自動パースの結果、CWDとブランチ名が表示されることを待機
		await expect(newRow.locator("text=feature/new-detected")).toBeVisible({
			timeout: 5000,
		});
		await expect(
			newRow.locator('text="/Users/test/projects/new-detected-app"'),
		).toBeVisible();
	});

	test("年月内の未解析セッションの一括解析ボタンが動作し、順次パースされ、完了後にすべて解析済みになること", async ({
		page,
	}) => {
		// 2件 of 未解析セッションを用意 (sess-u1, sess-u2)
		const testSessions = [
			{
				id: "sess-u1",
				file_path: "/path/to/u1",
				timestamp: "2026-05-20T10:00:00Z",
				file_size: 100,
				parsed: false,
			},
			{
				id: "sess-u2",
				file_path: "/path/to/u2",
				timestamp: "2026-05-20T10:01:00Z",
				file_size: 100,
				parsed: false,
			},
		];

		await mockWailsAPI(page, testSessions);

		// 初期ロード時の自動展開やタブ選択による自動パースを防ぐために、状態を制御する
		await page.addInitScript(() => {
			window.sessionStorage.setItem("session_list_expanded_paths", "[]");
			window.sessionStorage.setItem("session_list_active_tab", "history");
		});

		// GetSessionDetail をモック。解析時にダミーデータを設定する
		await page.addInitScript(() => {
			// biome-ignore lint/suspicious/noExplicitAny: mock
			const App = (window as any).go.main.App;
			App.GetSessionDetail = async (id: string) => {
				// 「解析中...」の表示を検知できるようにするため少し遅延
				await new Promise((resolve) => setTimeout(resolve, 300));

				// dummySessions 内のステータスを更新
				// biome-ignore lint/suspicious/noExplicitAny: mock
				const session = ((window as any).__dummySessions as any[])?.find(
					(s) => s.id === id,
				);
				if (session) {
					session.parsed = true;
					session.cwd = `/Users/test/projects/${id}`;
					session.branch = `feature/${id}`;
				}

				return {
					id: id,
					cache_schema_version: 3,
					parsed_at: "2026-05-20T14:00:00Z",
					nodes: [
						{
							id: "node-meta",
							type: "sessionMeta",
							position: { x: 0, y: 0 },
							data: {
								category: "meta",
								label: "Session Meta",
								icon: "⚙️",
								turnIndex: -1,
								meta: {
									cwd: `/Users/test/projects/${id}`,
									git_branch: `feature/${id}`,
								},
							},
						},
					],
					statistics: {
						duration_ms: 100,
						total_tokens: 0,
						tool_call_count: 0,
						turn_count: 0,
						turns: [],
					},
				};
			};
		});

		await page.goto("/");

		// 一括解析ボタンが表示されていることを確認
		// 未解析セッションが2件あるので、ボタンは「未解析を一括解析 (2件)」と表示される
		const parseAllBtn = page.locator("#bulk-parse-btn");
		await expect(parseAllBtn).toBeVisible();
		await expect(parseAllBtn).toHaveText("🔄 未解析を一括解析 (2件)");

		// ボタンをクリックして一括パースを実行
		await parseAllBtn.click();

		// 解析中の表示に切り替わることを確認
		// 3並列で実行されるため、2件とも即座に解析中状態になるはず
		// ボタンは「⏳ 解析中...」になり、かつ disabled であることを確認
		await expect(parseAllBtn).toHaveText(/⏳ 解析中... \(2\/2件\)/);
		await expect(parseAllBtn).toBeDisabled();

		// パース完了（遅延300ms後）を待ち、ボタンが「すべて解析済み」になり、disabled になることを確認
		await expect(parseAllBtn).toHaveText("✓ すべて解析済み");
		await expect(parseAllBtn).toBeDisabled();

		// 画面上の各セッション行が更新されていることを確認
		// （20日ノードを展開して確認）
		const dayHeader = page.locator("div[role='button']:has-text('20日')");
		await dayHeader.click();

		await expect(page.getByText("sess-u1").first()).toBeVisible();
		await expect(
			page.getByText("/Users/test/projects/sess-u1").first(),
		).toBeVisible();
		await expect(page.getByText("sess-u2").first()).toBeVisible();
		await expect(
			page.getByText("/Users/test/projects/sess-u2").first(),
		).toBeVisible();
	});

	test("カレンダーのセッション数に応じた色のグラデーション（草）およびホバー時のツールチップ表示検証", async ({
		page,
	}) => {
		// ページロード (デフォルトのダミーセッション)
		await page.goto("/");

		// 「ディレクトリ分類」タブをクリック
		await page.locator("button:has-text('ディレクトリ分類')").click();

		const dateInput = page.locator("input[type='date']");
		await expect(dateInput).toBeVisible();

		// 日付を 2026-05-20 に設定 (5件のセッションがある日)
		await dateInput.fill("2026-05-20");

		// カレンダーのポップアップを表示するためトリガーボタンをクリック
		const triggerBtn = page.locator("button:has-text('📅')");
		await triggerBtn.click();

		// 19日セル（1件のセッション -> レベル1）と20日セル（5件のセッション -> レベル4）を特定
		const cell19 = page
			.locator("div[class*='daysGrid'] button")
			.filter({ hasText: /^19$/ })
			.first();
		const cell20 = page
			.locator("div[class*='daysGrid'] button")
			.filter({ hasText: /^20$/ })
			.first();
		const cell18 = page
			.locator("div[class*='daysGrid'] button")
			.filter({ hasText: /^18$/ })
			.first();

		// レベル判定のクラスが正しく適用されていることを確認
		await expect(cell19).toHaveClass(/graphL1/);
		await expect(cell20).toHaveClass(/graphL4/);
		await expect(cell18).not.toHaveClass(/graphL/);

		// 19日セルへホバーした際、ツールチップが「1 session on 2026-05-19」として表示されることを確認
		await cell19.hover();
		const tooltip = page.locator("div[class*='tooltip']");
		await expect(tooltip).toBeVisible();
		await expect(tooltip).toHaveText("1 session on 2026-05-19");

		// 20日セルへホバーした際、ツールチップが「5 sessions on 2026-05-20」として表示されることを確認
		await cell20.hover();
		await expect(tooltip).toHaveText("5 sessions on 2026-05-20");

		// 18日セルへホバーした際、ツールチップが「0 sessions on 2026-05-18」として表示されることを確認
		await cell18.hover();
		await expect(tooltip).toHaveText("0 sessions on 2026-05-18");
	});
});
