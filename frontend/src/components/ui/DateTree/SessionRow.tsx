import type React from "react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
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
}

interface SessionRowProps {
	session: SessionSummary;
	sessionsMap?: Map<string, SessionSummary>;
	depth?: number;
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
}) => {
	const navigate = useNavigate();
	const [expanded, setExpanded] = useState(false);

	const handleClick = () => {
		navigate(`/sessions/${session.id}`);
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
							{isParsed ? (
								session.branch || "—"
							) : (
								<span className={styles.beforeParse}>解析前</span>
							)}
						</span>
						{!isParsed && <span className={styles.unparsedBadge}>未解析</span>}
					</div>
					<div
						className={styles.cwd}
						title={isParsed ? session.cwd || "" : "解析前"}
					>
						{isParsed ? (
							session.cwd || "—"
						) : (
							<span className={styles.beforeParse}>解析前</span>
						)}
					</div>
				</div>

				<div className={styles.metaInfo}>
					<div className={styles.metaRow}>
						<span className={styles.provider}>
							{isParsed ? (
								session.model_provider || "—"
							) : (
								<span className={styles.beforeParse}>解析前</span>
							)}
						</span>
						<span className={styles.version}>
							{isParsed ? (
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
			{expanded && children.length > 0 && (
				<div className={styles.childRows}>
					{children.map((child) => (
						<SessionRow
							key={child.id}
							session={child}
							sessionsMap={sessionsMap}
							depth={depth + 1}
						/>
					))}
				</div>
			)}
		</div>
	);
};
