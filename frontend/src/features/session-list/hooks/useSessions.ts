import { useCallback, useEffect, useRef, useState } from "react";
import { GetSessionDetail, ListSessions } from "wailsjs/go/main/App";
import type { SessionSummary } from "../../../components/ui/DateTree/SessionRow";

export function useSessions() {
	const [sessions, setSessions] = useState<SessionSummary[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const [searchQuery, setSearchQuery] = useState(() => {
		return sessionStorage.getItem("session_list_query") || "";
	});
	const [currentYear, setCurrentYear] = useState<number | null>(() => {
		const y = sessionStorage.getItem("session_list_year");
		return y ? parseInt(y, 10) : null;
	});
	const [currentMonth, setCurrentMonth] = useState<number | null>(() => {
		const m = sessionStorage.getItem("session_list_month");
		return m ? parseInt(m, 10) : null;
	});

	const prevYearMonthRef = useRef<{
		year: number | null;
		month: number | null;
	}>({
		year: currentYear,
		month: currentMonth,
	});

	// 直近で実際にフェッチされた（またはフェッチによって解決された）パラメータを保持して重複リクエストを防止する
	const lastFetchedRef = useRef<{
		query: string;
		year: number;
		month: number;
	} | null>(null);

	// 解析中ステータス管理用の状態と参照
	const [parsingSessionIds, setParsingSessionIds] = useState<Set<string>>(
		new Set(),
	);
	const parsingSessionIdsRef = useRef<Set<string>>(new Set());
	const failedSessionIdsRef = useRef<Set<string>>(new Set());
	const queueRef = useRef<string[]>([]);
	const activeCountRef = useRef<number>(0);

	const currentYearRef = useRef<number | null>(currentYear);
	const currentMonthRef = useRef<number | null>(currentMonth);
	currentYearRef.current = currentYear;
	currentMonthRef.current = currentMonth;

	const fetchSessions = useCallback(
		(query: string, year: number, month: number, isSilent = false) => {
			if (!isSilent) {
				setLoading(true);
				// 非サイレント更新（通常のフェッチ）の場合はキューと解析状態をクリアする
				queueRef.current = [];
				activeCountRef.current = 0;
				parsingSessionIdsRef.current = new Set();
				setParsingSessionIds(new Set());
				failedSessionIdsRef.current = new Set();
			}
			setError(null);
			lastFetchedRef.current = { query, year, month };

			ListSessions(query, year, month)
				.then((data) => {
					setSessions(data || []);
					if (!isSilent) {
						setLoading(false);
					}

					// 初回起動時（year=0, month=0）の場合は、返ってきたデータのタイムスタンプから年月を割り出して同期する
					if (year === 0 && month === 0) {
						let resolvedYear = 0;
						let resolvedMonth = 0;
						if (data && data.length > 0 && data[0].timestamp) {
							const date = new Date(data[0].timestamp);
							if (!Number.isNaN(date.getTime())) {
								resolvedYear = date.getFullYear();
								resolvedMonth = date.getMonth() + 1;
							}
						}
						if (resolvedYear === 0) {
							const now = new Date();
							resolvedYear = now.getFullYear();
							resolvedMonth = now.getMonth() + 1;
						}

						// 解決された年月で lastFetchedRef.current を更新し、直後の useEffect トリガーによる
						// 重複フェッチ（同一クエリ、同一の解決後年月）をスキップできるようにする
						lastFetchedRef.current = {
							query,
							year: resolvedYear,
							month: resolvedMonth,
						};

						setCurrentYear(resolvedYear);
						setCurrentMonth(resolvedMonth);
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
		const targetYear = currentYear === null ? 0 : currentYear;
		const targetMonth = currentMonth === null ? 0 : currentMonth;

		// すでに同じパラメータでフェッチが完了または実行中の場合はスキップ
		if (
			lastFetchedRef.current &&
			lastFetchedRef.current.query === searchQuery &&
			lastFetchedRef.current.year === targetYear &&
			lastFetchedRef.current.month === targetMonth
		) {
			return;
		}

		fetchSessions(searchQuery, targetYear, targetMonth);
	}, [searchQuery, currentYear, currentMonth, fetchSessions]);

	// 状態が変更されたら sessionStorage を更新する
	useEffect(() => {
		if (searchQuery) {
			sessionStorage.setItem("session_list_query", searchQuery);
		} else {
			sessionStorage.removeItem("session_list_query");
		}
	}, [searchQuery]);

	useEffect(() => {
		if (currentYear !== null) {
			sessionStorage.setItem("session_list_year", currentYear.toString());
		} else {
			sessionStorage.removeItem("session_list_year");
		}
		if (currentMonth !== null) {
			sessionStorage.setItem("session_list_month", currentMonth.toString());
		} else {
			sessionStorage.removeItem("session_list_month");
		}

		// 有効な年月から別の有効な年月へ明示的に「変更」されたときのみ、アコーディオンの展開キャッシュをクリアする
		if (
			prevYearMonthRef.current.year !== null &&
			prevYearMonthRef.current.month !== null &&
			(prevYearMonthRef.current.year !== currentYear ||
				prevYearMonthRef.current.month !== currentMonth)
		) {
			sessionStorage.removeItem("session_list_expanded_paths");
		}
		prevYearMonthRef.current = { year: currentYear, month: currentMonth };
	}, [currentYear, currentMonth]);

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

	const updateParsingIds = useCallback(
		(updater: (prev: Set<string>) => Set<string>) => {
			setParsingSessionIds((prev) => {
				const next = updater(new Set(prev));
				parsingSessionIdsRef.current = next;
				return next;
			});
		},
		[],
	);

	const processQueue = useCallback(() => {
		const maxConcurrency = 3;
		while (
			activeCountRef.current < maxConcurrency &&
			queueRef.current.length > 0
		) {
			const nextId = queueRef.current.shift();
			if (!nextId) break;

			activeCountRef.current++;

			GetSessionDetail(nextId)
				.then((detail) => {
					setSessions((prevSessions) => {
						return prevSessions.map((s) => {
							if (s.id !== nextId) return s;

							// detail.nodes から sessionMeta ノードを探してメタデータを抽出
							const metaNode = detail.nodes?.find(
								(n) => n.type === "sessionMeta",
							);
							const meta = metaNode?.data?.meta || {};
							return {
								...s,
								cwd: meta.cwd || null,
								cli_version: meta.cli_version || null,
								originator: meta.originator || null,
								model_provider: meta.model_provider || null,
								branch: meta.git_branch || null,
								source: meta.source || null,
								timestamp: meta.timestamp || s.timestamp,
								parsed: true,
								parent_session_id: detail.parent_session_id || null,
								child_session_ids: detail.child_session_ids || null,
							};
						});
					});
				})
				.catch((err) => {
					console.error(`Failed to parse session ${nextId}:`, err);
					failedSessionIdsRef.current.add(nextId);
				})
				.finally(() => {
					activeCountRef.current--;
					updateParsingIds((prev) => {
						prev.delete(nextId);
						return prev;
					});

					// キュー内の次の処理を開始
					processQueue();

					// キューと実行中タスクの両方が空になったらサイレントリフレッシュ
					if (activeCountRef.current === 0 && queueRef.current.length === 0) {
						fetchSessions(
							searchQuery,
							currentYearRef.current || 0,
							currentMonthRef.current || 0,
							true,
						);
					}
				});
		}
	}, [searchQuery, fetchSessions, updateParsingIds]);

	const parseSessions = useCallback(
		(ids: string[]) => {
			// 未解析かつ解析中/キュー/失敗履歴にないIDのみ抽出
			const newIds = ids.filter((id) => {
				const session = sessions.find((s) => s.id === id);
				const isParsed = session?.parsed;
				const isParsing = parsingSessionIdsRef.current.has(id);
				const isInQueue = queueRef.current.includes(id);
				const isFailed = failedSessionIdsRef.current.has(id);
				return !isParsed && !isParsing && !isInQueue && !isFailed;
			});

			if (newIds.length === 0) return;

			queueRef.current.push(...newIds);
			updateParsingIds((prev) => {
				for (const id of newIds) {
					prev.add(id);
				}
				return prev;
			});

			processQueue();
		},
		[sessions, updateParsingIds, processQueue],
	);

	const retry = useCallback(() => {
		// リトライ時はキャッシュを無視して強制的に最新化するため、lastFetchedRef をクリアしてフェッチする
		lastFetchedRef.current = null;
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
		parsingSessionIds,
		parseSessions,
	};
}
