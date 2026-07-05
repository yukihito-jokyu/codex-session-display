import type React from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import styles from "./DatePicker.module.css";

interface DatePickerProps {
	value: string; // YYYY-MM-DD
	onChange: (value: string) => void;
	sessionCounts?: { [dateStr: string]: number };
	onMonthChange?: (year: number, month: number) => void;
}

export const DatePicker: React.FC<DatePickerProps> = ({
	value,
	onChange,
	sessionCounts,
	onMonthChange,
}) => {
	const [isOpen, setIsOpen] = useState(false);
	const containerRef = useRef<HTMLDivElement>(null);
	const dropdownRef = useRef<HTMLDivElement>(null);

	const [hoveredDay, setHoveredDay] = useState<{
		date: Date;
		rect: { top: number; left: number; width: number; height: number };
	} | null>(null);

	// カレンダー表示用の年月
	const [viewDate, setViewDate] = useState(() => {
		const d = new Date(value);
		return Number.isNaN(d.getTime()) ? new Date() : d;
	});

	// value が外部から変わったら表示年月も同期する
	useEffect(() => {
		const d = new Date(value);
		if (!Number.isNaN(d.getTime())) {
			setViewDate(d);
		}
	}, [value]);

	// 外側クリックで閉じる
	useEffect(() => {
		const handleClickOutside = (event: MouseEvent) => {
			if (
				containerRef.current &&
				!containerRef.current.contains(event.target as Node)
			) {
				setIsOpen(false);
			}
		};
		document.addEventListener("mousedown", handleClickOutside);
		return () => {
			document.removeEventListener("mousedown", handleClickOutside);
		};
	}, []);

	// ドロップダウンが閉じられたらホバー状態をクリア
	useEffect(() => {
		if (!isOpen) {
			setHoveredDay(null);
		}
	}, [isOpen]);

	// 年・月の取得
	const year = viewDate.getFullYear();
	const month = viewDate.getMonth(); // 0-indexed

	// 前の月へ
	const prevMonth = () => {
		const nextDate = new Date(year, month - 1, 1);
		setViewDate(nextDate);
		onMonthChange?.(nextDate.getFullYear(), nextDate.getMonth() + 1);
	};

	// 次の月へ
	const nextMonth = () => {
		const nextDate = new Date(year, month + 1, 1);
		setViewDate(nextDate);
		onMonthChange?.(nextDate.getFullYear(), nextDate.getMonth() + 1);
	};

	// カレンダーの日付グリッド作成
	const days = useMemo(() => {
		// 当月1日の曜日 (0: 日, 1: 月, ...)
		const firstDayIndex = new Date(year, month, 1).getDay();
		// 当月の末日
		const lastDay = new Date(year, month + 1, 0).getDate();
		// 前月の末日
		const prevLastDay = new Date(year, month, 0).getDate();

		const grid: { date: Date; isCurrentMonth: boolean }[] = [];

		// 前月の日付で埋める
		for (let i = firstDayIndex - 1; i >= 0; i--) {
			grid.push({
				date: new Date(year, month - 1, prevLastDay - i),
				isCurrentMonth: false,
			});
		}

		// 当月の日付
		for (let i = 1; i <= lastDay; i++) {
			grid.push({
				date: new Date(year, month, i),
				isCurrentMonth: true,
			});
		}

		// 翌月の日付で残りを埋める (合計42マス = 6週間分)
		const remaining = 42 - grid.length;
		for (let i = 1; i <= remaining; i++) {
			grid.push({
				date: new Date(year, month + 1, i),
				isCurrentMonth: false,
			});
		}

		return grid;
	}, [year, month]);

	// 表示中の月における1日の最大セッション数を算出
	const maxSessions = useMemo(() => {
		let max = 0;
		if (!sessionCounts) return 0;
		const lastDay = new Date(year, month + 1, 0).getDate();
		for (let i = 1; i <= lastDay; i++) {
			const date = new Date(year, month, i);
			const y = date.getFullYear();
			const m = String(date.getMonth() + 1).padStart(2, "0");
			const d = String(date.getDate()).padStart(2, "0");
			const dateStr = `${y}-${m}-${d}`;
			const count = sessionCounts[dateStr] || 0;
			if (count > max) {
				max = count;
			}
		}
		return max;
	}, [sessionCounts, year, month]);

	// セッション数に応じた4段階の相対レベル（1〜4）を判定
	const getLevel = (count: number): number => {
		if (count <= 0) return 0;
		if (maxSessions < 4) {
			return Math.min(count, 4);
		}
		const step = maxSessions / 4;
		if (count <= step) return 1;
		if (count <= 2 * step) return 2;
		if (count <= 3 * step) return 3;
		return 4;
	};

	// 選択された日付のフォーマット (表示用)
	const formattedDisplay = useMemo(() => {
		const d = new Date(value);
		if (Number.isNaN(d.getTime())) return "日付を選択";
		const y = d.getFullYear();
		const m = d.getMonth() + 1;
		const dateVal = d.getDate();
		const dayNames = ["日", "月", "火", "水", "木", "金", "土"];
		const dayName = dayNames[d.getDay()];
		return `${y}年 ${m}月 ${dateVal}日 (${dayName})`;
	}, [value]);

	// 日付選択時
	const handleSelectDate = (date: Date) => {
		const y = date.getFullYear();
		const m = String(date.getMonth() + 1).padStart(2, "0");
		const d = String(date.getDate()).padStart(2, "0");
		onChange(`${y}-${m}-${d}`);
		setIsOpen(false);
	};

	const isSelected = (date: Date) => {
		const target = new Date(value);
		return (
			target.getFullYear() === date.getFullYear() &&
			target.getMonth() === date.getMonth() &&
			target.getDate() === date.getDate()
		);
	};

	const isToday = (date: Date) => {
		const today = new Date();
		return (
			today.getFullYear() === date.getFullYear() &&
			today.getMonth() === date.getMonth() &&
			today.getDate() === date.getDate()
		);
	};

	return (
		<div className={styles.datePickerContainer} ref={containerRef}>
			{/* E2Eテスト用隠し要素 */}
			<input
				type="date"
				className={styles.hiddenInput}
				value={value}
				onChange={(e) => onChange(e.target.value)}
			/>

			{/* 表示トリガー */}
			<button
				type="button"
				className={styles.triggerButton}
				onClick={() => setIsOpen(!isOpen)}
			>
				<span className={styles.calendarIcon}>📅</span>
				<span className={styles.dateText}>{formattedDisplay}</span>
				<span className={`${styles.arrow} ${isOpen ? styles.arrowOpen : ""}`}>
					▾
				</span>
			</button>

			{/* ポップアップカレンダー */}
			{isOpen && (
				<div className={styles.dropdown} ref={dropdownRef}>
					<div className={styles.header}>
						<button type="button" className={styles.navBtn} onClick={prevMonth}>
							◀
						</button>
						<span className={styles.currentMonth}>
							{year}年 {month + 1}月
						</span>
						<button type="button" className={styles.navBtn} onClick={nextMonth}>
							▶
						</button>
					</div>

					<div className={styles.weekdays}>
						<span>日</span>
						<span>月</span>
						<span>火</span>
						<span>水</span>
						<span>木</span>
						<span>金</span>
						<span>土</span>
					</div>

					<div className={styles.daysGrid}>
						{days.map(({ date, isCurrentMonth }) => {
							const selected = isSelected(date);
							const today = isToday(date);

							const y = date.getFullYear();
							const m = String(date.getMonth() + 1).padStart(2, "0");
							const d = String(date.getDate()).padStart(2, "0");
							const dateStr = `${y}-${m}-${d}`;
							const count = sessionCounts ? sessionCounts[dateStr] || 0 : 0;
							const level = isCurrentMonth ? getLevel(count) : 0;
							const levelClass = level > 0 ? styles[`graphL${level}`] : "";

							return (
								<button
									key={date.toISOString()}
									type="button"
									className={`${styles.dayCell} ${
										!isCurrentMonth ? styles.otherMonth : ""
									} ${selected ? styles.selected : ""} ${
										today ? styles.today : ""
									} ${levelClass}`}
									onClick={() => handleSelectDate(date)}
									onMouseEnter={(e) => {
										const button = e.currentTarget;
										const dropdown = dropdownRef.current;
										if (!button || !dropdown) return;
										const buttonRect = button.getBoundingClientRect();
										const dropdownRect = dropdown.getBoundingClientRect();
										setHoveredDay({
											date,
											rect: {
												top: buttonRect.top - dropdownRect.top,
												left: buttonRect.left - dropdownRect.left,
												width: buttonRect.width,
												height: buttonRect.height,
											},
										});
									}}
									onMouseLeave={() => setHoveredDay(null)}
								>
									{date.getDate()}
								</button>
							);
						})}
					</div>

					{hoveredDay && (
						<div
							className={styles.tooltip}
							style={{
								top: `${hoveredDay.rect.top - 8}px`,
								left: `${Math.max(
									80,
									Math.min(
										200,
										hoveredDay.rect.left + hoveredDay.rect.width / 2,
									),
								)}px`,
							}}
						>
							{(() => {
								const y = hoveredDay.date.getFullYear();
								const m = String(hoveredDay.date.getMonth() + 1).padStart(
									2,
									"0",
								);
								const d = String(hoveredDay.date.getDate()).padStart(2, "0");
								const dateKey = `${y}-${m}-${d}`;
								const count = sessionCounts?.[dateKey] || 0;
								return `${count} session${count === 1 ? "" : "s"} on ${dateKey}`;
							})()}
						</div>
					)}
				</div>
			)}
		</div>
	);
};
