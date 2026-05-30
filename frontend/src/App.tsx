import { useEffect, useState } from "react";
import {
	HashRouter,
	Route,
	Routes,
	useNavigate,
	useParams,
} from "react-router-dom";
import { ListSessions } from "wailsjs/go/main/App";
import styles from "./App.module.css";
import { DateTree } from "./components/ui/DateTree/DateTree";
import type { SessionSummary } from "./components/ui/DateTree/SessionRow";
import { TitleBar } from "./components/ui/TitleBar/TitleBar";
import { Toolbar } from "./components/ui/Toolbar/Toolbar";

function SessionListPage() {
	const [sessions, setSessions] = useState<SessionSummary[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchQuery, setSearchQuery] = useState("");
	const [currentYear, setCurrentYear] = useState<number | null>(null);
	const [currentMonth, setCurrentMonth] = useState<number | null>(null);

	const fetchSessions = (query: string, year: number, month: number) => {
		setLoading(true);
		setError(null);
		ListSessions(query, year, month)
			.then((data) => {
				setSessions(data || []);
				setLoading(false);

				// 初回起動時（year=0, month=0）の場合は、返ってきたデータのタイムスタンプから年月を割り出して同期する
				if (year === 0 && month === 0) {
					if (data && data.length > 0 && data[0].timestamp) {
						const date = new Date(data[0].timestamp);
						if (!Number.isNaN(date.getTime())) {
							setCurrentYear(date.getFullYear());
							setCurrentMonth(date.getMonth() + 1);
							return;
						}
					}
					// セッションが全くない場合は現在の年月をデフォルトにする
					const now = new Date();
					setCurrentYear(now.getFullYear());
					setCurrentMonth(now.getMonth() + 1);
				}
			})
			.catch((err) => {
				console.error(err);
				setError(`Failed to fetch sessions: ${err.message || err}`);
				setLoading(false);
			});
	};

	useEffect(() => {
		if (currentYear === null || currentMonth === null) {
			fetchSessions(searchQuery, 0, 0);
		} else {
			fetchSessions(searchQuery, currentYear, currentMonth);
		}
	}, [searchQuery, currentYear, currentMonth]);

	const handleSearch = (query: string) => {
		setSearchQuery(query);
	};

	const handlePrevMonth = () => {
		if (!currentYear || !currentMonth) return;
		if (currentMonth === 1) {
			setCurrentYear(currentYear - 1);
			setCurrentMonth(12);
		} else {
			setCurrentMonth(currentMonth - 1);
		}
	};

	const handleNextMonth = () => {
		if (!currentYear || !currentMonth) return;
		if (currentMonth === 12) {
			setCurrentYear(currentYear + 1);
			setCurrentMonth(1);
		} else {
			setCurrentMonth(currentMonth + 1);
		}
	};

	const filteredSessions = sessions;

	return (
		<div className={styles.listPage}>
			<Toolbar
				totalCount={filteredSessions.length}
				sourcePath="~/.codex/sessions"
				onSearch={handleSearch}
			/>

			<div className={styles.monthNavigator}>
				<button
					type="button"
					className={styles.navBtn}
					onClick={handlePrevMonth}
					disabled={loading || currentYear === null}
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
					disabled={loading || currentYear === null}
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
					<button
						className={styles.retryBtn}
						onClick={() =>
							fetchSessions(searchQuery, currentYear || 0, currentMonth || 0)
						}
					>
						Retry
					</button>
				</div>
			) : (
				<DateTree sessions={filteredSessions} />
			)}
		</div>
	);
}

function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();

	return (
		<div className={styles.detailPage}>
			<div className={styles.detailHeader}>
				<button className={styles.backBtn} onClick={() => navigate("/")}>
					← Back to List
				</button>
				<span className={styles.detailTitle}>Session Detail</span>
			</div>
			<div className={styles.detailContent}>
				<div className={styles.detailCard}>
					<h3>Session ID</h3>
					<p className={styles.mono}>{id}</p>
					<div className={styles.placeholderBadge}>
						Acceptance test passed for navigation!
					</div>
					<p className={styles.placeholderDesc}>
						The React Flow canvas and statistical charts will be implemented in
						the next phase (C3/C4).
					</p>
				</div>
			</div>
		</div>
	);
}

function App() {
	return (
		<HashRouter>
			<div className={styles.appContainer}>
				<TitleBar />
				<main className={styles.mainContent}>
					<Routes>
						<Route path="/" element={<SessionListPage />} />
						<Route path="/sessions/:id" element={<SessionDetailPage />} />
					</Routes>
				</main>
			</div>
		</HashRouter>
	);
}

export default App;
