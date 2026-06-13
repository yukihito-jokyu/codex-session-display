import { lazy, Suspense } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { FlowCanvas } from "../../components/ui/FlowCanvas/FlowCanvas";
import { useSessionDetail } from "./hooks/useSessionDetail";
import styles from "./SessionDetailPage.module.css";

const RightPanel = lazy(() =>
	import("./components/RightPanel/RightPanel").then((module) => ({
		default: module.RightPanel,
	})),
);

function formatNumber(value?: number) {
	return new Intl.NumberFormat().format(value ?? 0);
}

export function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const {
		sessionData,
		loading,
		error,
		selectedNode,
		splitRatio,
		zoomTarget,
		logActionMessage,
		splitContainerRef,
		boundTokenCounts,
		latestBoundToken,
		showTokenSplit,
		retry,
		handleTokenLogClick,
		handleCanvasNodeSelect,
		handleTokenBadgeClick,
		handleOpenLogDirectory,
		handleCopyLogPath,
		startResize,
		handleResizerKeyDown,
	} = useSessionDetail(id);

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
				<div className={styles.errorActions}>
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
				<div className={styles.errorActions}>
					<button
						type="button"
						className={styles.secondaryBtn}
						onClick={handleOpenLogDirectory}
					>
						ログフォルダを開く
					</button>
					<button
						type="button"
						className={styles.secondaryBtn}
						onClick={handleCopyLogPath}
					>
						ログパスをコピー
					</button>
				</div>
				{logActionMessage && (
					<span className={styles.logActionMessage}>{logActionMessage}</span>
				)}
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
								onNodeSelect={handleCanvasNodeSelect}
								onTokenBadgeClick={handleTokenBadgeClick}
								interactionLocked={Boolean(selectedNode)}
								selectedNodeId={selectedNode?.id}
								selectedNodeType={selectedNode?.type}
								zoomTarget={zoomTarget}
							/>
						)}
					</div>

					{selectedNode && (
						<div className={styles.bottomPanel} data-testid="bottom-panel">
							<div className={styles.bottomPanelHeader}>
								<span className={styles.bottomPanelTitle}>
									Node Detail: {selectedNode.data?.label || selectedNode.id}
								</span>
								<button
									type="button"
									className={styles.panelCloseBtn}
									onClick={() => handleCanvasNodeSelect(null)}
								>
									✕
								</button>
							</div>
							<div
								ref={splitContainerRef}
								className={`${styles.bottomPanelContent} ${
									showTokenSplit ? styles.bottomPanelContentSplit : ""
								}`}
								data-testid={
									showTokenSplit ? "bottom-panel-split" : "bottom-panel-single"
								}
							>
								<div
									className={styles.panelLeft}
									data-testid="bottom-panel-node-detail"
									style={
										showTokenSplit
											? {
													width: `${splitRatio * 100}%`,
													flex: `0 0 ${splitRatio * 100}%`,
												}
											: undefined
									}
								>
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
								{showTokenSplit && (
									<>
										<div
											className={styles.panelResizer}
											data-testid="bottom-panel-resizer"
											onPointerDown={startResize}
											onKeyDown={handleResizerKeyDown}
											role="separator"
											aria-orientation="vertical"
											aria-label="Resize token detail panel"
											aria-valuemin={25}
											aria-valuemax={75}
											aria-valuenow={Math.round(splitRatio * 100)}
											tabIndex={0}
										/>
										<div
											className={styles.panelRight}
											data-testid="bottom-panel-token-detail"
										>
											<div className={styles.tokenDetailHeader}>
												<span>Token Detail</span>
												<span className={styles.tokenDetailBadge}>
													{boundTokenCounts.length} entries
												</span>
											</div>
											{latestBoundToken && (
												<div className={styles.tokenSummaryGrid}>
													<div className={styles.tokenSummaryItem}>
														<span>Total</span>
														<strong>
															{formatNumber(latestBoundToken.total_tokens)}
														</strong>
													</div>
													<div className={styles.tokenSummaryItem}>
														<span>Input</span>
														<strong>
															{formatNumber(latestBoundToken.input_tokens)}
														</strong>
													</div>
													<div className={styles.tokenSummaryItem}>
														<span>Output</span>
														<strong>
															{formatNumber(latestBoundToken.output_tokens)}
														</strong>
													</div>
													<div className={styles.tokenSummaryItem}>
														<span>Reasoning</span>
														<strong>
															{formatNumber(
																latestBoundToken.reasoning_output_tokens,
															)}
														</strong>
													</div>
													<div className={styles.tokenSummaryItem}>
														<span>Cached</span>
														<strong>
															{formatNumber(
																latestBoundToken.cached_input_tokens,
															)}
														</strong>
													</div>
												</div>
											)}
											<div className={styles.tokenEntries}>
												{boundTokenCounts.map((entry) => {
													const usage =
														entry.total_token_usage ?? entry.last_token_usage;
													return (
														<div
															key={entry.index}
															className={styles.tokenEntryCard}
														>
															<div className={styles.tokenEntryHeader}>
																<span>Index #{entry.index}</span>
																<span>Turn #{entry.turn_index + 1}</span>
															</div>
															<div className={styles.tokenEntryGrid}>
																<div>
																	<span>Total</span>
																	<strong>
																		{formatNumber(usage?.total_tokens)}
																	</strong>
																</div>
																<div>
																	<span>Input</span>
																	<strong>
																		{formatNumber(usage?.input_tokens)}
																	</strong>
																</div>
																<div>
																	<span>Output</span>
																	<strong>
																		{formatNumber(usage?.output_tokens)}
																	</strong>
																</div>
																<div>
																	<span>Reasoning</span>
																	<strong>
																		{formatNumber(
																			usage?.reasoning_output_tokens,
																		)}
																	</strong>
																</div>
																<div>
																	<span>Cached</span>
																	<strong>
																		{formatNumber(usage?.cached_input_tokens)}
																	</strong>
																</div>
															</div>
														</div>
													);
												})}
											</div>
										</div>
									</>
								)}
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
