import type React from "react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { generateTokenBreakdownText } from "../../../utils/tokenCopy";
import styles from "./SessionRow.module.css";

export interface SessionSummary {
	id: string;
	file_path: string;
	cwd?: string | null;
	cli_version?: string | null;
	originator?: string | null;
	model_provider?: string | null;
	branch?: string | null;
	source?: string | null;
	timestamp?: string | null;
	file_size: number;
	file_modified_at?: string | null;
	parsed: boolean;
	parent_session_id?: string | null;
	child_session_ids?: string[] | null;
	total_tokens?: number | null;
	input_tokens?: number | null;
	output_tokens?: number | null;
	reasoning_tokens?: number | null;
	turn_count?: number | null;
	step_count?: number | null;
	duration_ms?: number | null;
}

interface SessionRowProps {
	session: SessionSummary;
	sessionsMap?: Map<string, SessionSummary>;
	depth?: number;
	isParsing?: boolean;
	parsingSessionIds?: Set<string>;
}

const formatTimestamp = (isoStr: string | null | undefined) => {
	if (!isoStr) return "—";
	try {
		const d = new Date(isoStr);
		if (Number.isNaN(d.getTime())) return isoStr;

		const pad = (n: number) => n.toString().padStart(2, "0");
		const year = d.getFullYear();
		const month = pad(d.getMonth() + 1);
		const date = pad(d.getDate());
		const hours = pad(d.getHours());
		const minutes = pad(d.getMinutes());
		const seconds = pad(d.getSeconds());

		return `${year}-${month}-${date} ${hours}:${minutes}:${seconds}`;
	} catch (_e) {
		return isoStr;
	}
};

