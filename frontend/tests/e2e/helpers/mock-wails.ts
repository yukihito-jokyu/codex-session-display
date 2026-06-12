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
	{
		id: "sess-005-timestamp-error",
		file_path: "/path/to/session-5",
		cwd: "/Users/test/projects/error-app",
		cli_version: "1.0.0",
		originator: "user-1",
		model_provider: "anthropic",
		branch: "main",
		source: "cli",
		timestamp: "1999-05-20T10:00:00Z", // 1999年
		file_size: 1024,
		file_modified_at: "1999-05-20T10:00:00Z",
		parsed: true,
	},
	{
		id: "sess-006-different-day",
		file_path: "/path/to/session-6",
		cwd: "/Users/test/projects/react-app",
		cli_version: "1.0.0",
		originator: "user-1",
		model_provider: "anthropic",
		branch: "main",
		source: "cli",
		timestamp: "2026-05-19T10:00:00Z", // 19日
		file_size: 1024,
		file_modified_at: "2026-05-19T10:00:00Z",
		parsed: true,
	},
	{
		id: "sess-007-null-fields",
		file_path: "/path/to/session-7",
		cwd: undefined,
		cli_version: undefined,
		originator: "user-1",
		model_provider: undefined,
		branch: undefined,
		source: "cli",
		timestamp: "2026-05-20T10:00:00Z", // 20日
		file_size: 1024,
		file_modified_at: "2026-05-20T10:00:00Z",
		parsed: true, // 解析済だがフィールドが空
	},
	{
		id: "sess-010-zero-size",
		file_path: "/path/to/session-10",
		cwd: "/Users/test/projects/zero-size-app",
		cli_version: "1.0.0",
		originator: "user-1",
		model_provider: "openai",
		branch: "zero-size",
		source: "cli",
		timestamp: "2026-05-20T01:00:00Z",
		file_size: 0,
		file_modified_at: "2026-05-20T01:00:00Z",
		parsed: true,
	},
];

