/* biome-ignore-all lint/suspicious/noExplicitAny: Wails mock requires dynamically extending window object */
import type { Page } from "@playwright/test";
import type { dto } from "wailsjs/go/models";

export const dummySessions: dto.SessionSummary[] = [
	{
		id: "sess-001-uuid-long-name",
		file_path: "/path/to/session-1",
		cwd: "/Users/test/projects/react-app",
		cli_version: "1.0.0",
		originator: "user-1",
		model_provider: "anthropic",
		branch: "main",
		source: "cli",
		timestamp: "2026-05-20T10:00:00Z", // 2026年5月20日 (session-2と同じ日にして同時表示)
		file_size: 1024,
		file_modified_at: "2026-05-20T10:00:00Z",
		parsed: true,
	},
	{
		id: "sess-002-uuid-long-name",
		file_path: "/path/to/session-2",
		cwd: "/Users/test/projects/go-app",
		cli_version: "1.1.0",
		originator: "user-2",
		model_provider: "openai",
		branch: "feature/auth",
		source: "web",
		timestamp: "2026-05-20T12:00:00Z", // 2026年5月20日
		file_size: 2048,
		file_modified_at: "2026-05-20T12:00:00Z",
		parsed: true,
	},
	{
		id: "sess-003-uuid-long-name",
		file_path: "/path/to/session-3",
		cwd: "/Users/test/projects/python-app",
		cli_version: "2.0.0",
		originator: "user-1",
		model_provider: "google",
		branch: "main",
		source: "cli",
		timestamp: "2026-04-10T09:00:00Z", // 2026年4月10日
		file_size: 512,
		file_modified_at: "2026-04-10T09:00:00Z",
		parsed: true,
	},
	{
		id: "sess-004-unparsed-session",
		file_path: "/path/to/session-4",
		cwd: undefined,
		cli_version: undefined,
		originator: "user-1",
		model_provider: undefined,
		branch: undefined,
		source: "cli",
		timestamp: "2026-05-20T14:00:00Z", // 2026年5月20日
		file_size: 512,
		file_modified_at: "2026-05-20T14:00:00Z",
		parsed: false,
	},
];

export async function mockWailsAPI(
	page: Page,
	sessions: dto.SessionSummary[] = dummySessions,
) {
	// Wails APIのモックをブラウザコンテキストに注入
	await page.addInitScript((sessionsArg) => {
		const dummySessions = sessionsArg as dto.SessionSummary[];

		// グローバルオブジェクト go.main.App.ListSessions を定義
		const go = (window as any).go || {};
		go.main = go.main || {};
		go.main.App = go.main.App || {};
		go.main.App.ListSessions = async (
			query: string,
			year: number,
			month: number,
		) => {
			// デバッグ用およびテストアサーション用に呼び出し引数を記録
			(window as any).__listSessionsCalls =
				(window as any).__listSessionsCalls || [];
			(window as any).__listSessionsCalls.push({ query, year, month });

			// クエリが trigger-error の場合は意図的に例外をスロー
			if (query === "trigger-error") {
				throw new Error("Mocked API Error");
			}

			let targetYear = year;
			let targetMonth = month;

			// 初回呼び出し (year=0, month=0) の場合は最新のセッションがある2026年5月をデフォルトとする
			if (year === 0 && month === 0) {
				targetYear = 2026;
				targetMonth = 5;
			}

			// 年月でフィルタリング
			let filtered = dummySessions.filter((s) => {
				if (!s.timestamp) return false;
				const d = new Date(s.timestamp);
				return (
					d.getFullYear() === targetYear && d.getMonth() + 1 === targetMonth
				);
			});

			// 検索クエリでフィルタリング
			if (query) {
				const q = query.toLowerCase();
				filtered = filtered.filter(
					(s) =>
						s.id.toLowerCase().includes(q) ||
						s.branch?.toLowerCase().includes(q) ||
						s.cwd?.toLowerCase().includes(q) ||
						s.model_provider?.toLowerCase().includes(q),
				);
			}

			return filtered;
		};

		go.main.App.GetSessionDetail = async (id: string) => {
			(window as any).__getSessionDetailCalls =
				(window as any).__getSessionDetailCalls || [];
			(window as any).__getSessionDetailCalls.push({ id });

			if (id === "trigger-error") {
				throw new Error("Mocked Detail API Error");
			}

			return {
				id: id,
				parsed_at: "2026-05-20T10:00:00Z",
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
							fullText: "Full session meta details here",
							turnIndex: -1,
						},
					},
					{
						id: "node-context-doc",
						type: "contextDoc",
						position: { x: 400, y: 0 },
						data: {
							category: "context",
							label: "User Instructions",
							icon: "📜",
							summary: "▸ クリックして展開",
							fullText:
								"This is a detailed user instruction text that can be expanded.",
							turnIndex: -1,
						},
					},
					{
						id: "node-user-msg",
						type: "userMessage",
						position: { x: 0, y: 120 },
						data: {
							category: "message",
							label: "User Message",
							icon: "👤",
							summary: "Hello, agent",
							fullText: "Hello, agent, please help me.",
							turnIndex: 0,
							tokenBadge: {
								totalTokens: 150000,
								tokenCountIndex: 0,
								boundCount: 2,
							},
						},
					},
					{
						id: "node-orphan-event",
						type: "taskEvent",
						position: { x: 0, y: 240 },
						data: {
							category: "event",
							label: "Orphan Complete",
							icon: "⚠️",
							summary: "Orphan complete/aborted event without task_started",
							fullText: "Orphan complete details",
							turnIndex: -1,
						},
					},
				],
				edges: [
					{
						id: "edge-meta-usermsg",
						source: "node-meta",
						target: "node-user-msg",
						type: "default",
						animated: false,
					},
					{
						id: "edge-context-meta",
						source: "node-context-doc",
						target: "node-meta",
						type: "step",
						animated: false,
					},
				],
				statistics: {
					duration_ms: 5000,
					total_tokens: 150,
					tool_call_count: 0,
					token_count_count: 0,
					context_window_size: 2048,
					turn_count: 1,
					turns: [],
				},
				token_counts: [],
			};
		};

		(window as any).go = go;
	}, sessions);
}
