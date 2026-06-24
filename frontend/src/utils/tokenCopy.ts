export interface TokenDetail {
	total: number;
	input: number;
	output: number;
	turnCount?: number | null;
	stepCount?: number | null;
	durationMs?: number | null;
}

export interface SubagentTokenInfo {
	nickname: string;
	total: number;
	input: number;
	output: number;
	turnCount?: number | null;
	stepCount?: number | null;
	durationMs?: number | null;
}

function formatDuration(ms: number | null | undefined): string {
	if (!ms || ms <= 0) return "0s";
	const totalSec = Math.floor(ms / 1000);
	const hours = Math.floor(totalSec / 3600);
	const minutes = Math.floor((totalSec % 3600) / 60);
	const seconds = totalSec % 60;

	if (hours > 0) {
		return `${hours}h ${minutes}m ${seconds}s`;
	}
	if (minutes > 0) {
		return `${minutes}m ${seconds}s`;
	}
	return `${seconds}s`;
}

/**
 * トータルトークンおよびサブエージェントの内訳整形テキストを生成します。
 */
export function generateTokenBreakdownText(
	sessionId: string,
	mainTokens: TokenDetail,
	subagents: SubagentTokenInfo[],
): string {
	const format = (n: number) => n.toLocaleString();
	const totalTokens =
		mainTokens.total + subagents.reduce((sum, s) => sum + s.total, 0);
	const totalInput =
		mainTokens.input + subagents.reduce((sum, s) => sum + s.input, 0);
	const totalOutput =
		mainTokens.output + subagents.reduce((sum, s) => sum + s.output, 0);

	const mainTurns = mainTokens.turnCount ?? 0;
	const mainSteps = mainTokens.stepCount ?? 0;
	const mainDuration = formatDuration(mainTokens.durationMs);

	const lines = [
		`セッション [${sessionId}] トータルトークン消費（サブエージェント含む）: ${format(totalTokens)} tokens (入力: ${format(totalInput)} / 出力: ${format(totalOutput)})`,
		`- 本体: ${format(mainTokens.total)} tokens (入力: ${format(mainTokens.input)} / 出力: ${format(mainTokens.output)}) | ${mainTurns} ターン, ${mainSteps} ステップ, ${mainDuration}`,
	];

	for (const sub of subagents) {
		const subTurns = sub.turnCount ?? 0;
		const subSteps = sub.stepCount ?? 0;
		const subDuration = formatDuration(sub.durationMs);
		lines.push(
			`- サブエージェント (${sub.nickname}): ${format(sub.total)} tokens (入力: ${format(sub.input)} / 出力: ${format(sub.output)}) | ${subTurns} ターン, ${subSteps} ステップ, ${subDuration}`,
		);
	}

	return lines.join("\n");
}
