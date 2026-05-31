import { useNavigate, useParams } from "react-router-dom";
import { FlowCanvas } from "../../components/ui/FlowCanvas/FlowCanvas";
import { useSessionDetail } from "./hooks/useSessionDetail";
import styles from "./SessionDetailPage.module.css";

export function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const { sessionData, loading, error, selectedNode, handleNodeSelect } =
		useSessionDetail(id);

	if (loading) {
		return (
			<div className={styles.loading}>
				<div className={styles.spinner}></div>
				<span>Loading session detail...</span>
			</div>
		);
	}

	if (error) {
		return (
			<div className={styles.errorContainer}>
				<span className={styles.errorIcon}>⚠️</span>
				<span className={styles.errorMessage}>{error}</span>
				<button
					type="button"
					className={styles.retryBtn}
					onClick={() => navigate("/")}
				>
					Back to List
				</button>
			</div>
		);
	}

	return (
		<div className={styles.detailPage}>
			<div className={styles.detailHeader}>
				<button
					type="button"
					className={styles.backBtn}
					onClick={() => navigate("/")}
				>
					← Back to List
				</button>
				<span className={styles.detailTitle}>Session Detail: {id}</span>
			</div>
			<div className={styles.detailContent}>
				<div className={styles.canvasWrapper}>
					{sessionData && (
						<FlowCanvas
							nodes={sessionData.nodes || []}
							edges={sessionData.edges || []}
							onNodeSelect={handleNodeSelect}
							selectedNodeId={selectedNode?.id}
						/>
					)}
				</div>

				{selectedNode && (
					<div className={styles.bottomPanel}>
						<div className={styles.bottomPanelHeader}>
							<span className={styles.bottomPanelTitle}>
								Node Detail: {selectedNode.data?.label || selectedNode.id}
							</span>
							<button
								type="button"
								className={styles.panelCloseBtn}
								onClick={() => handleNodeSelect(null)}
							>
								✕
							</button>
						</div>
						<div className={styles.bottomPanelContent}>
							<div className={styles.panelLeft}>
								{selectedNode.data?.meta &&
									Object.keys(selectedNode.data.meta).length > 0 && (
										<table className={styles.metaTable}>
											<tbody>
												{Object.entries(selectedNode.data.meta).map(
													([key, val]) => (
														<tr key={key}>
															<th>{key}</th>
															<td>
																{typeof val === "object"
																	? JSON.stringify(val)
																	: String(val)}
															</td>
														</tr>
													),
												)}
											</tbody>
										</table>
									)}
								{selectedNode.data?.fullText && (
									<div>
										<div className={styles.fullTextLabel}>Full Text</div>
										<pre className={styles.fullTextContent}>
											{selectedNode.data.fullText}
										</pre>
									</div>
								)}
								{!selectedNode.data?.fullText &&
									(!selectedNode.data?.meta ||
										Object.keys(selectedNode.data.meta).length === 0) && (
										<div>No additional details available.</div>
									)}
							</div>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}
