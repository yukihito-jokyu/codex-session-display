import { expect, test } from "@playwright/test";
import {
	generateTokenBreakdownText,
	type SubagentTokenInfo,
} from "../../src/utils/tokenCopy";

test.describe("generateTokenBreakdownText", () => {
	test("should format token breakdown text with turns, steps, and duration (under 1 hour)", () => {
		const sessionId = "test-session-123";
		const mainTokens = {
			total: 1000,
			input: 800,
			output: 200,
			turnCount: 5,
			stepCount: 12,
			durationMs: 930000, // 15m 30s
		};
		const subagents = [
			{
				nickname: "Subagent1",
				total: 500,
				input: 400,
				output: 100,
				turnCount: 3,
				stepCount: 6,
				durationMs: 310000, // 5m 10s
			},
		];

		const result = generateTokenBreakdownText(sessionId, mainTokens, subagents);

		expect(result).toBe(
			`セッション [test-session-123] トータルトークン消費（サブエージェント含む）: 1,500 tokens (入力: 1,200 / 出力: 300)\n` +
				`- 本体: 1,000 tokens (入力: 800 / 出力: 200) | 5 ターン, 12 ステップ, 15m 30s\n` +
				`- サブエージェント (Subagent1): 500 tokens (入力: 400 / 出力: 100) | 3 ターン, 6 ステップ, 5m 10s`,
		);
	});

	test("should format duration with hours if over 1 hour", () => {
		const sessionId = "test-session-123";
		const mainTokens = {
			total: 1000,
			input: 800,
			output: 200,
			turnCount: 5,
			stepCount: 12,
			durationMs: 3910000, // 1h 5m 10s
		};
		const subagents: SubagentTokenInfo[] = [];

		const result = generateTokenBreakdownText(sessionId, mainTokens, subagents);

		expect(result).toBe(
			`セッション [test-session-123] トータルトークン消費（サブエージェント含む）: 1,000 tokens (入力: 800 / 出力: 200)\n` +
				`- 本体: 1,000 tokens (入力: 800 / 出力: 200) | 5 ターン, 12 ステップ, 1h 5m 10s`,
		);
	});

	test("should format duration with seconds only if under 1 minute", () => {
		const sessionId = "test-session-123";
		const mainTokens = {
			total: 1000,
			input: 800,
			output: 200,
			turnCount: 5,
			stepCount: 12,
			durationMs: 45000, // 45s
		};
		const subagents: SubagentTokenInfo[] = [];

		const result = generateTokenBreakdownText(sessionId, mainTokens, subagents);

		expect(result).toBe(
			`セッション [test-session-123] トータルトークン消費（サブエージェント含む）: 1,000 tokens (入力: 800 / 出力: 200)\n` +
				`- 本体: 1,000 tokens (入力: 800 / 出力: 200) | 5 ターン, 12 ステップ, 45s`,
		);
	});

	test("should fallback to 0/0s if optional fields are missing", () => {
		const sessionId = "test-session-123";
		const mainTokens = {
			total: 1000,
			input: 800,
			output: 200,
			turnCount: null,
			stepCount: null,
			durationMs: null,
		};
		const subagents = [
			{
				nickname: "Subagent1",
				total: 500,
				input: 400,
				output: 100,
				// missing optional fields
			},
		];

		const result = generateTokenBreakdownText(sessionId, mainTokens, subagents);

		expect(result).toBe(
			`セッション [test-session-123] トータルトークン消費（サブエージェント含む）: 1,500 tokens (入力: 1,200 / 出力: 300)\n` +
				`- 本体: 1,000 tokens (入力: 800 / 出力: 200) | 0 ターン, 0 ステップ, 0s\n` +
				`- サブエージェント (Subagent1): 500 tokens (入力: 400 / 出力: 100) | 0 ターン, 0 ステップ, 0s`,
		);
	});
});
