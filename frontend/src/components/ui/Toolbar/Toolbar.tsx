import type React from "react";
import { useEffect, useRef, useState } from "react";
import styles from "./Toolbar.module.css";

interface ToolbarProps {
	totalCount: number;
	sourcePath: string;
	onSearch: (query: string) => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
	totalCount,
	sourcePath,
	onSearch,
}) => {
	const [inputValue, setInputValue] = useState("");
	const timeoutRef = useRef<number | null>(null);

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const value = e.target.value;
		setInputValue(value);

		if (timeoutRef.current) {
			window.clearTimeout(timeoutRef.current);
		}

		timeoutRef.current = window.setTimeout(() => {
			onSearch(value);
		}, 200); // 200ms デバウンス
	};

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
			</div>
			<div className={styles.searchContainer}>
				<span className={styles.searchIcon}>🔍</span>
				<input
					type="text"
					placeholder="Filter by ID, CWD, branch, provider..."
					value={inputValue}
					onChange={handleChange}
					className={styles.searchInput}
				/>
			</div>
		</div>
	);
};
