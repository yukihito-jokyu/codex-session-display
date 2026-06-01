import { lazy, Suspense, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { FlowCanvas } from "../../components/ui/FlowCanvas/FlowCanvas";
import { useSessionDetail } from "./hooks/useSessionDetail";
import styles from "./SessionDetailPage.module.css";

const RightPanel = lazy(() =>
	import("./components/RightPanel/RightPanel").then((module) => ({
		default: module.RightPanel,
	})),
);

export function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const { sessionData, loading, error, selectedNode, handleNodeSelect, retry } =
		useSessionDetail(id);
	const [zoomTarget, setZoomTarget] = useState<{
		nodeId: string;
		timestamp: number;
	} | null>(null);

	const handleTokenLogClick = (nodeId: string) => {
		if (!sessionData?.nodes) return;
		const node = sessionData.nodes.find((n) => n.id === nodeId);
		if (node) {
			handleNodeSelect(node);
			setZoomTarget({ nodeId, timestamp: Date.now() });
		}
	};

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
				<div style={{ display: "flex", gap: "8px" }}>
					<button type="button" className={styles.retryBtn} onClick={retry}>
						Retry
					</button>
					<button
						type="button"
						className={styles.backBtn}
						onClick={() => navigate("/")}
					>
						Back to List
					</button>
				</div>
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
				<div className={styles.mainArea}>
					<div className={styles.canvasWrapper}>
						{sessionData && (
							<FlowCanvas
								nodes={sessionData.nodes || []}
								edges={sessionData.edges || []}
								onNodeSelect={handleNodeSelect}
								selectedNodeId={selectedNode?.id}
								selectedNodeType={selectedNode?.type}
								zoomTarget={zoomTarget}
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

				{sessionData?.statistics && (
					<Suspense fallback={null}>
						<RightPanel
							statistics={sessionData.statistics}
							tokenCounts={sessionData.token_counts || []}
							nodes={sessionData.nodes || []}
							onTokenLogClick={handleTokenLogClick}
						/>
					</Suspense>
				)}
			</div>
		</div>
	);
}
