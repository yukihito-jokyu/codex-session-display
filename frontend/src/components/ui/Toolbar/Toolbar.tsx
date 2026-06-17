import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import styles from "./Toolbar.module.css";

interface ToolbarProps {
	totalCount: number;
	sourcePath: string;
	onSearch: (query: string) => void;
	hasUpdate?: boolean;
	latestVersion?: string;
	onUpdateClick?: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
	totalCount,
	sourcePath,
	onSearch,
	hasUpdate,
	latestVersion,
	onUpdateClick,
}) => {
	const [inputValue, setInputValue] = useState("");
	const timeoutRef = useRef<number | null>(null);

	const handleChange = useCallback(
		(e: React.ChangeEvent<HTMLInputElement>) => {
			const value = e.target.value;
			setInputValue(value);

			if (timeoutRef.current) {
				window.clearTimeout(timeoutRef.current);
			}

			timeoutRef.current = window.setTimeout(() => {
				onSearch(value);
			}, 200); // 200ms デバウンス
		},
		[onSearch],
	);

	useEffect(() => {
		return () => {
			if (timeoutRef.current) {
				window.clearTimeout(timeoutRef.current);
			}
		};
	}, []);

	return (
		<div className={styles.toolbar}>
			<div className={styles.info}>
				<div className={styles.countBadge}>
					<span className={styles.badgeLabel}>SESSIONS</span>
					<span className={styles.badgeValue}>{totalCount}</span>
				</div>
				<div className={styles.path}>
					<span className={styles.pathLabel}>Source:</span>
					<span className={styles.pathValue} title={sourcePath}>
						{sourcePath}
					</span>
				</div>
				{hasUpdate && latestVersion && (
					<button
						type="button"
						className={styles.updateBadgeBtn}
						onClick={onUpdateClick}
					>
						🚀 新バージョンがあります (v{latestVersion})
					</button>
				)}
			</div>
			<div className={styles.searchContainer}>
				<span className={styles.searchIcon} aria-hidden="true">
					🔍
				</span>
				<input
					type="text"
					placeholder="Filter by ID, CWD, branch, provider..."
					value={inputValue}
					onChange={handleChange}
					className={styles.searchInput}
					aria-label="Filter sessions"
				/>
			</div>
		</div>
	);
};
