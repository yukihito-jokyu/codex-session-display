import React, { useMemo, useState } from "react";
import type { TooltipContentProps } from "recharts";
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
import { SaveChartImage } from "wailsjs/go/main/App";
import type { dto } from "wailsjs/go/models";
import { generateTokenBreakdownText } from "../../../../utils/tokenCopy";
import styles from "./RightPanel.module.css";

interface RightPanelProps {
	sessionId: string;
	statistics: dto.Statistics;
	transcriptStats?: dto.TranscriptStats;
	tokenCounts: dto.TokenCountEntry[];
	nodes: dto.FlowNode[];
	selectedTokenCountIndices: number[];
	onTokenLogClick?: (tokenIndex: number) => void;
	subagents?: dto.SubagentDetail[];
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
	contextUsage?: number;
}

interface TokenChartDotProps {
	cx?: number;
	cy?: number;
	stroke?: string;
	payload?: LastTokenChartData;
	series: "input" | "output" | "reasoning" | "cached" | "total" | "context";
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

function calculateContextUsage(
	inputTokens: number,
	modelContextWindow: number,
): number | undefined {
	if (inputTokens <= 0 || modelContextWindow <= 0) {
		return undefined;
	}
	return (inputTokens / modelContextWindow) * 100;
}

function formatContextUsage(value: number | undefined): string {
	if (value === undefined) {
		return "N/A";
	}
	return `${new Intl.NumberFormat(undefined, {
		maximumFractionDigits: 1,
	}).format(value)}%`;
}

function LastTokenTooltip({ active, payload, label }: TooltipContentProps) {
	if (!active || payload.length === 0) {
		return null;
	}

	const data = payload[0]?.payload as LastTokenChartData | undefined;
	if (!data) {
		return null;
	}

	const rows = [
		["Total", data.total.toLocaleString()],
		["Input", data.input.toLocaleString()],
		["Output", data.output.toLocaleString()],
		["Reasoning", data.reasoning.toLocaleString()],
		["Cached", data.cached.toLocaleString()],
	] as const;

	return (
		<div className={styles.chartTooltip}>
			<div className={styles.chartTooltipLabel}>{label}</div>
			{rows.map(([name, value]) => (
				<div key={name}>{`${name}: ${value}`}</div>
			))}
			<div data-testid="last-token-tooltip-context">
				{`Context Usage (%): ${formatContextUsage(data.contextUsage)}`}
			</div>
		</div>
	);
}

export function RightPanel({
	sessionId,
	statistics,
	transcriptStats,
	tokenCounts,
	nodes,
	selectedTokenCountIndices,
	onTokenLogClick,
	subagents = [],
}: RightPanelProps) {
	const [tokenLogMode, setTokenLogMode] = useState<"cumulative" | "last">(
		"cumulative",
	);
	const [copied, setCopied] = useState(false);

	const handleCopyTokens = (e: React.MouseEvent) => {
		e.stopPropagation();
		const mainTokens = {
			total: Number(statistics.total_tokens),
			input: statistics.turns
				? statistics.turns.reduce(
						(sum, t) => sum + (t.consumed_tokens?.input_tokens ?? 0),
						0,
					)
				: 0,
			output: statistics.turns
				? statistics.turns.reduce(
						(sum, t) => sum + (t.consumed_tokens?.output_tokens ?? 0),
						0,
					)
				: 0,
			turnCount: statistics.turn_count,
			stepCount: statistics.tool_call_count,
			durationMs: statistics.duration_ms,
		};

		const subagentInfos = subagents.map((s) => ({
			nickname: s.nickname,
			total: s.total_tokens,
			input: s.input_tokens,
			output: s.output_tokens,
			turnCount: s.turn_count,
			stepCount: s.step_count,
			durationMs: s.duration_ms,
		}));

		const text = generateTokenBreakdownText(
			sessionId,
			mainTokens,
			subagentInfos,
		);
		navigator.clipboard.writeText(text).then(() => {
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		});
	};

	const handleExportChart = async (e: React.MouseEvent) => {
		e.stopPropagation();

		const container = document.querySelector(
			'[data-testid="last-token-chart"]',
		);
		// 凡例内のSVGアイコンを選択しないよう、凡例以外のSVGを指定して取得
		const svg = container?.querySelector(
			"svg:not(.recharts-legend-wrapper svg)",
		);
		if (!svg) {
			console.error("Chart SVG element not found");
			return;
		}

		try {
			// クローンを作成して、CSS変数の実値をインライン化する
			const clonedSvg = svg.cloneNode(true) as SVGElement;

			// CSS変数を実際のカラー値に解決するヘルパー関数
			const resolveCssVariables = (val: string): string => {
				if (!val?.includes("var(")) {
					return val;
				}

				let resolved = val;
				// 最大3段階のネストに対応
				for (let depth = 0; depth < 3; depth++) {
					const matches = resolved.match(/var\((--[^)]+)\)/g);
					if (!matches) {
						break;
					}

					let replaced = false;
					for (const match of matches) {
						const varName = match.slice(4, -1).trim();
						let actualVal = window
							.getComputedStyle(document.documentElement)
							.getPropertyValue(varName)
							.trim();
						if (!actualVal) {
							actualVal = window
								.getComputedStyle(document.body)
								.getPropertyValue(varName)
								.trim();
						}

						if (actualVal) {
							resolved = resolved.replace(match, actualVal);
							replaced = true;
						}
					}

					if (!replaced) {
						break;
					}
				}
				return resolved;
			};

			// CSS変数を実際の値に変換する関数
			const inlineStyles = (source: Element, target: Element) => {
				const computed = window.getComputedStyle(source);

				// RechartsのSVG要素で使われがちな属性とスタイルを計算後の値に置換
				const styleProps = [
					"fill",
					"stroke",
					"color",
					"font-size",
					"font-family",
					"stroke-width",
					"stroke-dasharray",
					"opacity",
				];
				for (const prop of styleProps) {
					let val = computed.getPropertyValue(prop);
					if (val) {
						val = resolveCssVariables(val);
						(target as HTMLElement).style.setProperty(prop, val);
					}
				}

				// 子要素に対しても再帰的に適用
				for (let i = 0; i < source.children.length; i++) {
					inlineStyles(source.children[i], target.children[i]);
				}
			};

			inlineStyles(svg, clonedSvg);

			const scale = 2; // 高解像度（2倍）で出力
			const width = svg.clientWidth || svg.getBoundingClientRect().width || 500;
			const height =
				svg.clientHeight || svg.getBoundingClientRect().height || 180;

			// viewBoxが設定されていなければ設定する
			if (!svg.getAttribute("viewBox")) {
				clonedSvg.setAttribute("viewBox", `0 0 ${width} ${height}`);
			}
			clonedSvg.setAttribute("width", (width * scale).toString());
			clonedSvg.setAttribute("height", (height * scale).toString());

			// clonedSvgのインラインスタイルにあるwidth/heightが優先されて解像度が制限されるのを防ぐため、明示的にスケール後の値を設定する
			clonedSvg.style.removeProperty("width");
			clonedSvg.style.removeProperty("height");
			clonedSvg.style.width = `${width * scale}px`;
			clonedSvg.style.height = `${height * scale}px`;

			// テーマに応じた背景色を取得（透明背景を回避）
			const bodyStyle = window.getComputedStyle(document.body);
			let bgColor = bodyStyle.getPropertyValue("--bg-app").trim() || "#0f1117";
			if (bgColor.startsWith("rgba")) {
				const matches = bgColor.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
				if (matches) {
					bgColor = `rgb(${matches[1]}, ${matches[2]}, ${matches[3]})`;
				}
			}

			const serializer = new XMLSerializer();
			const svgString = serializer.serializeToString(clonedSvg);

			const img = new Image();
			img.width = width * scale;
			img.height = height * scale;

			const svgBlob = new Blob([svgString], {
				type: "image/svg+xml;charset=utf-8",
			});
			const url = URL.createObjectURL(svgBlob);

			img.onload = () => {
				const canvas = document.createElement("canvas");
				canvas.width = width * scale;
				canvas.height = height * scale;

				const ctx = canvas.getContext("2d");
				if (ctx) {
					ctx.imageSmoothingEnabled = true;
					ctx.imageSmoothingQuality = "high";
					ctx.fillStyle = bgColor;
					ctx.fillRect(0, 0, canvas.width, canvas.height);
					ctx.drawImage(img, 0, 0, canvas.width, canvas.height);

					try {
						const base64Data = canvas.toDataURL("image/png");
						SaveChartImage(
							base64Data,
							`token-consumption-chart-${sessionId}.png`,
						)
							.then(() => {
								console.log("Chart image exported successfully");
							})
							.catch((err) => {
								console.error("Save image failed", err);
								const errMsg = err instanceof Error ? err.message : String(err);
								alert(`画像の保存に失敗しました: ${errMsg}`);
							});
					} catch (e) {
						console.error("Canvas toDataURL failed", e);
						alert("画像の生成に失敗しました");
					}
				}
				URL.revokeObjectURL(url);
			};

			img.onerror = (err) => {
				console.error("Image load failed", err);
				URL.revokeObjectURL(url);
				alert("画像の読み込みに失敗しました");
			};

			img.src = url;
		} catch (error) {
			console.error("Failed to export chart image", error);
			alert("チャートのエクスポート中にエラーが発生しました");
		}
	};

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
				toolCalls: turn.tool_call_count ?? 0,
				duration: (Number(turn.duration_ms) / 1000).toFixed(1), // 秒単位に変換
			};
		});
	}, [statistics.turns]);

	// Last Token 用の折れ線グラフ用データ整形
	const lastTokenChartData = useMemo(() => {
		return tokenCounts.map((entry) => {
			const usage = entry.last_token_usage;
			const input = usage ? Number(usage.input_tokens) : 0;
			return {
				name: `#${entry.index}`,
				index: entry.index,
				nodeId:
					entry.bound_to_node_id && nodeIds.has(entry.bound_to_node_id)
						? entry.bound_to_node_id
						: undefined,
				total: usage ? Number(usage.total_tokens) : 0,
				input,
				output: usage ? Number(usage.output_tokens) : 0,
				reasoning: usage ? Number(usage.reasoning_output_tokens) : 0,
				cached: usage
					? Number(usage.cached_input_tokens) +
						Number(usage.cache_creation_input_tokens || 0)
					: 0,
				contextUsage: calculateContextUsage(
					input,
					Number(entry.model_context_window),
				),
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
				<div className={`${styles.statCard} ${styles.tokenStatCard}`}>
					<span className={styles.statLabel}>Total Tokens</span>
					<span className={styles.statValue}>
						{formatNumber(
							Number(statistics.total_tokens) +
								subagents.reduce((sum, s) => sum + s.total_tokens, 0),
						)}
					</span>
					<button
						type="button"
						className={styles.copyBtn}
						onClick={handleCopyTokens}
						title="トークン消費の内訳をコピー"
						aria-label="Copy token breakdown"
					>
						{copied ? "✓ コピー済" : "📋 コピー"}
					</button>
					{subagents.length > 0 && (
						<span className={styles.statSubValue}>
							(本体: {formatNumber(Number(statistics.total_tokens))} / 子:{" "}
							{formatNumber(
								subagents.reduce((sum, s) => sum + s.total_tokens, 0),
							)}
							)
						</span>
					)}
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
				{transcriptStats?.total_cost_usd != null && (
					<div className={styles.statCard}>
						<span className={styles.statLabel}>Cost</span>
						<span className={styles.statValue}>
							${transcriptStats.total_cost_usd.toFixed(4)}
						</span>
					</div>
				)}
				{transcriptStats?.cache_read_input_tokens != null &&
					transcriptStats.cache_read_input_tokens > 0 && (
						<div className={styles.statCard}>
							<span className={styles.statLabel}>Cache Read</span>
							<span className={styles.statValue}>
								{formatNumber(transcriptStats.cache_read_input_tokens)}
							</span>
						</div>
					)}
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

					<div className={styles.subsectionHeader}>
						<h3 className={styles.subsectionTitle}>
							Last Token Consumption per Index
						</h3>
						<button
							type="button"
							className={styles.exportBtn}
							onClick={handleExportChart}
							title="画像をエクスポート"
							aria-label="Export chart as image"
							data-testid="export-chart-button"
						>
							📤
						</button>
					</div>
					<div className={styles.chartWrapper} data-testid="last-token-chart">
						<ResponsiveContainer width="100%" height={180}>
							<LineChart
								data={lastTokenChartData}
								margin={{ top: 5, right: 0, left: 0, bottom: 5 }}
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
									yAxisId="tokens"
									stroke="var(--text-secondary)"
									fontSize={10}
									tickLine={false}
									width={60}
								/>
								<YAxis
									yAxisId="context"
									orientation="right"
									stroke="var(--node-warning-text)"
									fontSize={10}
									tickLine={false}
									width={42}
									domain={[0, "auto"]}
									tickFormatter={(value: number) => `${value}%`}
								/>
								<Tooltip content={LastTokenTooltip} />
								<Legend wrapperStyle={{ fontSize: 10 }} />
								<Line
									yAxisId="tokens"
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
									yAxisId="tokens"
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
									yAxisId="tokens"
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
									yAxisId="tokens"
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
									yAxisId="tokens"
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
								<Line
									yAxisId="context"
									type="monotone"
									dataKey="contextUsage"
									name="Context Usage (%)"
									stroke="var(--node-warning-text)"
									style={{ stroke: "var(--node-warning-text)" }}
									strokeWidth={2}
									connectNulls={false}
									dot={(props) => (
										<TokenChartDot
											{...props}
											series="context"
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
									<th>Cached Read</th>
									<th>Cached Write</th>
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
													<td colSpan={7} className={styles.separatorCell}>
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
												<td>
													{usage
														? formatNumber(
																Number(usage.cache_creation_input_tokens),
															)
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
