import type { dto } from "wailsjs/go/models";
import styles from "./ConversationTimeline.module.css";

type ConversationTimelineProps = {
	turns: dto.ConversationTimelineTurn[];
};

const compactNumber = new Intl.NumberFormat("en", {
	notation: "compact",
	maximumFractionDigits: 1,
});

function formatTokens(value: number) {
	return `${compactNumber.format(value)} tokens`;
}

function formatDuration(durationMs: number) {
	if (durationMs < 1000) {
		return `${durationMs}ms`;
	}
	const seconds = durationMs / 1000;
	return `${Number.isInteger(seconds) ? seconds : seconds.toFixed(1)}秒`;
}

export function ConversationTimeline({ turns }: ConversationTimelineProps) {
	return (
		<aside
			aria-label="会話タイムライン"
			className={styles.timeline}
			data-testid="conversation-timeline"
		>
			<h2 className={styles.heading}>会話タイムライン</h2>
			<div className={styles.turnList}>
				{turns.map((turn) => (
					<section
						className={styles.turn}
						key={`${turn.index}-${turn.turn_id}-${turn.items[0]?.timestamp}-${turn.items[0]?.body}`}
					>
						<header className={styles.turnHeader}>
							<strong>
								{turn.pseudo ? "ターン外イベント" : `ターン ${turn.index + 1}`}
							</strong>
							{!turn.pseudo && (
								<div className={styles.turnMetrics}>
									<span>{formatDuration(turn.duration_ms)}</span>
									<span>
										{formatTokens(turn.consumed_tokens?.total_tokens || 0)}
									</span>
								</div>
							)}
						</header>

						<div className={styles.itemList}>
							{turn.items.map((item) => (
								<article
									className={`${styles.item} ${
										item.role === "user" ? styles.user : styles.assistant
									}`}
									key={`${item.role}-${item.timestamp}-${item.body}`}
								>
									<div className={styles.itemHeader}>
										<strong>{item.role === "user" ? "User" : "AI"}</strong>
										{item.timestamp && (
											<time dateTime={item.timestamp}>{item.timestamp}</time>
										)}
									</div>
									<p className={styles.body}>{item.body}</p>
									<div className={styles.tokenMetrics}>
										{item.token_count_count > 0 ? (
											<>
												<strong>
													{formatTokens(
														item.last_token_usage?.total_tokens || 0,
													)}
												</strong>
												<span>{item.token_count_count}件</span>
												{item.total_token_usage && (
													<span>
														累計{" "}
														{compactNumber.format(
															item.total_token_usage.total_tokens,
														)}
													</span>
												)}
											</>
										) : (
											<span>計測なし</span>
										)}
									</div>
								</article>
							))}
						</div>
					</section>
				))}
			</div>
		</aside>
	);
}
