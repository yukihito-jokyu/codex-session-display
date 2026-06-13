import { useMemo, useState } from "react";
import type { dto } from "wailsjs/go/models";
import styles from "./ConversationTimeline.module.css";

type ConversationTimelineProps = {
	turns: dto.ConversationTimelineTurn[];
};

type TimelineCategory =
	| "conversation"
	| "reasoning"
	| "tool"
	| "reference"
	| "system";

const CATEGORY_FILTERS: ReadonlyArray<{
	value: TimelineCategory;
	label: string;
}> = [
	{ value: "conversation", label: "会話" },
	{ value: "reasoning", label: "推論" },
	{ value: "tool", label: "ツール・コマンド" },
	{ value: "reference", label: "参照情報" },
	{ value: "system", label: "システムイベント" },
];

const compactNumber = new Intl.NumberFormat("en", {
	notation: "compact",
	maximumFractionDigits: 1,
});

function getTimelineCategory(kind: string): TimelineCategory {
	switch (kind) {
		case "conversation":
			return "conversation";
		case "reasoning":
			return "reasoning";
		case "tool":
		case "web":
		case "mcp":
			return "tool";
		case "instructions":
		case "reference":
			return "reference";
		default:
			return "system";
	}
}

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
	selectedCategories: ReadonlySet<TimelineCategory>,
	measuredOnly: boolean,
	normalizedQuery: string,
) {
	return (
		selectedCategories.has(getTimelineCategory(item.kind || "system")) &&
		(!measuredOnly || item.token_count_count > 0) &&
		matchesSearchQuery(item, normalizedQuery)
	);
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

export function ConversationTimeline({ turns }: ConversationTimelineProps) {
	const [searchQuery, setSearchQuery] = useState("");
	const [measuredOnly, setMeasuredOnly] = useState(false);
	const [selectedCategories, setSelectedCategories] = useState<
		Set<TimelineCategory>
	>(() => new Set(CATEGORY_FILTERS.map((category) => category.value)));
	const [expandedItems, setExpandedItems] = useState<Set<string>>(
		() => new Set(),
	);
	const filteredTurns = useMemo(() => {
		const normalizedQuery = searchQuery.trim().toLocaleLowerCase();

		return turns.flatMap((turn) => {
			const items = turn.items.filter((item) =>
				matchesTimelineFilters(
					item,
					selectedCategories,
					measuredOnly,
					normalizedQuery,
				),
			);

			return items.length > 0 ? [{ ...turn, items }] : [];
		});
	}, [measuredOnly, searchQuery, selectedCategories, turns]);

	const toggleCategory = (category: TimelineCategory) => {
		setSelectedCategories((current) => {
			const next = new Set(current);
			if (next.has(category)) {
				next.delete(category);
			} else {
				next.add(category);
			}
			return next;
		});
	};

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
				<fieldset className={styles.categoryFilters}>
					<legend>種別</legend>
					{CATEGORY_FILTERS.map((category) => (
						<label key={category.value}>
							<input
								checked={selectedCategories.has(category.value)}
								onChange={() => toggleCategory(category.value)}
								type="checkbox"
							/>
							<span>{category.label}</span>
						</label>
					))}
				</fieldset>
				<label className={styles.checkboxFilter}>
					<input
						checked={measuredOnly}
						onChange={(event) => setMeasuredOnly(event.target.checked)}
						type="checkbox"
					/>
					<span>トークン計測あり</span>
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

						<div className={styles.itemList}>
							{turn.items.map((item) => {
								const kind = item.kind || "conversation";
								const itemKey = getTimelineItemKey(turn, item);
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
