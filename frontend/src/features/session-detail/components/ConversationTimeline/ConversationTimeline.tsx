import { type MouseEvent, useEffect, useMemo, useRef, useState } from "react";
import type { dto } from "wailsjs/go/models";
import styles from "./ConversationTimeline.module.css";
import {
	getTimelineItemPreview,
	getTimelineItemTextLength,
	TIMELINE_PREVIEW_LENGTH,
} from "./timelineItemText";

type ConversationTimelineProps = {
	turns: dto.ConversationTimelineTurn[];
	selectedSelectionId: string | null;
	scrollTarget: { selectionId: string; timestamp: number } | null;
	onSelect: (item: dto.ConversationTimelineItem) => void;
	onShowFullText: (item: dto.ConversationTimelineItem) => void;
	onSubagentClick?: (threadId: string, provider?: string) => void;
};

const compactNumber = new Intl.NumberFormat("en", {
	notation: "compact",
	maximumFractionDigits: 1,
});

function matchesSearchQuery(
	item: dto.ConversationTimelineItem,
	normalizedQuery: string,
) {
	if (!normalizedQuery) {
		return true;
	}

	const searchableText = [
		item.body,
		...(item.details ?? []).map((detail) => detail.value),
	]
		.join("\n")
		.toLocaleLowerCase();
	return searchableText.includes(normalizedQuery);
}

function matchesTimelineFilters(
	item: dto.ConversationTimelineItem,
	normalizedQuery: string,
) {
	const allowedKinds = ["conversation", "collab", "tool"];
	const isAllowed = allowedKinds.includes(item.kind || "");
	return isAllowed && matchesSearchQuery(item, normalizedQuery);
}

function getTimelineItemKey(
	turn: Pick<dto.ConversationTimelineTurn, "index" | "turn_id">,
	item: dto.ConversationTimelineItem,
) {
	return [
		turn.index,
		turn.turn_id,
		item.kind,
		item.timestamp,
		item.role,
		item.label,
		item.body,
	].join("\u0000");
}

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

