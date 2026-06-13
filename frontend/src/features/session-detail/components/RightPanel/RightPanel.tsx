import React, { useMemo, useState } from "react";
import {
	Bar,
	BarChart,
	CartesianGrid,
	Legend,
	Line,
	LineChart,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import type { dto } from "wailsjs/go/models";
import styles from "./RightPanel.module.css";

interface RightPanelProps {
	statistics: dto.Statistics;
	tokenCounts: dto.TokenCountEntry[];
	nodes: dto.FlowNode[];
	selectedTokenCountIndices: number[];
	onTokenLogClick?: (tokenIndex: number) => void;
}

interface LastTokenChartData {
	name: string;
	index: number;
	nodeId?: string;
	total: number;
	input: number;
	output: number;
	reasoning: number;
	cached: number;
}

interface TokenChartDotProps {
	cx?: number;
	cy?: number;
	stroke?: string;
	payload?: LastTokenChartData;
	series: "input" | "output" | "reasoning" | "cached" | "total";
	radius: number;
	selected: boolean;
	onSelect?: (tokenIndex: number) => void;
}

function TokenChartDot({
	cx,
	cy,
	stroke,
	payload,
	series,
	radius,
	selected,
	onSelect,
}: TokenChartDotProps) {
	if (cx === undefined || cy === undefined || !payload) {
		return null;
	}

	const interactive = Boolean(payload.nodeId && onSelect);
	const selectTokenCount = () => {
		if (onSelect) {
			onSelect(payload.index);
		}
	};

	if (!interactive) {
		return (
			<circle
				cx={cx}
				cy={cy}
				r={radius}
				fill={stroke}
				stroke={stroke}
				data-selected={selected}
				data-testid={
					series === "total"
						? `last-token-point-${payload.index}`
						: `last-token-point-${series}-${payload.index}`
				}
			/>
		);
	}

	return (
		<circle
			cx={cx}
			cy={cy}
			r={selected ? radius + 2 : radius}
			fill={stroke}
			stroke={stroke}
			className={`${styles.interactiveChartPoint} ${
				selected ? styles.selectedChartPoint : ""
			}`}
			data-selected={selected}
			data-testid={
				series === "total"
					? `last-token-point-${payload.index}`
					: `last-token-point-${series}-${payload.index}`
			}
			role="button"
			aria-pressed={selected}
			tabIndex={0}
			aria-label={`Select token index ${payload.index}`}
			onClick={(event) => {
				event.stopPropagation();
				selectTokenCount();
			}}
			onKeyDown={(event) => {
				if (event.key === "Enter" || event.key === " ") {
					event.preventDefault();
					selectTokenCount();
				}
			}}
		/>
	);
}

export function RightPanel({
	statistics,
	tokenCounts,
	nodes,
	selectedTokenCountIndices,
	onTokenLogClick,
}: RightPanelProps) {
	const [tokenLogMode, setTokenLogMode] = useState<"cumulative" | "last">(
		"cumulative",
	);

	// ターンごとのツール呼び出し回数を nodes から集計
	const toolCallsPerTurn = useMemo(() => {
		const counts = Array(statistics.turn_count || 0).fill(0);
		for (const node of nodes) {
			if (
				node.type === "action" &&
				node.data?.turnIndex !== undefined &&
				node.data.turnIndex >= 0 &&
				node.data.turnIndex < counts.length
			) {
				counts[node.data.turnIndex]++;
			}
		}
		return counts;
	}, [nodes, statistics.turn_count]);

	const nodeIds = useMemo(() => new Set(nodes.map((node) => node.id)), [nodes]);
	const selectedTokenIndices = useMemo(
		() => new Set(selectedTokenCountIndices),
		[selectedTokenCountIndices],
	);

	// グラフ用データの整形
	const chartData = useMemo(() => {
		return (statistics.turns || []).map((turn) => {
			const idx = turn.index;
			return {
				name: `T${idx + 1}`,
				input: Number(turn.consumed_tokens?.input_tokens ?? 0),
				output: Number(turn.consumed_tokens?.output_tokens ?? 0),
				reasoning: Number(turn.consumed_tokens?.reasoning_output_tokens ?? 0),
				total: Number(turn.consumed_tokens?.total_tokens ?? 0),
				toolCalls: toolCallsPerTurn[idx] || 0,
				duration: (Number(turn.duration_ms) / 1000).toFixed(1), // 秒単位に変換
			};
		});
	}, [statistics.turns, toolCallsPerTurn]);

	// Last Token 用の折れ線グラフ用データ整形
	const lastTokenChartData = useMemo(() => {
		return tokenCounts.map((entry) => {
			const usage = entry.last_token_usage;
			return {
				name: `#${entry.index}`,
				index: entry.index,
				nodeId:
					entry.bound_to_node_id && nodeIds.has(entry.bound_to_node_id)
						? entry.bound_to_node_id
						: undefined,
				total: usage ? Number(usage.total_tokens) : 0,
				input: usage ? Number(usage.input_tokens) : 0,
				output: usage ? Number(usage.output_tokens) : 0,
				reasoning: usage ? Number(usage.reasoning_output_tokens) : 0,
				cached: usage ? Number(usage.cached_input_tokens) : 0,
			};
		});
	}, [nodeIds, tokenCounts]);

	const handleRowKeyDown = (event: React.KeyboardEvent, tokenIndex: number) => {
		if ((event.key === "Enter" || event.key === " ") && onTokenLogClick) {
			event.preventDefault();
			onTokenLogClick(tokenIndex);
		}
	};

	// 単位フォーマッタ
	const formatDuration = (ms: number) => {
		if (ms < 1000) return `${ms}ms`;
		const sec = ms / 1000;
		if (sec < 60) return `${sec.toFixed(1)}s`;
		const min = Math.floor(sec / 60);
		const remSec = Math.floor(sec % 60);
		return `${min}m ${remSec}s`;
	};

	const formatNumber = (num: number) => {
		return new Intl.NumberFormat().format(num);
	};

	return (
		<div className={styles.rightPanel}>
			<h2 className={styles.sectionTitle}>Session Analytics</h2>

			{/* 1. StatisticsPanel (6 stat cards) */}
			<div className={styles.statsGrid}>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Duration</span>
					<span className={styles.statValue}>
						{formatDuration(Number(statistics.duration_ms))}
					</span>
				</div>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Total Tokens</span>
					<span className={styles.statValue}>
						{formatNumber(Number(statistics.total_tokens))}
					</span>
				</div>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Tool Calls</span>
					<span className={styles.statValue}>{statistics.tool_call_count}</span>
				</div>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Token Counts</span>
					<span className={styles.statValue}>
						{statistics.token_count_count}
					</span>
				</div>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Context Window</span>
					<span className={styles.statValue}>
						{formatNumber(Number(statistics.context_window_size))}
					</span>
				</div>
				<div className={styles.statCard}>
					<span className={styles.statLabel}>Turns</span>
					<span className={styles.statValue}>{statistics.turn_count}</span>
				</div>
			</div>

			{/* Turn Duration Bar Sparkline */}
			{chartData.length > 0 && (
				<div className={styles.sparklineSection}>
					<h3 className={styles.subsectionTitle}>Turn Duration (seconds)</h3>
					<div className={styles.chartWrapperSpark}>
						<ResponsiveContainer width="100%" height={80}>
							<BarChart data={chartData}>
								<CartesianGrid
									strokeDasharray="3 3"
									stroke="var(--border-color)"
									vertical={false}
								/>
								<XAxis
									dataKey="name"
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
								/>
								<YAxis
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
									width={25}
								/>
								<Tooltip
									contentStyle={{
										background: "var(--bg-header)",
										border: "1px solid var(--border-color)",
										borderRadius: "4px",
										color: "var(--text-primary)",
										fontSize: "11px",
									}}
								/>
								<Bar
									dataKey="duration"
									fill="var(--color-accent)"
									style={{ fill: "var(--color-accent)" }}
									radius={[2, 2, 0, 0]}
									name="Duration (s)"
								/>
							</BarChart>
						</ResponsiveContainer>
					</div>
				</div>
			)}

			{/* 2. InteractiveCharts (Token consumption and Tool calls) */}
			{chartData.length > 0 && (
				<div className={styles.chartsSection}>
					<h3 className={styles.subsectionTitle}>Token Consumption per Turn</h3>
					<div className={styles.chartWrapper}>
						<ResponsiveContainer width="100%" height={180}>
							<BarChart
								data={chartData}
								margin={{ top: 5, right: 5, left: 0, bottom: 5 }}
							>
								<CartesianGrid
									strokeDasharray="3 3"
									stroke="var(--border-color)"
									vertical={false}
								/>
								<XAxis
									dataKey="name"
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
								/>
								<YAxis
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
									width={60}
								/>
								<Tooltip
									contentStyle={{
										background: "var(--bg-header)",
										border: "1px solid var(--border-color)",
										borderRadius: "4px",
										color: "var(--text-primary)",
										fontSize: "11px",
									}}
								/>
								<Legend wrapperStyle={{ fontSize: 10 }} />
								{/* Stacked bar chart colors align with node themes: input=blue, output=green, reasoning=purple */}
								<Bar
									dataKey="input"
									name="Input"
									stackId="tokens"
									fill="var(--node-input-text)"
									className={styles.barInput}
								/>
								<Bar
									dataKey="output"
									name="Output"
									stackId="tokens"
									fill="var(--node-output-text)"
									className={styles.barOutput}
								/>
								<Bar
									dataKey="reasoning"
									name="Reasoning"
									stackId="tokens"
									fill="var(--node-think-text)"
									className={styles.barReasoning}
								/>
							</BarChart>
						</ResponsiveContainer>
					</div>

					<h3 className={styles.subsectionTitle}>Tool Calls per Turn</h3>
					<div className={styles.chartWrapper}>
						<ResponsiveContainer width="100%" height={150}>
							<LineChart
								data={chartData}
								margin={{ top: 5, right: 5, left: -20, bottom: 5 }}
							>
								<CartesianGrid
									strokeDasharray="3 3"
									stroke="var(--border-color)"
									vertical={false}
								/>
								<XAxis
									dataKey="name"
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
								/>
								<YAxis
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
									allowDecimals={false}
								/>
								<Tooltip
									contentStyle={{
										background: "var(--bg-header)",
										border: "1px solid var(--border-color)",
										borderRadius: "4px",
										color: "var(--text-primary)",
										fontSize: "11px",
									}}
								/>
								<Line
									type="monotone"
									dataKey="toolCalls"
									name="Tool Calls"
									stroke="var(--node-action-text)"
									style={{ stroke: "var(--node-action-text)" }}
									strokeWidth={2}
									dot={{ r: 3 }}
									activeDot={{ r: 5 }}
								/>
							</LineChart>
						</ResponsiveContainer>
					</div>

					<h3 className={styles.subsectionTitle}>
						Last Token Consumption per Index
					</h3>
					<div className={styles.chartWrapper} data-testid="last-token-chart">
						<ResponsiveContainer width="100%" height={180}>
							<LineChart
								data={lastTokenChartData}
								margin={{ top: 5, right: 5, left: 0, bottom: 5 }}
							>
								<CartesianGrid
									strokeDasharray="3 3"
									stroke="var(--border-color)"
									vertical={false}
								/>
								<XAxis
									dataKey="name"
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
								/>
								<YAxis
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
									width={60}
								/>
								<Tooltip
									contentStyle={{
										background: "var(--bg-header)",
										border: "1px solid var(--border-color)",
										borderRadius: "4px",
										color: "var(--text-primary)",
										fontSize: "11px",
									}}
								/>
								<Legend wrapperStyle={{ fontSize: 10 }} />
								<Line
									type="monotone"
									dataKey="input"
									name="Input"
									stroke="var(--node-input-text)"
									style={{ stroke: "var(--node-input-text)" }}
									strokeWidth={1.5}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="input"
											radius={2}
											selected={selectedTokenIndices.has(props.payload?.index)}
											onSelect={onTokenLogClick}
										/>
									)}
									activeDot={false}
								/>
								<Line
									type="monotone"
									dataKey="output"
									name="Output"
									stroke="var(--node-output-text)"
									style={{ stroke: "var(--node-output-text)" }}
									strokeWidth={1.5}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="output"
											radius={2}
											selected={selectedTokenIndices.has(props.payload?.index)}
											onSelect={onTokenLogClick}
										/>
									)}
									activeDot={false}
								/>
								<Line
									type="monotone"
									dataKey="reasoning"
									name="Reasoning"
									stroke="var(--node-think-text)"
									style={{ stroke: "var(--node-think-text)" }}
									strokeWidth={1.5}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="reasoning"
											radius={2}
											selected={selectedTokenIndices.has(props.payload?.index)}
											onSelect={onTokenLogClick}
										/>
									)}
									activeDot={false}
								/>
								<Line
									type="monotone"
									dataKey="cached"
									name="Cached"
									stroke="var(--node-action-text)"
									style={{ stroke: "var(--node-action-text)" }}
									strokeWidth={1.5}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="cached"
											radius={2}
											selected={selectedTokenIndices.has(props.payload?.index)}
											onSelect={onTokenLogClick}
										/>
									)}
									activeDot={false}
								/>
								<Line
									type="monotone"
									dataKey="total"
									name="Total"
									stroke="var(--color-accent)"
									style={{ stroke: "var(--color-accent)" }}
									strokeWidth={2}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="total"
											radius={3}
											selected={selectedTokenIndices.has(props.payload?.index)}
											onSelect={onTokenLogClick}
										/>
									)}
									activeDot={false}
								/>
							</LineChart>
						</ResponsiveContainer>
					</div>
				</div>
			)}

			{/* 3. TurnTokenSummary */}
			<div className={styles.summarySection}>
				<h3 className={styles.subsectionTitle}>Turn Summary Cards</h3>
				<div className={styles.turnsContainer}>
					{(statistics.turns || []).map((turn) => {
						const consumedTokens = turn.consumed_tokens;
						return (
							<div key={turn.index} className={styles.turnCard}>
								<div className={styles.turnCardHeader}>
									<span className={styles.turnIndexLabel}>
										Turn #{turn.index + 1}
									</span>
									<span className={styles.turnModeBadge}>
										{turn.collaboration_mode_kind || "normal"}
									</span>
								</div>
								<div className={styles.turnCardDetails}>
									<div className={styles.detailRow}>
										<span>Duration:</span>
										<span>{formatDuration(Number(turn.duration_ms))}</span>
									</div>
									<div className={styles.detailRow}>
										<span>TTFT:</span>
										<span>
											{formatDuration(Number(turn.time_to_first_token_ms))}
										</span>
									</div>
									<div className={styles.tokenTitle}>Consumed Tokens</div>
									<div className={styles.tokenGrid}>
										<div className={styles.tokenSub}>
											<span>Total</span>
											<span>
												{formatNumber(
													Number(consumedTokens?.total_tokens ?? 0),
												)}
											</span>
										</div>
										<div className={styles.tokenSub}>
											<span>Input</span>
											<span>
												{formatNumber(
													Number(consumedTokens?.input_tokens ?? 0),
												)}
											</span>
										</div>
										<div className={styles.tokenSub}>
											<span>Output</span>
											<span>
												{formatNumber(
													Number(consumedTokens?.output_tokens ?? 0),
												)}
											</span>
										</div>
										<div className={styles.tokenSub}>
											<span>Reasoning</span>
											<span>
												{formatNumber(
													Number(consumedTokens?.reasoning_output_tokens ?? 0),
												)}
											</span>
										</div>
									</div>
								</div>
							</div>
						);
					})}
				</div>
			</div>

			{/* 4. TokenCountTable */}
			{tokenCounts.length > 0 && (
				<div className={styles.tableSection}>
					<div className={styles.tableHeaderArea}>
						<h3 className={styles.subsectionTitle}>Token Count Log</h3>
						<div className={styles.modeToggle}>
							<button
								type="button"
								className={`${styles.toggleBtn} ${
									tokenLogMode === "cumulative" ? styles.toggleBtnActive : ""
								}`}
								onClick={() => setTokenLogMode("cumulative")}
							>
								Cumulative
							</button>
							<button
								type="button"
								className={`${styles.toggleBtn} ${
									tokenLogMode === "last" ? styles.toggleBtnActive : ""
								}`}
								onClick={() => setTokenLogMode("last")}
							>
								Step (Last)
							</button>
						</div>
					</div>
					<div className={styles.tableWrapper}>
						<table className={styles.tokenTable}>
							<thead>
								<tr>
									<th>Index</th>
									<th>Total</th>
									<th>Input</th>
									<th>Output</th>
									<th>Reasoning</th>
									<th>Cached</th>
								</tr>
							</thead>
							<tbody>
								{tokenCounts.map((entry, idx) => {
									const showSeparator =
										idx > 0 &&
										tokenCounts[idx - 1].turn_index !== entry.turn_index;
									const targetNodeId =
										entry.bound_to_node_id &&
										nodeIds.has(entry.bound_to_node_id)
											? entry.bound_to_node_id
											: undefined;
									const usage =
										tokenLogMode === "cumulative"
											? entry.total_token_usage
											: entry.last_token_usage;

									const selected = selectedTokenIndices.has(entry.index);

									return (
										<React.Fragment key={entry.index}>
											{showSeparator && (
												<tr className={styles.separatorRow}>
													<td colSpan={6} className={styles.separatorCell}>
														Turn #{entry.turn_index + 1}
													</td>
												</tr>
											)}
											<tr
												className={`${targetNodeId && onTokenLogClick ? styles.tokenRow : ""} ${
													selected ? styles.selectedTokenRow : ""
												}`}
												aria-current={selected ? "true" : undefined}
												data-selected={selected}
												data-testid={`token-row-${entry.index}`}
												onClick={() => {
													if (targetNodeId && onTokenLogClick) {
														onTokenLogClick(entry.index);
													}
												}}
												onKeyDown={(e) => {
													if (targetNodeId) {
														handleRowKeyDown(e, entry.index);
													}
												}}
												tabIndex={
													targetNodeId && onTokenLogClick ? 0 : undefined
												}
												role={
													targetNodeId && onTokenLogClick ? "button" : undefined
												}
											>
												<td>{entry.index}</td>
												<td>
													{usage
														? formatNumber(Number(usage.total_tokens))
														: "-"}
												</td>
												<td>
													{usage
														? formatNumber(Number(usage.input_tokens))
														: "-"}
												</td>
												<td>
													{usage
														? formatNumber(Number(usage.output_tokens))
														: "-"}
												</td>
												<td>
													{usage
														? formatNumber(
																Number(usage.reasoning_output_tokens),
															)
														: "-"}
												</td>
												<td>
													{usage
														? formatNumber(Number(usage.cached_input_tokens))
														: "-"}
												</td>
											</tr>
										</React.Fragment>
									);
								})}
							</tbody>
						</table>
					</div>
				</div>
			)}
		</div>
	);
}
