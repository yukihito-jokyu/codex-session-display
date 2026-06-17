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
		await expect(page.locator("text=未解析")).toBeVisible();

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
					await new Promise((resolve) => setTimeout(resolve, 200));

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
});