const formatFileSize = (bytes: number) => {
	if (bytes === 0) return "0 B";
	const k = 1024;
	const sizes = ["B", "KB", "MB", "GB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i]}`;
};

export const SessionRow: React.FC<SessionRowProps> = ({
	session,
	sessionsMap,
	depth = 0,
	isParsing = false,
	parsingSessionIds,
}) => {
	const navigate = useNavigate();
	const [expanded, setExpanded] = useState(false);
	const [copied, setCopied] = useState(false);

	const handleClick = () => {
		navigate(`/sessions/${session.id}`);
	};

	const handleCopyBreakdown = (e: React.MouseEvent) => {
		e.stopPropagation();

		const subagentInfos: {
			nickname: string;
			total: number;
			input: number;
			output: number;
			turnCount?: number | null;
			stepCount?: number | null;
			durationMs?: number | null;
		}[] = [];
		const visited = new Set<string>();

		const collect = (id: string, isRoot: boolean) => {
			if (visited.has(id)) return;
			visited.add(id);

			const current = sessionsMap?.get(id);
			if (!current) return;

			if (!isRoot) {
				subagentInfos.push({
					nickname: current.originator || "Subagent",
					total: current.total_tokens ?? 0,
					input: current.input_tokens ?? 0,
					output: current.output_tokens ?? 0,
					turnCount: current.turn_count,
					stepCount: current.step_count,
					durationMs: current.duration_ms,
				});
			}

			if (current.child_session_ids) {
				for (const cid of current.child_session_ids) {
					collect(cid, false);
				}
			}
		};

		collect(session.id, true);

		const mainTokens = {
			total: session.total_tokens ?? 0,
			input: session.input_tokens ?? 0,
			output: session.output_tokens ?? 0,
			turnCount: session.turn_count,
			stepCount: session.step_count,
			durationMs: session.duration_ms,
		};

		const text = generateTokenBreakdownText(
			session.id,
			mainTokens,
			subagentInfos,
		);
		navigator.clipboard.writeText(text).then(() => {
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		});
	};

	const isParsed = session.parsed;

	const children = useMemo(() => {
		if (!session.child_session_ids || !sessionsMap) return [];
		return session.child_session_ids
			.map((id) => sessionsMap.get(id))
			.filter((s): s is SessionSummary => !!s);
	}, [session.child_session_ids, sessionsMap]);

	return (
		<div className={styles.container}>
			<div
				className={`${styles.row} ${isParsed ? styles.parsed : styles.unparsed}`}
				onClick={handleClick}
				role="button"
				tabIndex={0}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.key === " ") {
						e.preventDefault();
						handleClick();
					}
				}}
				style={{ paddingLeft: "12px" }}
			>
				<div className={styles.topContent}>
					{children.length > 0 && (
						<button
							type="button"
							className={`${styles.toggleBtn} ${expanded ? styles.open : ""}`}
							onClick={(e) => {
								e.stopPropagation();
								setExpanded(!expanded);
							}}
							aria-label={expanded ? "Collapse subagents" : "Expand subagents"}
						>
							▶
						</button>
					)}
					<div className={styles.mainInfo}>
						<div className={styles.header}>
							<span className={styles.sessionId} title={session.id}>
								{session.id.slice(0, 8)}
							</span>
							<span className={styles.branch}>
								{isParsing ? (
									<span className={styles.parsing}>解析中...</span>
								) : isParsed ? (
									session.branch || "—"
								) : (
									<span className={styles.beforeParse}>解析前</span>
								)}
							</span>
							{isParsing ? (
								<span className={styles.parsingBadge}>解析中</span>
							) : (
								!isParsed && (
									<span className={styles.unparsedBadge}>未解析</span>
								)
							)}
						</div>
						<div
							className={styles.cwd}
							title={
								isParsing
									? "解析中..."
									: isParsed
										? session.cwd || ""
										: "解析前"
							}
						>
							{isParsing ? (
								<span className={styles.parsing}>解析中...</span>
							) : isParsed ? (
								session.cwd || "—"
							) : (
								<span className={styles.beforeParse}>解析前</span>
							)}
						</div>
					</div>

					<div className={styles.metaInfo}>
						<div className={styles.metaRow}>
							<span className={styles.provider}>
								{isParsing ? (
									<span className={styles.parsing}>解析中...</span>
								) : isParsed ? (
									session.model_provider || "—"
								) : (
									<span className={styles.beforeParse}>解析前</span>
								)}
							</span>
							<span className={styles.version}>
								{isParsing ? (
									<span className={styles.parsing}>解析中...</span>
								) : isParsed ? (
									session.cli_version ? (
										`v${session.cli_version}`
									) : (
										"—"
									)
								) : (
									<span className={styles.beforeParse}>解析前</span>
								)}
							</span>
						</div>
						<div className={styles.timeRow}>
							<span className={styles.time}>
								{formatTimestamp(session.timestamp)}
							</span>
							<span className={styles.size}>
								{formatFileSize(session.file_size)}
							</span>
						</div>
					</div>
				</div>

				<div className={styles.statsBar}>
					{isParsing ? (
						<span className={styles.parsing}>解析中...</span>
					) : !isParsed ? (
						<span className={styles.beforeParse}>解析前</span>
					) : (
						<>
							<span className={styles.statsItem}>
								<span className={styles.statsIcon}>🪙</span>
								トークン:{" "}
								<strong className={styles.statsValue}>
									{(session.total_tokens ?? 0).toLocaleString()}
								</strong>
								<span className={styles.statsBreakdown}>
									(入力:{" "}
									<span className={styles.tokenInput}>
										{(session.input_tokens ?? 0).toLocaleString()}
									</span>{" "}
									/ 出力:{" "}
									<span className={styles.tokenOutput}>
										{(session.output_tokens ?? 0).toLocaleString()}
									</span>{" "}
									/ 推論:{" "}
									<span className={styles.tokenReasoning}>
										{(session.reasoning_tokens ?? 0).toLocaleString()}
									</span>
									)
								</span>
							</span>
							<span className={styles.statsDivider}>|</span>
							<span className={styles.statsItem}>
								<span className={styles.statsIcon}>🔄</span>
								ターン数:{" "}
								<strong className={styles.statsValue}>
									{session.turn_count ?? 0}
								</strong>
							</span>
							<span className={styles.statsDivider}>|</span>
							<span className={styles.statsItem}>
								<span className={styles.statsIcon}>🛠️</span>
								ステップ数:{" "}
								<strong className={styles.statsValue}>
									{session.step_count ?? 0}
								</strong>
							</span>

							{/* ホバー時に表示されるコピーボタン */}
							<button
								type="button"
								className={styles.rowCopyBtn}
								onClick={handleCopyBreakdown}
								title="トークン消費の内訳をコピー"
								aria-label="Copy token breakdown"
							>
								{copied ? "✓ コピー済" : "📋 コピー"}
							</button>
						</>
					)}
				</div>
			</div>
			{expanded && children.length > 0 && (
				<div className={styles.childRows}>
					{children.map((child) => (
						<SessionRow
							key={child.id}
							session={child}
							sessionsMap={sessionsMap}
							depth={depth + 1}
							isParsing={parsingSessionIds?.has(child.id)}
							parsingSessionIds={parsingSessionIds}
						/>
					))}
				</div>
			)}
		</div>
	);
};
