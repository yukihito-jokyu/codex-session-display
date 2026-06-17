import type React from "react";
import { useEffect, useMemo, useState } from "react";
import styles from "./DateTree.module.css";
import { SessionRow, type SessionSummary } from "./SessionRow";

interface DateTreeProps {
	sessions: SessionSummary[];
}

interface GroupedSessionEntry {
	session: SessionSummary;
	sortKey: string;
}

interface GroupedDays {
	[day: string]: {
		count: number;
		sessions: GroupedSessionEntry[];
	};
}

const getGroupingDate = (session: SessionSummary) => {
	const candidates = [session.timestamp, session.file_modified_at];
	for (const candidate of candidates) {
		if (!candidate) continue;
		const date = new Date(candidate);
		if (!Number.isNaN(date.getTime())) {
			return date;
		}
	}
	return null;
};

export const DateTree: React.FC<DateTreeProps> = ({ sessions }) => {
	const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => {
		const saved = sessionStorage.getItem("session_list_expanded_paths");
		if (saved) {
			try {
				const paths = JSON.parse(saved);
				if (Array.isArray(paths)) {
					return new Set(paths);
				}
			} catch (e) {
				console.error("Failed to parse expanded paths", e);
			}
		}
		return new Set();
	});

	// sessionsMap を作成
	const sessionsMap = useMemo(() => {
		const map = new Map<string, SessionSummary>();
		for (const s of sessions) {
			map.set(s.id, s);
		}
		return map;
	}, [sessions]);

	// 親が sessions リストの中にいる子セッションを日付ツリーのルートから除外
	const rootSessions = useMemo(() => {
		return sessions.filter((s) => {
			if (!s.parent_session_id) return true;
			return !sessionsMap.has(s.parent_session_id);
		});
	}, [sessions, sessionsMap]);

	// セッションを日ごとにグループ化
	const groupedDays = useMemo(() => {
		const result: GroupedDays = {};
		rootSessions.forEach((s) => {
			const date = getGroupingDate(s);
			if (!date) return;

			const d = date.getDate().toString().padStart(2, "0");
			const sortKey = date.toISOString();

			if (!result[d]) {
				result[d] = { count: 0, sessions: [] };
			}

			result[d].count++;
			result[d].sessions.push({ session: s, sortKey });
		});
		return result;
	}, [rootSessions]);

	// 状態が変更されたら sessionStorage を更新する
	useEffect(() => {
		sessionStorage.setItem(
			"session_list_expanded_paths",
			JSON.stringify(Array.from(expandedPaths)),
		);
	}, [expandedPaths]);

	// 初回ロード時に最新の「日」をデフォルトで展開する
	useEffect(() => {
		if (sessions.length === 0) return;

		// すでに sessionStorage に保存されていた場合はデフォルト展開をスキップ
		const saved = sessionStorage.getItem("session_list_expanded_paths");
		if (saved) {
			return;
		}

		const days = Object.keys(groupedDays).sort((a, b) => b.localeCompare(a));
		if (days.length === 0) return;
		const latestDay = days[0];

		const initialExpanded = new Set<string>();
		initialExpanded.add(latestDay);
		setExpandedPaths(initialExpanded);
	}, [sessions, groupedDays]);

	const togglePath = (path: string) => {
		const nextExpanded = new Set(expandedPaths);
		if (nextExpanded.has(path)) {
			nextExpanded.delete(path);
		} else {
			nextExpanded.add(path);
		}
		setExpandedPaths(nextExpanded);
	};

	const sortedDays = Object.keys(groupedDays).sort((a, b) =>
		b.localeCompare(a),
	);

	if (sessions.length === 0 || sortedDays.length === 0) {
		return (
			<div className={styles.empty}>
				<div className={styles.emptyIcon}>📂</div>
				<div className={styles.emptyText}>
					No sessions found matching filters.
				</div>
			</div>
		);
	}

	return (
		<div className={styles.tree}>
			{sortedDays.map((day) => {
				const dayExpanded = expandedPaths.has(day);
				const daySessions = groupedDays[day].sessions;

				// 念のため、日のセッションをタイムスタンプの降順でソートする
				const sortedSessions = [...daySessions].sort((a, b) => {
					return b.sortKey.localeCompare(a.sortKey);
				});

				return (
					<div key={day} className={styles.dayNode}>
						<div
							className={styles.header}
							onClick={() => togglePath(day)}
							onKeyDown={(e) => {
								if (e.key === "Enter" || e.key === " ") {
									e.preventDefault();
									togglePath(day);
								}
							}}
							role="button"
							tabIndex={0}
						>
							<span
								className={`${styles.arrow} ${dayExpanded ? styles.open : ""}`}
							>
								▶
							</span>
							<span className={styles.label}>{parseInt(day, 10)}日</span>
							<span className={styles.countBadge}>
								{groupedDays[day].count}
							</span>
						</div>

						{dayExpanded && (
							<div className={styles.sessionsList}>
								{sortedSessions.map(({ session }) => (
									<SessionRow
										key={session.id}
										session={session}
										sessionsMap={sessionsMap}
									/>
								))}
							</div>
						)}
					</div>
				);
			})}
		</div>
	);
};
