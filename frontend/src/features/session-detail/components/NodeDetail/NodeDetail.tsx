import { useMemo, useState } from "react";
import type { dto } from "wailsjs/go/models";
import styles from "./NodeDetail.module.css";

type NodeDetailProps = {
	node: dto.FlowNode;
	splitRatio?: number;
};

function findMatchIndexes(text: string, query: string) {
	const indexes: number[] = [];
	const normalizedQuery = query.toLocaleLowerCase();
	const normalizedText = text.toLocaleLowerCase();
	let startIndex = 0;

	while (normalizedQuery && startIndex < text.length) {
		const matchIndex = normalizedText.indexOf(normalizedQuery, startIndex);
		if (matchIndex === -1) {
			break;
		}
		indexes.push(matchIndex);
		startIndex = matchIndex + normalizedQuery.length;
	}

	return indexes;
}

export function NodeDetail({ node, splitRatio }: NodeDetailProps) {
	const [searchQuery, setSearchQuery] = useState("");
	const fullText = node.data?.fullText || "";
	const normalizedQuery = searchQuery.trim();
	const matchIndexes = useMemo(
		() => findMatchIndexes(fullText, normalizedQuery),
		[fullText, normalizedQuery],
	);
	const highlightedFullText = useMemo(() => {
		if (!normalizedQuery || matchIndexes.length === 0) {
			return fullText;
		}

		const parts = [];
		let startIndex = 0;
		for (const matchIndex of matchIndexes) {
			parts.push(fullText.slice(startIndex, matchIndex));
			parts.push(
				<mark key={matchIndex}>
					{fullText.slice(matchIndex, matchIndex + normalizedQuery.length)}
				</mark>,
			);
			startIndex = matchIndex + normalizedQuery.length;
		}
		parts.push(fullText.slice(startIndex));
		return parts;
	}, [fullText, matchIndexes, normalizedQuery]);

	return (
		<div
			className={styles.nodeDetail}
			data-testid="bottom-panel-node-detail"
			style={
				splitRatio === undefined
					? undefined
					: {
							width: `${splitRatio * 100}%`,
							flex: `0 0 ${splitRatio * 100}%`,
						}
			}
		>
			{node.data?.meta && Object.keys(node.data.meta).length > 0 && (
				<table className={styles.metaTable}>
					<tbody>
						{Object.entries(node.data.meta).map(([key, val]) => (
							<tr key={key}>
								<th>{key}</th>
								<td>
									{typeof val === "object" ? JSON.stringify(val) : String(val)}
								</td>
							</tr>
						))}
					</tbody>
				</table>
			)}
			{node.data?.fullText && (
				<div>
					<div className={styles.fullTextHeader}>
						<div className={styles.fullTextLabel}>Full Text</div>
						<label className={styles.searchField}>
							<span>全文を検索</span>
							<input
								aria-label="全文を検索"
								onChange={(event) => setSearchQuery(event.target.value)}
								type="search"
								value={searchQuery}
							/>
						</label>
						{normalizedQuery && (
							<span className={styles.matchCount}>{matchIndexes.length}件</span>
						)}
					</div>
					<pre className={styles.fullTextContent}>{highlightedFullText}</pre>
				</div>
			)}
			{!node.data?.fullText &&
				(!node.data?.meta || Object.keys(node.data.meta).length === 0) && (
					<div>No additional details available.</div>
				)}
		</div>
	);
}
