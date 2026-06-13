import type { dto } from "wailsjs/go/models";
import styles from "./TokenDetail.module.css";

type TokenDetailProps = {
	entries: dto.TokenCountEntry[];
	latestUsage: dto.TokenCountEntry["total_token_usage"];
};

function formatNumber(value?: number) {
	return new Intl.NumberFormat().format(value ?? 0);
}

export function TokenDetail({ entries, latestUsage }: TokenDetailProps) {
	return (
		<div className={styles.tokenDetail} data-testid="bottom-panel-token-detail">
			<div className={styles.tokenDetailHeader}>
				<span>Token Detail</span>
				<span className={styles.tokenDetailBadge}>
					{entries.length} entries
				</span>
			</div>
			{latestUsage && (
				<div className={styles.tokenSummaryGrid}>
					<div className={styles.tokenSummaryItem}>
						<span>Total</span>
						<strong>{formatNumber(latestUsage.total_tokens)}</strong>
					</div>
					<div className={styles.tokenSummaryItem}>
						<span>Input</span>
						<strong>{formatNumber(latestUsage.input_tokens)}</strong>
					</div>
					<div className={styles.tokenSummaryItem}>
						<span>Output</span>
						<strong>{formatNumber(latestUsage.output_tokens)}</strong>
					</div>
					<div className={styles.tokenSummaryItem}>
						<span>Reasoning</span>
						<strong>{formatNumber(latestUsage.reasoning_output_tokens)}</strong>
					</div>
					<div className={styles.tokenSummaryItem}>
						<span>Cached</span>
						<strong>{formatNumber(latestUsage.cached_input_tokens)}</strong>
					</div>
				</div>
			)}
			<div className={styles.tokenEntries}>
				{entries.map((entry) => {
					const usage = entry.total_token_usage ?? entry.last_token_usage;
					return (
						<div key={entry.index} className={styles.tokenEntryCard}>
							<div className={styles.tokenEntryHeader}>
								<span>Index #{entry.index}</span>
								<span>Turn #{entry.turn_index + 1}</span>
							</div>
							<div className={styles.tokenEntryGrid}>
								<div>
									<span>Total</span>
									<strong>{formatNumber(usage?.total_tokens)}</strong>
								</div>
								<div>
									<span>Input</span>
									<strong>{formatNumber(usage?.input_tokens)}</strong>
								</div>
								<div>
									<span>Output</span>
									<strong>{formatNumber(usage?.output_tokens)}</strong>
								</div>
								<div>
									<span>Reasoning</span>
									<strong>
										{formatNumber(usage?.reasoning_output_tokens)}
									</strong>
								</div>
								<div>
									<span>Cached</span>
									<strong>{formatNumber(usage?.cached_input_tokens)}</strong>
								</div>
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}
