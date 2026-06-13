import type { dto } from "wailsjs/go/models";
import styles from "./NodeDetail.module.css";

type NodeDetailProps = {
	node: dto.FlowNode;
	splitRatio?: number;
};

export function NodeDetail({ node, splitRatio }: NodeDetailProps) {
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
					<div className={styles.fullTextLabel}>Full Text</div>
					<pre className={styles.fullTextContent}>{node.data.fullText}</pre>
				</div>
			)}
			{!node.data?.fullText &&
				(!node.data?.meta || Object.keys(node.data.meta).length === 0) && (
					<div>No additional details available.</div>
				)}
		</div>
	);
}
