import { useState } from "react";
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
	const [expandedItems, setExpandedItems] = useState<Set<string>>(
		() => new Set(),
	);

	const toggleItem = (itemKey: string) => {
		setExpandedItems((current) => {
			const next = new Set(current);
			if (next.has(itemKey)) {
				next.delete(itemKey);
			} else {
				next.add(itemKey);
			}
			return next;
		});
	};

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
							{turn.items.map((item, itemIndex) => {
								const kind = item.kind || "conversation";
								const itemKey = `${turn.index}-${itemIndex}-${kind}-${item.timestamp}`;
								const collapsible = item.collapsible ?? kind !== "conversation";
								const expanded = !collapsible || expandedItems.has(itemKey);

								return (
									<article
										className={`${styles.item} ${
											kind === "conversation"
												? item.role === "user"
													? styles.user
													: styles.assistant
												: styles.event
										}`}
										key={itemKey}
									>
										{collapsible ? (
											<button
												aria-expanded={expanded}
												className={styles.eventToggle}
												onClick={() => toggleItem(itemKey)}
												type="button"
											>
												<strong>{item.label || kind}</strong>
												<span>{item.record_count || 1}件の記録</span>
											</button>
										) : (
											<div className={styles.itemHeader}>
												<strong>{item.role === "user" ? "User" : "AI"}</strong>
												{item.timestamp && (
													<time dateTime={item.timestamp}>
														{item.timestamp}
													</time>
												)}
											</div>
										)}
										{expanded && (
											<div className={styles.expandedContent}>
												<p className={styles.body}>{item.body}</p>
												{item.details?.length > 0 && (
													<pre className={styles.detail}>
														{item.details
															.map(
																(detail) => `${detail.label}\n${detail.value}`,
															)
															.join("\n\n")}
													</pre>
												)}
											</div>
										)}
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
								);
							})}
						</div>
					</section>
				))}
			</div>
		</aside>
	);
}
