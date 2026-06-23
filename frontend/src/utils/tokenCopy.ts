export interface TokenDetail {
	total: number;
	input: number;
	output: number;
}

export interface SubagentTokenInfo {
	nickname: string;
	total: number;
	input: number;
	output: number;
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

	const lines = [
		`セッション [${sessionId}] トータルトークン消費（サブエージェント含む）: ${format(totalTokens)} tokens`,
		`- 本体: ${format(mainTokens.total)} tokens (入力: ${format(mainTokens.input)} / 出力: ${format(mainTokens.output)})`,
	];

	for (const sub of subagents) {
		lines.push(
			`- サブエージェント (${sub.nickname}): ${format(sub.total)} tokens (入力: ${format(sub.input)} / 出力: ${format(sub.output)})`,
		);
	}

	return lines.join("\n");
}
