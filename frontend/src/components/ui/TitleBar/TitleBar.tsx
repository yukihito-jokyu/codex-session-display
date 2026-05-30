import type React from "react";
import {
	Quit,
	WindowMinimise,
	WindowToggleMaximise,
} from "wailsjs/runtime/runtime";
import styles from "./TitleBar.module.css";

export const TitleBar: React.FC = () => {
	return (
		<header className={styles.titleBar}>
			<div className={styles.dragArea}>
				<span className={styles.title}>Codex Session Display</span>
			</div>
			<div className={styles.controls}>
				<button
					type="button"
					className={styles.controlBtn}
					onClick={WindowMinimise}
					title="Minimize"
					aria-label="Minimize"
				>
					<svg width="10" height="1" viewBox="0 0 10 1">
						<title>Minimize</title>
						<line
							x1="0"
							y1="0.5"
							x2="10"
							y2="0.5"
							stroke="currentColor"
							strokeWidth="1"
						/>
					</svg>
				</button>
				<button
					type="button"
					className={styles.controlBtn}
					onClick={WindowToggleMaximise}
					title="Maximize"
					aria-label="Maximize"
				>
					<svg width="10" height="10" viewBox="0 0 10 10">
						<title>Maximize</title>
						<rect
							x="0.5"
							y="0.5"
							width="9"
							height="9"
							fill="none"
							stroke="currentColor"
							strokeWidth="1"
						/>
					</svg>
				</button>
				<button
					type="button"
					className={`${styles.controlBtn} ${styles.closeBtn}`}
					onClick={Quit}
					title="Close"
					aria-label="Close"
				>
					<svg width="10" height="10" viewBox="0 0 10 10">
						<title>Close</title>
						<path
							d="M 0,0 L 10,10 M 10,0 L 0,10"
							stroke="currentColor"
							strokeWidth="1"
						/>
					</svg>
				</button>
			</div>
		</header>
	);
};