export function ConversationTimeline({
	turns,
	selectedSelectionId,
	scrollTarget,
	onSelect,
	onShowFullText,
	onSubagentClick,
}: ConversationTimelineProps) {
	const [searchQuery, setSearchQuery] = useState("");
	const [expandedItems, setExpandedItems] = useState<Set<string>>(
		() => new Set(),
	);
	const itemRefs = useRef(new Map<string, HTMLElement>());
	const filteredTurns = useMemo(() => {
		const normalizedQuery = searchQuery.trim().toLocaleLowerCase();

		return turns.flatMap((turn) => {
			const items = turn.items.filter((item) =>
				matchesTimelineFilters(item, normalizedQuery),
			);

			return items.length > 0 ? [{ ...turn, items }] : [];
		});
	}, [searchQuery, turns]);

	useEffect(() => {
		if (!scrollTarget) {
			return;
		}
		itemRefs.current
			.get(scrollTarget.selectionId)
			?.scrollIntoView({ behavior: "smooth", block: "center" });
	}, [scrollTarget]);

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

	const stopPropagation = (event: MouseEvent<HTMLElement>) => {
		event.stopPropagation();
	};

	return (
		<aside
			aria-label="会話タイムライン"
			className={styles.timeline}
			data-testid="conversation-timeline"
		>
			<h2 className={styles.heading}>会話タイムライン</h2>
			<div className={styles.filters}>
				<label className={styles.searchField}>
					<span>検索</span>
					<input
						aria-label="タイムラインを検索"
						onChange={(event) => setSearchQuery(event.target.value)}
						type="search"
						value={searchQuery}
					/>
				</label>
			</div>
			<div className={styles.turnList}>
				{filteredTurns.length === 0 && (
					<p className={styles.emptyState}>
						条件に一致するタイムライン項目はありません
					</p>
				)}
				{filteredTurns.map((turn) => (
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

						<div
							aria-label={
								turn.pseudo
									? "ターン外イベントの表示単位"
									: `ターン ${turn.index + 1} の表示単位`
							}
							className={styles.itemList}
							role="listbox"
						>
							{turn.items.map((item) => {
								const kind = item.kind || "conversation";
								const itemKey = getTimelineItemKey(turn, item);
								const collapsible = item.collapsible ?? kind !== "conversation";
								const expanded = !collapsible || expandedItems.has(itemKey);
								const truncated =
									getTimelineItemTextLength(item) > TIMELINE_PREVIEW_LENGTH;
								const preview =
									expanded && truncated ? getTimelineItemPreview(item) : "";

								return (
									<div
										className={`${styles.item} ${
											kind === "conversation"
												? item.role === "user"
													? styles.user
													: styles.assistant
												: styles.event
										} ${
											item.selection_id === selectedSelectionId
												? styles.selected
												: ""
										}`}
										aria-selected={item.selection_id === selectedSelectionId}
										data-testid={
											item.selection_id
												? `timeline-item-${item.selection_id}`
												: undefined
										}
										key={itemKey}
										onClick={() => onSelect(item)}
										onKeyDown={(event) => {
											if (event.target !== event.currentTarget) {
												return;
											}
											if (event.key === "Enter" || event.key === " ") {
												event.preventDefault();
												onSelect(item);
											}
										}}
										ref={(element) => {
											if (!item.selection_id) {
												return;
											}
											if (element) {
												itemRefs.current.set(item.selection_id, element);
											} else {
												itemRefs.current.delete(item.selection_id);
											}
										}}
										role="option"
										tabIndex={0}
									>
										<article>
											<div className={styles.itemHeader}>
												<div className={styles.headerLeft}>
													{collapsible ? (
														<button
															aria-expanded={expanded}
															className={styles.eventToggle}
															onClick={(event) => {
																stopPropagation(event);
																toggleItem(itemKey);
															}}
															type="button"
														>
															<span className={styles.toggleIcon}>
																{expanded ? "▼" : "▶"}
															</span>
															<strong>{item.label || kind}</strong>
														</button>
													) : (
														<strong>
															{item.role === "user" ? "User" : "AI"}
														</strong>
													)}
													{item.timestamp && (
														<time
															className={styles.timestamp}
															dateTime={item.timestamp}
														>
															{item.timestamp}
														</time>
													)}
												</div>
												{item.token_count_count > 0 &&
													item.last_token_usage && (
														<span className={styles.tokenBadge}>
															{formatTokens(
																item.last_token_usage.total_tokens || 0,
															)}
														</span>
													)}
											</div>
											{expanded && (
												<div className={styles.expandedContent}>
													{truncated ? (
														<>
															<pre className={styles.detail}>
																{preview}
																{"\n..."}
															</pre>
															<button
																className={styles.showFullText}
																onClick={(event) => {
																	stopPropagation(event);
																	onShowFullText(item);
																}}
																type="button"
															>
																全文を表示
															</button>
														</>
													) : (
														<>
															<p className={styles.body}>{item.body}</p>
															{item.details?.length > 0 && (
																<pre className={styles.detail}>
																	{item.details
																		.map(
																			(detail) =>
																				`${detail.label}\n${detail.value}`,
																		)
																		.join("\n\n")}
																</pre>
															)}
															{item.kind === "collab" && (
																<button
																	type="button"
																	className={styles.subagentButton}
																	onClick={(event) => {
																		stopPropagation(event);
																		const threadIdDetail = item.details?.find(
																			(d) => d.label === "Thread ID",
																		);
																		const providerDetail = item.details?.find(
																			(d) => d.label === "Provider",
																		);
																		if (
																			threadIdDetail?.value &&
																			onSubagentClick
																		) {
																			onSubagentClick(
																				threadIdDetail.value,
																				providerDetail?.value,
																			);
																		}
																	}}
																>
																	サブエージェントを表示
																</button>
															)}
														</>
													)}
												</div>
											)}
											{item.token_count_count > 0 && (
												<div className={styles.tokenSubMetrics}>
													<span>
														In:{" "}
														{compactNumber.format(
															item.last_token_usage?.input_tokens || 0,
														)}
													</span>
													<span>
														Out:{" "}
														{compactNumber.format(
															item.last_token_usage?.output_tokens || 0,
														)}
													</span>
													{item.last_token_usage &&
														item.last_token_usage.reasoning_output_tokens >
															0 && (
															<span>
																Reasoning:{" "}
																{compactNumber.format(
																	item.last_token_usage.reasoning_output_tokens,
																)}
															</span>
														)}
													{item.total_token_usage && (
														<span className={styles.cumulativeTokens}>
															累計:{" "}
															{compactNumber.format(
																item.total_token_usage.total_tokens,
															)}
														</span>
													)}
												</div>
											)}
										</article>
									</div>
								);
							})}
						</div>
					</section>
				))}
			</div>
		</aside>
	);
}