export async function mockWailsAPI(
	page: Page,
	sessions: dto.SessionSummary[] = dummySessions,
) {
	// Wails APIのモックをブラウザコンテキストに注入
	await page.addInitScript((sessionsArg) => {
		const dummySessions = sessionsArg as dto.SessionSummary[];

		const clipboard = {
			writeText: async (text: string) => {
				(window as any).__copiedTexts = (window as any).__copiedTexts || [];
				(window as any).__copiedTexts.push(text);
			},
		};
		Object.defineProperty(navigator, "clipboard", {
			value: clipboard,
			configurable: true,
		});

		// グローバルオブジェクト go.main.App.ListSessions を定義
		const go = (window as any).go || {};
		const runtime = (window as any).runtime || {};
		const eventListeners = new Map<string, Set<(...args: unknown[]) => void>>();

		go.main = go.main || {};
		go.main.App = go.main.App || {};
		runtime.EventsOnMultiple = (
			eventName: string,
			callback: (...args: unknown[]) => void,
			_maxCallbacks: number,
		) => {
			const listeners = eventListeners.get(eventName) || new Set();
			listeners.add(callback);
			eventListeners.set(eventName, listeners);

			return () => {
				listeners.delete(callback);
				if (listeners.size === 0) {
					eventListeners.delete(eventName);
				}
			};
		};

		(window as any).__emitWailsEvent = (
			eventName: string,
			...args: unknown[]
		) => {
			const listeners = eventListeners.get(eventName);
			if (!listeners) {
				return;
			}
			for (const listener of listeners) {
				listener(...args);
			}
		};

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

			if (query === "timestamp-error") {
				return dummySessions.filter((s) => s.id === "sess-005-timestamp-error");
			}

			if (query === "hang") {
				return new Promise(() => {}); // never resolves
			}

			if (query === "return-undefined") {
				return undefined as any;
			}

			if (query === "trigger-string-error") {
				throw "Mocked List String Error";
			}

			if (year === 0 && month === 0 && (window as any).__triggerInitialError) {
				throw new Error("Initial Load Error");
			}

			if (year === 0 && month === 0 && (window as any).__hangInitialLoad) {
				return new Promise(() => {}); // never resolves
			}

			if (query === "multi-date") {
				return dummySessions;
			}

			if (query === "test-timestamp-fallbacks") {
				const sessA = {
					id: "sess-008-fallback-undefined",
					file_path: "/path/to/session-8",
					cwd: "/Users/test/projects/react-app",
					cli_version: "1.0.0",
					originator: "user-1",
					model_provider: "anthropic",
					branch: "main",
					source: "cli",
					timestamp: undefined,
					file_size: 1024,
					file_modified_at: "2026-05-20T10:00:00Z",
					parsed: true,
				};
				const sessB = {
					id: "sess-009-fallback-invalid",
					file_path: "/path/to/session-9",
					cwd: "/Users/test/projects/react-app",
					cli_version: "1.0.0",
					originator: "user-1",
					model_provider: "anthropic",
					branch: "main",
					source: "cli",
					timestamp: "invalid-date-format-string",
					file_size: 1024,
					file_modified_at: "2026-05-20T10:00:00Z",
					parsed: true,
				};
				return [sessA, sessB];
			}

			if (query === "all-invalid-grouping-dates") {
				return [
					{
						id: "sess-011-invalid-grouping",
						file_path: "/path/to/session-11",
						cwd: "/Users/test/projects/invalid-grouping",
						cli_version: "1.0.0",
						originator: "user-1",
						model_provider: "openai",
						branch: "broken-date",
						source: "cli",
						timestamp: "invalid-date-format-string",
						file_size: 128,
						file_modified_at: "also-invalid",
						parsed: true,
					},
				];
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

		go.main.App.ResolveSessionIDFromPath = async (filePath: string) => {
			(window as any).__resolveSessionIDCalls =
				(window as any).__resolveSessionIDCalls || [];
			(window as any).__resolveSessionIDCalls.push({ filePath });

			const session = dummySessions.find((item) => item.file_path === filePath);
			if (!session) {
				throw new Error("Session path not found");
			}

			return session.id;
		};

		go.main.App.FrontendReady = async () => {
			(window as any).__frontendReadyCalls =
				((window as any).__frontendReadyCalls || 0) + 1;
		};

		go.main.App.GetSessionDetail = async (id: string) => {
			(window as any).__getSessionDetailCalls =
				(window as any).__getSessionDetailCalls || [];
			(window as any).__getSessionDetailCalls.push({ id });

			if (id === "trigger-error") {
				throw new Error("Mocked Detail API Error");
			}

			if (id === "trigger-string-error") {
				throw "Mocked Detail String Error";
			}

			if (id === "sess-002-uuid-long-name") {
				return {
					id: id,
					cache_schema_version: 2,
					parsed_at: "2026-05-20T12:00:00Z",
					nodes: undefined,
					edges: undefined,
					statistics: {
						duration_ms: 1000,
						total_tokens: 0,
						tool_call_count: 0,
						token_count_count: 0,
						context_window_size: 0,
						turn_count: 0,
						turns: [],
					},
					token_counts: [
						{
							index: 0,
							turn_index: 0,
							bound_to_node_id: "node-user-msg",
							last_token_usage: undefined,
							total_token_usage: undefined,
						},
					],
				};
			}

			if (id === "sess-003-uuid-long-name") {
				return {
					id: id,
					cache_schema_version: 2,
					parsed_at: "2026-04-10T09:00:00Z",
					nodes: [
						{
							id: "node-summary-only",
							type: "agentMessage",
							position: { x: 0, y: 0 },
							data: {
								category: "message",
								label: "Summary Only",
								icon: "🤖",
								summary: "Token log omitted session",
								turnIndex: 0,
							},
						},
					],
					edges: [],
					statistics: {
						duration_ms: 500,
						total_tokens: 0,
						tool_call_count: 0,
						token_count_count: 0,
						context_window_size: 0,
						turn_count: 1,
						turns: [
							{
								index: 0,
								duration_ms: 500,
								time_to_first_token_ms: 200,
								token_count_count: 0,
								consumed_tokens: {
									total_tokens: 0,
									input_tokens: 0,
									output_tokens: 0,
									reasoning_output_tokens: 0,
								},
							},
						],
					},
					token_counts: undefined,
				};
			}

			if (id === "sess-no-turns") {
				return {
					id: id,
					cache_schema_version: 2,
					parsed_at: "2026-05-20T09:00:00Z",
					nodes: [
						{
							id: "node-no-turns",
							type: "sessionMeta",
							position: { x: 0, y: 0 },
							data: {
								category: "meta",
								label: "No Turns",
								icon: "⚙️",
								summary: "No turn data",
								turnIndex: -1,
							},
						},
					],
					edges: [],
					statistics: {
						duration_ms: 0,
						total_tokens: 0,
						tool_call_count: 0,
						token_count_count: 0,
						context_window_size: 0,
						turn_count: 0,
						turns: undefined,
					},
					token_counts: undefined,
				};
			}

			if (id === "performance-visibility") {
				return {
					id: id,
					cache_schema_version: 2,
					parsed_at: "2026-05-20T10:00:00Z",
					nodes: [
						{
							id: "node-near",
							type: "userMessage",
							position: { x: 0, y: 0 },
							data: {
								category: "message",
								label: "Near Node",
								icon: "👤",
								summary: "Zoom target",
								turnIndex: 0,
							},
						},
						{
							id: "node-far-away",
							type: "generic",
							position: { x: 100000, y: 100000 },
							data: {
								category: "event",
								label: "Far Away Node",
								icon: "📄",
								summary: "This node is outside the target viewport",
								turnIndex: 0,
							},
						},
					],
					edges: [],
					statistics: {
						duration_ms: 1000,
						total_tokens: 1000,
						tool_call_count: 0,
						token_count_count: 1,
						context_window_size: 100000,
						turn_count: 1,
						turns: [],
					},
					token_counts: [
						{
							index: 0,
							turn_index: 0,
							bound_to_node_id: "node-near",
							last_token_usage: {
								total_tokens: 1000,
								input_tokens: 800,
								output_tokens: 200,
								reasoning_output_tokens: 0,
								cached_input_tokens: 0,
							},
							total_token_usage: {
								total_tokens: 1000,
								input_tokens: 800,
								output_tokens: 200,
								reasoning_output_tokens: 0,
								cached_input_tokens: 0,
							},
						},
					],
				};
			}

			return {
				id: id,
				cache_schema_version: 2,
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
							tokenBadge: {
								consumedTokens: 2500000, // 2.5M
								tokenCountIndex: 0,
								boundCount: 1,
							},
							meta: {
								version: "1.0.0",
								config: { debug: true },
							},
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
						id: "node-context-doc-long",
						type: "contextDoc",
						position: { x: 650, y: 0 },
						data: {
							category: "context",
							label: "Long Context",
							icon: "",
							summary: "▸ クリックして展開",
							fullText: "L".repeat(1201),
							turnIndex: 0,
						},
					},
					{
						id: "node-context-doc-warning",
						type: "contextDoc",
						position: { x: 900, y: 0 },
						data: {
							category: "context",
							label: "Warning Context",
							icon: "⚠️",
							summary: "▸ クリックして展開",
							fullText: "Warning context details",
							textLength: 42,
							turnIndex: -1,
						},
					},
					{
						id: "node-context-doc-empty",
						type: "contextDoc",
						position: { x: 400, y: 120 },
						data: {
							category: "context",
							label: "Empty Context",
							icon: "",
							summary: "▸ クリックして展開",
							fullText: undefined,
							turnIndex: 0,
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
								consumedTokens: 50000, // 50K
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
							fullText: undefined, // fullTextを空にして「No additional details...」を検証
							turnIndex: -1,
							tokenBadge: {
								consumedTokens: 1500, // 1.5K
								tokenCountIndex: 0,
								boundCount: 1,
							},
						},
					},
					{
						id: "node-agent-msg",
						type: "agentMessage",
						position: { x: 200, y: 240 },
						data: {
							category: "message",
							label: "Agent Message",
							icon: "🤖",
							summary: "Helpful agent response",
							fullText: "Here is how to solve...",
							turnIndex: 1,
							tokenBadge: {
								consumedTokens: 500, // 500
								tokenCountIndex: 1,
								boundCount: 1,
							},
						},
					},
					{
						id: "node-turn-ctx",
						type: "turnContext",
						position: { x: 0, y: 360 },
						data: {
							category: "context",
							label: "Turn Context",
							icon: "🔄",
							summary: "Context summary",
							turnIndex: 0,
						},
					},
					{
						id: "node-dev-msg",
						type: "developerMessage",
						position: { x: 200, y: 360 },
						data: {
							category: "message",
							label: "Developer Message",
							icon: "",
							summary: "Developer instructions",
							turnIndex: 0,
						},
					},
					{
						id: "node-user-api-msg",
						type: "userApiMessage",
						position: { x: 400, y: 360 },
						data: {
							category: "message",
							label: "User API Message",
							icon: "🔌",
							summary: "API Call message",
							turnIndex: 0,
							tokenBadge: {
								consumedTokens: 0,
								tokenCountIndex: 2,
								boundCount: 1,
							},
						},
					},
					{
						id: "node-reasoning",
						type: "reasoning",
						position: { x: 600, y: 360 },
						data: {
							category: "thought",
							label: "Reasoning Step",
							icon: "🧠",
							summary: "Thinking process details",
							turnIndex: 0,
						},
					},
					{
						id: "node-action",
						type: "action",
						position: { x: 800, y: 360 },
						data: {
							category: "action",
							label: "Call Tool",
							icon: "🛠️",
							summary: "Tool invocation details",
							turnIndex: 0,
						},
					},
					{
						id: "node-websearch",
						type: "webSearchAction",
						position: { x: 1000, y: 360 },
						data: {
							category: "action",
							label: "Web Search",
							icon: "🔍",
							summary: "Searching the web details",
							turnIndex: 0,
						},
					},
					{
						id: "node-external-event",
						type: "externalEvent",
						position: { x: 0, y: 480 },
						data: {
							category: "event",
							label: "External Input",
							icon: "📡",
							summary: "External system trigger",
							turnIndex: 0,
						},
					},
					{
						id: "node-task-event-started",
						type: "taskEvent",
						position: { x: 200, y: 480 },
						data: {
							category: "event",
							label: "Task Started",
							icon: "▶️",
							summary: "Workflow process start",
							turnIndex: 0,
						},
					},
					{
						id: "node-task-event-aborted",
						type: "taskEvent",
						position: { x: 400, y: 480 },
						data: {
							category: "event",
							label: "Task Aborted",
							icon: "⏹️",
							summary: "Workflow process aborted",
							turnIndex: 0,
						},
					},
					{
						id: "node-item-completed",
						type: "itemCompleted",
						position: { x: 600, y: 480 },
						data: {
							category: "event",
							label: "Item Completed",
							icon: "✅",
							summary: "Item completed details",
							turnIndex: 0,
						},
					},
					{
						id: "node-generic",
						type: "generic", // typeをgenericにしてnodeTypesにマッピング
						position: { x: 800, y: 480 },
						data: {
							category: "generic",
							label: "", // labelを空にしてidへのフォールバックを検証
							icon: "❓",
							summary: "Fallback node details",
							turnIndex: 0,
							meta: {},
						},
					},
					{
						id: "node-task-event-general",
						type: "taskEvent",
						position: { x: 1000, y: 480 },
						data: {
							category: "event",
							label: "Task Event", // Started/Complete/Aborted を含まないラベル
							icon: "🔔",
							summary: "General workflow event",
							turnIndex: 0,
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
					duration_ms: 75000, // 1分15秒 (1m 15s)
					total_tokens: 150000,
					tool_call_count: 5,
					token_count_count: 10,
					context_window_size: 128000,
					turn_count: 3,
					turns: [
						{
							index: 0,
							collaboration_mode_kind: "normal",
							duration_ms: 5000,
							time_to_first_token_ms: 500,
							token_count_count: 4,
							consumed_tokens: {
								total_tokens: 50000,
								input_tokens: 40000,
								output_tokens: 10000,
								reasoning_output_tokens: 2000,
							},
						},
						{
							index: 1,
							collaboration_mode_kind: "collaboration",
							duration_ms: 10500,
							time_to_first_token_ms: 800,
							token_count_count: 6,
							consumed_tokens: {
								total_tokens: 100000,
								input_tokens: 80000,
								output_tokens: 20000,
								reasoning_output_tokens: 8000,
							},
						},
						{
							index: 2,
							collaboration_mode_kind: undefined,
							duration_ms: 500,
							time_to_first_token_ms: 200,
							token_count_count: 0,
							consumed_tokens: undefined,
						},
					],
				},
				token_counts: [
					{
						index: 0,
						turn_index: 0,
						bound_to_node_id: "node-user-msg",
						last_token_usage: {
							total_tokens: 20000,
							input_tokens: 15000,
							output_tokens: 5000,
							reasoning_output_tokens: 1000,
							cached_input_tokens: 5000,
						},
						total_token_usage: {
							total_tokens: 20000,
							input_tokens: 15000,
							output_tokens: 5000,
							reasoning_output_tokens: 1000,
							cached_input_tokens: 5000,
						},
					},
					{
						index: 1,
						turn_index: 0,
						bound_to_node_id: "node-user-msg",
						last_token_usage: {
							total_tokens: 30000,
							input_tokens: 25000,
							output_tokens: 5000,
							reasoning_output_tokens: 1000,
							cached_input_tokens: 5000,
						},
						total_token_usage: {
							total_tokens: 50000,
							input_tokens: 40000,
							output_tokens: 10000,
							reasoning_output_tokens: 2000,
							cached_input_tokens: 10000,
						},
					},
					{
						index: 2,
						turn_index: 1,
						bound_to_node_id: "node-orphan-event",
						last_token_usage: {
							total_tokens: 100000,
							input_tokens: 80000,
							output_tokens: 20000,
							reasoning_output_tokens: 8000,
							cached_input_tokens: 20000,
						},
						total_token_usage: {
							total_tokens: 150000,
							input_tokens: 120000,
							output_tokens: 30000,
							reasoning_output_tokens: 10000,
							cached_input_tokens: 30000,
						},
					},
					{
						index: 3,
						turn_index: 1,
						bound_to_node_id: "node-user-api-msg",
						last_token_usage: undefined,
						total_token_usage: undefined,
					},
				],
			};
		};

		go.main.App.OpenLogDirectory = async () => {
			(window as any).__openLogDirectoryCalls =
				(window as any).__openLogDirectoryCalls || 0;
			(window as any).__openLogDirectoryCalls += 1;
		};

		go.main.App.GetLogFilePath = async () => {
			(window as any).__getLogFilePathCalls =
				(window as any).__getLogFilePathCalls || 0;
			(window as any).__getLogFilePathCalls += 1;
			return "/Users/test/.codex-display/logs/app.log";
		};

		(window as any).go = go;
		(window as any).runtime = runtime;
	}, sessions);
}
