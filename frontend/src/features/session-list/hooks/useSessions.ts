import { useCallback, useEffect, useState } from "react";
import { ListSessions } from "wailsjs/go/main/App";
import type { SessionSummary } from "../../../components/ui/DateTree/SessionRow";

export function useSessions() {
	const [sessions, setSessions] = useState<SessionSummary[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchQuery, setSearchQuery] = useState("");
	const [currentYear, setCurrentYear] = useState<number | null>(null);
	const [currentMonth, setCurrentMonth] = useState<number | null>(null);

	const fetchSessions = useCallback(
		(query: string, year: number, month: number) => {
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
		},
		[],
	);

	useEffect(() => {
		if (currentYear === null || currentMonth === null) {
			fetchSessions(searchQuery, 0, 0);
		} else {
			fetchSessions(searchQuery, currentYear, currentMonth);
		}
	}, [searchQuery, currentYear, currentMonth, fetchSessions]);

	const handleSearch = useCallback((query: string) => {
		setSearchQuery(query);
	}, []);

	const handlePrevMonth = useCallback(() => {
		if (!currentYear || !currentMonth) return;
		if (currentMonth === 1) {
			setCurrentYear(currentYear - 1);
			setCurrentMonth(12);
		} else {
			setCurrentMonth(currentMonth - 1);
		}
	}, [currentYear, currentMonth]);

	const handleNextMonth = useCallback(() => {
		if (!currentYear || !currentMonth) return;
		if (currentMonth === 12) {
			setCurrentYear(currentYear + 1);
			setCurrentMonth(1);
		} else {
			setCurrentMonth(currentMonth + 1);
		}
	}, [currentYear, currentMonth]);

	const retry = useCallback(() => {
		fetchSessions(searchQuery, currentYear || 0, currentMonth || 0);
	}, [fetchSessions, searchQuery, currentYear, currentMonth]);

	return {
		sessions,
		loading,
		error,
		searchQuery,
		currentYear,
		currentMonth,
		handleSearch,
		handlePrevMonth,
		handleNextMonth,
		retry,
	};
}
