import { useEffect, useMemo, useState } from "react";
import { DatePicker } from "../../components/ui/DatePicker/DatePicker";
import { DateTree } from "../../components/ui/DateTree/DateTree";
import type { SessionSummary } from "../../components/ui/DateTree/SessionRow";
import { SessionRow } from "../../components/ui/DateTree/SessionRow";
import { Toolbar } from "../../components/ui/Toolbar/Toolbar";
import { UpdateModal } from "../../components/ui/UpdateModal/UpdateModal";
import { useSessions } from "./hooks/useSessions";
import { useUpdate } from "./hooks/useUpdate";
import styles from "./SessionListPage.module.css";

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

const getFormattedDate = (session: SessionSummary) => {
	const date = getGroupingDate(session);
	if (!date) return "";
	const y = date.getFullYear();
	const m = String(date.getMonth() + 1).padStart(2, "0");
	const d = String(date.getDate()).padStart(2, "0");
	return `${y}-${m}-${d}`;
};

export function SessionListPage() {
	const {
		sessions,
		loading,
		error,
		currentYear,
		currentMonth,
		setCurrentYear,
		setCurrentMonth,
		handleSearch,
		handlePrevMonth,
		handleNextMonth,
		retry,
		parsingSessionIds,
		parseSessions,
	} = useSessions();

	const { updateResult, progress, apply } = useUpdate();
	const [isModalOpen, setIsModalOpen] = useState(false);
	const [activeTab, setActiveTab] = useState<"history" | "directory">(() => {
		const saved = sessionStorage.getItem("session_list_active_tab");
		return saved === "history" || saved === "directory" ? saved : "history";
	});
	const [selectedDate, setSelectedDate] = useState<string>(() => {
		const saved = sessionStorage.getItem("session_list_selected_date");
		if (saved) return saved;

		const now = new Date();
		const y = now.getFullYear();
		const m = String(now.getMonth() + 1).padStart(2, "0");
		const d = String(now.getDate()).padStart(2, "0");
		return `${y}-${m}-${d}`;
	});

	useEffect(() => {
		sessionStorage.setItem("session_list_active_tab", activeTab);
	}, [activeTab]);

	useEffect(() => {
		sessionStorage.setItem("session_list_selected_date", selectedDate);
	}, [selectedDate]);

	// sessionsMap を作成 (親子関係の解決用)
	const sessionsMap = useMemo(() => {
		const map = new Map<string, SessionSummary>();
		for (const s of sessions) {
			map.set(s.id, s);
		}
		return map;
	}, [sessions]);

	// 選択された日付に該当するセッションをフィルタリング
	const filteredSessions = useMemo(() => {
		return sessions.filter((s) => getFormattedDate(s) === selectedDate);
	}, [sessions, selectedDate]);

	const filteredSessionsMap = useMemo(() => {
		const map = new Map<string, SessionSummary>();
		for (const s of filteredSessions) {
			map.set(s.id, s);
		}
		return map;
	}, [filteredSessions]);

	// フィルタリングされた結果のうち、ルートセッションのみを抽出
	const rootFilteredSessions = useMemo(() => {
		return filteredSessions.filter((s) => {
			if (!s.parent_session_id) return true;
			return !filteredSessionsMap.has(s.parent_session_id);
		});
	}, [filteredSessions, filteredSessionsMap]);

	// 日付変更時の処理
	const handleDateChange = (dateStr: string) => {
		setSelectedDate(dateStr);
		if (!dateStr) return;
		const [yearStr, monthStr] = dateStr.split("-");
		const y = parseInt(yearStr, 10);
		const m = parseInt(monthStr, 10);
		if (y && m) {
			if (y !== currentYear) {
				setCurrentYear(y);
			}
			if (m !== currentMonth) {
				setCurrentMonth(m);
			}
		}
	};

	return (
		<div className={styles.listPage}>
			<Toolbar
				totalCount={sessions.length}
				sourcePath="~/.codex/sessions"
				onSearch={handleSearch}
				hasUpdate={updateResult?.hasUpdate}
				latestVersion={updateResult?.latest}
				onUpdateClick={() => setIsModalOpen(true)}
			/>

			<div className={styles.tabsHeader}>
				<button
					type="button"
					className={`${styles.tabBtn} ${activeTab === "history" ? styles.activeTab : ""}`}
					onClick={() => setActiveTab("history")}
				>
					履歴ツリー
				</button>
				<button
					type="button"
					className={`${styles.tabBtn} ${activeTab === "directory" ? styles.activeTab : ""}`}
					onClick={() => setActiveTab("directory")}
				>
					ディレクトリ分類
				</button>
			</div>

			{activeTab === "history" ? (
				<>
					<div className={styles.monthNavigator}>
						<button
							type="button"
							className={styles.navBtn}
							onClick={handlePrevMonth}
						>
							◀
						</button>
						<span className={styles.currentMonthLabel}>
							{currentYear && currentMonth
								? `${currentYear}年 ${currentMonth}月`
								: "読み込み中..."}
						</span>
						<button
							type="button"
							className={styles.navBtn}
							onClick={handleNextMonth}
						>
							▶
						</button>
					</div>

					{loading ? (
						<div className={styles.loading}>
							<div className={styles.spinner}></div>
							<span>Scanning session logs...</span>
						</div>
					) : error ? (
						<div className={styles.errorContainer}>
							<span className={styles.errorIcon}>⚠️</span>
							<span className={styles.errorMessage}>{error}</span>
							<button type="button" className={styles.retryBtn} onClick={retry}>
								Retry
							</button>
						</div>
					) : (
						<DateTree
							sessions={sessions}
							parsingSessionIds={parsingSessionIds}
							onParseSessions={parseSessions}
						/>
					)}
				</>
			) : (
				<>
					<div className={styles.calendarContainer}>
						<DatePicker value={selectedDate} onChange={handleDateChange} />
					</div>

					{loading ? (
						<div className={styles.loading}>
							<div className={styles.spinner}></div>
							<span>Scanning session logs...</span>
						</div>
					) : error ? (
						<div className={styles.errorContainer}>
							<span className={styles.errorIcon}>⚠️</span>
							<span className={styles.errorMessage}>{error}</span>
							<button type="button" className={styles.retryBtn} onClick={retry}>
								Retry
							</button>
						</div>
					) : (
						<div className={styles.directoryList}>
							{rootFilteredSessions.map((session) => (
								<SessionRow
									key={session.id}
									session={session}
									sessionsMap={sessionsMap}
									isParsing={parsingSessionIds?.has(session.id)}
									parsingSessionIds={parsingSessionIds}
								/>
							))}
							{rootFilteredSessions.length === 0 && (
								<div className={styles.empty}>
									<div className={styles.emptyIcon}>📂</div>
									<div className={styles.emptyText}>
										No sessions found for this date.
									</div>
								</div>
							)}
						</div>
					)}
				</>
			)}

			<UpdateModal
				isOpen={isModalOpen}
				onClose={() => setIsModalOpen(false)}
				latestVersion={updateResult?.latest || ""}
				currentVersion={updateResult?.current || ""}
				status={progress.status}
				progress={progress.progress}
				error={progress.error}
				onUpdate={apply}
			/>
		</div>
	);
}
