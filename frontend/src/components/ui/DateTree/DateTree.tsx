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

interface GroupedMonths {
	[month: string]: {
		count: number;
		days: GroupedDays;
	};
}

interface GroupedData {
	[year: string]: {
		count: number;
		months: GroupedMonths;
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
	const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());

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

	// セッションをグループ化
	const grouped = useMemo(() => {
		const result: GroupedData = {};
		rootSessions.forEach((s) => {
			const date = getGroupingDate(s);
			if (!date) return;

			const y = date.getFullYear().toString();
			const m = (date.getMonth() + 1).toString().padStart(2, "0");
			const d = date.getDate().toString().padStart(2, "0");
			const sortKey = date.toISOString();

			if (!result[y]) {
				result[y] = { count: 0, months: {} };
			}
			if (!result[y].months[m]) {
				result[y].months[m] = { count: 0, days: {} };
			}
			if (!result[y].months[m].days[d]) {
				result[y].months[m].days[d] = { count: 0, sessions: [] };
			}

			result[y].count++;
			result[y].months[m].count++;
			result[y].months[m].days[d].count++;
			result[y].months[m].days[d].sessions.push({ session: s, sortKey });
		});
		return result;
	}, [rootSessions]);

	// 初回ロード時に最新の年、月、日をデフォルトで展開する
	useEffect(() => {
		if (sessions.length === 0) return;

		const years = Object.keys(grouped).sort((a, b) => b.localeCompare(a));
		if (years.length === 0) return;
		const latestYear = years[0];

		const months = Object.keys(grouped[latestYear].months).sort((a, b) =>
			b.localeCompare(a),
		);
		const latestMonth = months[0];

		const days = Object.keys(grouped[latestYear].months[latestMonth].days).sort(
			(a, b) => b.localeCompare(a),
		);
		const latestDay = days[0];

		const initialExpanded = new Set<string>();
		initialExpanded.add(latestYear);
		initialExpanded.add(`${latestYear}/${latestMonth}`);
		initialExpanded.add(`${latestYear}/${latestMonth}/${latestDay}`);
		setExpandedPaths(initialExpanded);
	}, [sessions, grouped]);

	const togglePath = (path: string) => {
		const nextExpanded = new Set(expandedPaths);
		if (nextExpanded.has(path)) {
			nextExpanded.delete(path);
		} else {
			nextExpanded.add(path);
		}
		setExpandedPaths(nextExpanded);
	};

	const sortedYears = Object.keys(grouped).sort((a, b) => b.localeCompare(a));

	if (sessions.length === 0 || sortedYears.length === 0) {
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
			{sortedYears.map((year) => {
				const yearExpanded = expandedPaths.has(year);
				const yearMonths = grouped[year].months;
				const sortedMonths = Object.keys(yearMonths).sort((a, b) =>
					b.localeCompare(a),
				);

				return (
					<div key={year} className={styles.yearNode}>
						<div
							className={styles.header}
							onClick={() => togglePath(year)}
							onKeyDown={(e) => {
								if (e.key === "Enter" || e.key === " ") {
									e.preventDefault();
									togglePath(year);
								}
							}}
							role="button"
							tabIndex={0}
						>
							<span
								className={`${styles.arrow} ${yearExpanded ? styles.open : ""}`}
							>
								▶
							</span>
							<span className={styles.label}>{year}年</span>
							<span className={styles.countBadge}>{grouped[year].count}</span>
						</div>

						{yearExpanded && (
							<div className={styles.children}>
								{sortedMonths.map((month) => {
									const monthPath = `${year}/${month}`;
									const monthExpanded = expandedPaths.has(monthPath);
									const monthDays = yearMonths[month].days;
									const sortedDays = Object.keys(monthDays).sort((a, b) =>
										b.localeCompare(a),
									);

									return (
										<div key={month} className={styles.monthNode}>
											<div
												className={styles.header}
												onClick={() => togglePath(monthPath)}
												onKeyDown={(e) => {
													if (e.key === "Enter" || e.key === " ") {
														e.preventDefault();
														togglePath(monthPath);
													}
												}}
												role="button"
												tabIndex={0}
											>
												<span
													className={`${styles.arrow} ${monthExpanded ? styles.open : ""}`}
												>
													▶
												</span>
												<span className={styles.label}>
													{parseInt(month, 10)}月
												</span>
												<span className={styles.countBadge}>
													{yearMonths[month].count}
												</span>
											</div>

											{monthExpanded && (
												<div className={styles.children}>
													{sortedDays.map((day) => {
														const dayPath = `${monthPath}/${day}`;
														const dayExpanded = expandedPaths.has(dayPath);
														const daySessions = monthDays[day].sessions;

														// 念のため、日のセッションをタイムスタンプの降順でソートする
														const sortedSessions = [...daySessions].sort(
															(a, b) => {
																return b.sortKey.localeCompare(a.sortKey);
															},
														);

														return (
															<div key={day} className={styles.dayNode}>
																<div
																	className={styles.header}
																	onClick={() => togglePath(dayPath)}
																	onKeyDown={(e) => {
																		if (e.key === "Enter" || e.key === " ") {
																			e.preventDefault();
																			togglePath(dayPath);
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
																	<span className={styles.label}>
																		{parseInt(day, 10)}日
																	</span>
																	<span className={styles.countBadge}>
																		{monthDays[day].count}
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
											)}
										</div>
									);
								})}
							</div>
						)}
					</div>
				);
			})}
		</div>
	);
};
