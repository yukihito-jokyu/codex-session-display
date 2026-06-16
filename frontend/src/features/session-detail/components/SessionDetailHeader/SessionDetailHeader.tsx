import styles from "./SessionDetailHeader.module.css";

type SessionDetailHeaderProps = {
	sessionId: string;
	parentSessionId?: string | null;
	onBack: () => void;
	onBackToParent?: () => void;
};

export function SessionDetailHeader({
	sessionId,
	parentSessionId,
	onBack,
	onBackToParent,
}: SessionDetailHeaderProps) {
	return (
		<div className={styles.detailHeader}>
			<div className={styles.leftActions}>
				<button type="button" className={styles.backBtn} onClick={onBack}>
					← Back to List
				</button>
				{parentSessionId && (
					<button
						type="button"
						className={styles.parentBtn}
						onClick={onBackToParent}
					>
						← Back to Parent ({parentSessionId.slice(0, 8)})
					</button>
				)}
			</div>
			<span className={styles.detailTitle}>Session Detail: {sessionId}</span>
		</div>
	);
}
