import type React from "react";
import styles from "./UpdateModal.module.css";

interface UpdateModalProps {
	isOpen: boolean;
	onClose: () => void;
	latestVersion: string;
	currentVersion: string;
	status: "idle" | "downloading" | "extracting" | "restarting" | "error";
	progress: number;
	error?: string;
	onUpdate: () => void;
}

export const UpdateModal: React.FC<UpdateModalProps> = ({
	isOpen,
	onClose,
	latestVersion,
	currentVersion,
	status,
	progress,
	error,
	onUpdate,
}) => {
	if (!isOpen) return null;

	const isProcessing =
		status === "downloading" ||
		status === "extracting" ||
		status === "restarting";

	// ステータスに応じた進捗表示のメッセージ
	const getStatusMessage = () => {
		switch (status) {
			case "downloading":
				return `最新バージョンをダウンロードしています... ${Math.round(progress)}%`;
			case "extracting":
				return "パッケージを展開しています...";
			case "restarting":
				return "システムを再起動しています...";
			case "error":
				return "アップデートの適用中にエラーが発生しました。";
			default:
				return "";
		}
	};

	return (
		<div
			className={styles.overlay}
			role="dialog"
			aria-modal="true"
			aria-labelledby="modal-title"
		>
			<div className={styles.modalCard}>
				<h2 id="modal-title" className={styles.title}>
					{isProcessing
						? "アップデートを適用中"
						: status === "error"
							? "エラーが発生しました"
							: "新しいバージョンがあります"}
				</h2>

				{!isProcessing && status !== "error" && (
					<div className={styles.confirmContent}>
						<p className={styles.description}>
							codex-session-display
							の新しいバージョンが利用可能です。今すぐアップデートを実行しますか？
						</p>
						<div className={styles.versionCompare}>
							<div className={styles.versionBadge}>
								<span className={styles.badgeLabel}>現在</span>
								<span className={styles.badgeValue}>v{currentVersion}</span>
							</div>
							<span className={styles.arrow}>➔</span>
							<div className={`${styles.versionBadge} ${styles.latest}`}>
								<span className={styles.badgeLabel}>最新</span>
								<span className={styles.badgeValue}>v{latestVersion}</span>
							</div>
						</div>
						<div className={styles.actions}>
							<button
								type="button"
								className={styles.cancelBtn}
								onClick={onClose}
							>
								後で
							</button>
							<button
								type="button"
								className={styles.updateBtn}
								onClick={onUpdate}
							>
								アップデート
							</button>
						</div>
					</div>
				)}

				{(isProcessing || status === "error") && (
					<div className={styles.progressContent}>
						{status === "error" ? (
							<div className={styles.errorContainer}>
								<div className={styles.errorIcon} aria-hidden="true">
									⚠️
								</div>
								<p className={styles.errorText}>
									{error || "不明なエラーが発生しました。"}
								</p>
								<button
									type="button"
									className={styles.closeBtn}
									onClick={onClose}
								>
									閉じる
								</button>
							</div>
						) : (
							<div className={styles.loadingContainer}>
								<p className={styles.statusLabel}>{getStatusMessage()}</p>
								<div className={styles.progressBarBg} aria-hidden="true">
									<div
										className={styles.progressBarFill}
										style={{
											width: `${status === "downloading" ? progress : 100}%`,
										}}
									/>
								</div>
								<div className={styles.spinner} aria-hidden="true" />
								<p className={styles.warningText}>
									※
									アップデート処理には数秒〜十数秒かかります。完了後、アプリは自動で再起動します。このウインドウを閉じないでください。
								</p>
							</div>
						)}
					</div>
				)}
			</div>
		</div>
	);
};
