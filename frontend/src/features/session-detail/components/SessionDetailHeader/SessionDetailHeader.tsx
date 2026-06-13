import styles from "./SessionDetailHeader.module.css";

type SessionDetailHeaderProps = {
	sessionId: string;
	onBack: () => void;
};

export function SessionDetailHeader({
	sessionId,
	onBack,
}: SessionDetailHeaderProps) {
	return (
		<div className={styles.detailHeader}>
			<button type="button" className={styles.backBtn} onClick={onBack}>
				← Back to List
			</button>
			<span className={styles.detailTitle}>Session Detail: {sessionId}</span>
		</div>
	);
}
