import { useState } from "react";
import { DateTree } from "../../components/ui/DateTree/DateTree";
import { Toolbar } from "../../components/ui/Toolbar/Toolbar";
import { UpdateModal } from "../../components/ui/UpdateModal/UpdateModal";
import { useSessions } from "./hooks/useSessions";
import { useUpdate } from "./hooks/useUpdate";
import styles from "./SessionListPage.module.css";

export function SessionListPage() {
	const {
		sessions,
		loading,
		error,
		currentYear,
		currentMonth,
		handleSearch,
		handlePrevMonth,
		handleNextMonth,
		retry,
		parsingSessionIds,
		parseSessions,
	} = useSessions();

	const { updateResult, progress, apply } = useUpdate();
	const [isModalOpen, setIsModalOpen] = useState(false);

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
