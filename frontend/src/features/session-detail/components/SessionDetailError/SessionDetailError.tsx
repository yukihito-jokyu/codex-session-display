import styles from "./SessionDetailError.module.css";

type SessionDetailErrorProps = {
	error: string;
	logActionMessage: string | null;
	onRetry: () => void;
	onBack: () => void;
	onOpenLogDirectory: () => void;
	onCopyLogPath: () => void;
};

export function SessionDetailError({
	error,
	logActionMessage,
	onRetry,
	onBack,
	onOpenLogDirectory,
	onCopyLogPath,
}: SessionDetailErrorProps) {
	return (
		<div className={styles.errorContainer}>
			<span className={styles.errorIcon}>⚠️</span>
			<span className={styles.errorMessage}>{error}</span>
			<div className={styles.errorActions}>
				<button type="button" className={styles.retryBtn} onClick={onRetry}>
					Retry
				</button>
				<button type="button" className={styles.backBtn} onClick={onBack}>
					Back to List
				</button>
			</div>
			<div className={styles.errorActions}>
				<button
					type="button"
					className={styles.secondaryBtn}
					onClick={onOpenLogDirectory}
				>
					ログフォルダを開く
				</button>
				<button
					type="button"
					className={styles.secondaryBtn}
					onClick={onCopyLogPath}
				>
					ログパスをコピー
				</button>
			</div>
			{logActionMessage && (
				<span className={styles.logActionMessage}>{logActionMessage}</span>
			)}
		</div>
	);
}
