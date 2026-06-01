import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { OpenLogDirectory } from "wailsjs/go/main/App";
import type { dto } from "wailsjs/go/models";
import { FlowCanvas } from "../../components/ui/FlowCanvas/FlowCanvas";
import { useSessionDetail } from "./hooks/useSessionDetail";
import styles from "./SessionDetailPage.module.css";

const RightPanel = lazy(() =>
	import("./components/RightPanel/RightPanel").then((module) => ({
		default: module.RightPanel,
	})),
);

type AppWindow = Window & {
	go?: {
		main?: {
			App?: {
				GetLogFilePath?: () => Promise<string>;
			};
		};
	};
};

async function getLogFilePath() {
	const app = (window as AppWindow).go?.main?.App;
	if (!app?.GetLogFilePath) {
		throw new Error("GetLogFilePath is unavailable");
	}
	return app.GetLogFilePath();
}

export function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const { sessionData, loading, error, selectedNode, handleNodeSelect, retry } =
		useSessionDetail(id);
	const [bottomPanelMode, setBottomPanelMode] = useState<"node" | "token">(
		"node",
	);
	const [splitRatio, setSplitRatio] = useState(0.52);
	const [zoomTarget, setZoomTarget] = useState<{
		nodeId: string;
		timestamp: number;
	} | null>(null);
	const [logActionMessage, setLogActionMessage] = useState<string | null>(null);
	const splitContainerRef = useRef<HTMLDivElement | null>(null);
	const dragStateRef = useRef<{
		startX: number;
		startRatio: number;
		width: number;
	} | null>(null);

	const handleTokenLogClick = (nodeId: string) => {
		if (!sessionData?.nodes) return;
		const node = sessionData.nodes.find((n) => n.id === nodeId);
		if (node) {
			handleNodeSelect(node);
			setBottomPanelMode("token");
			setZoomTarget({ nodeId, timestamp: Date.now() });
		}
	};

	const handleCanvasNodeSelect = (node: dto.FlowNode | null) => {
		handleNodeSelect(node);
		if (node) {
			setBottomPanelMode("node");
		}
	};

	const handleTokenBadgeClick = (node: dto.FlowNode) => {
		handleNodeSelect(node);
		setBottomPanelMode("token");
	};

	const handleOpenLogDirectory = async () => {
		try {
			await OpenLogDirectory();
			setLogActionMessage("ログフォルダを開きました");
		} catch (openError) {
			setLogActionMessage(
				`ログフォルダを開けませんでした: ${String(openError)}`,
			);
		}
	};

	const handleCopyLogPath = async () => {
		try {
			const logPath = await getLogFilePath();
			await navigator.clipboard.writeText(logPath);
			setLogActionMessage("ログパスをコピーしました");
		} catch (copyError) {
			setLogActionMessage(
				`ログパスをコピーできませんでした: ${String(copyError)}`,
			);
		}
	};

	useEffect(() => {
		if (!selectedNode) {
			setBottomPanelMode("node");
		}
	}, [selectedNode]);

	useEffect(() => {
		const handlePointerMove = (event: PointerEvent) => {
			const dragState = dragStateRef.current;
			if (!dragState) {
				return;
			}
			const deltaX = event.clientX - dragState.startX;
			const nextRatio =
				(dragState.startRatio * dragState.width + deltaX) / dragState.width;
			setSplitRatio(Math.min(0.75, Math.max(0.25, nextRatio)));
		};

		const handlePointerUp = () => {
			dragStateRef.current = null;
		};

		window.addEventListener("pointermove", handlePointerMove);
		window.addEventListener("pointerup", handlePointerUp);

		return () => {
			window.removeEventListener("pointermove", handlePointerMove);
			window.removeEventListener("pointerup", handlePointerUp);
		};
	}, []);

	const boundTokenCounts = useMemo(() => {
		if (!selectedNode || !sessionData?.token_counts) {
			return [];
		}
		return sessionData.token_counts.filter(
			(entry) => entry.bound_to_node_id === selectedNode.id,
		);
	}, [selectedNode, sessionData?.token_counts]);

	const latestBoundToken = boundTokenCounts.at(-1)?.total_token_usage;

	const formatNumber = (value?: number) => {
		return new Intl.NumberFormat().format(value ?? 0);
	};

	const startResize = (event: React.PointerEvent<HTMLDivElement>) => {
		if (!splitContainerRef.current) {
			return;
		}
		const rect = splitContainerRef.current.getBoundingClientRect();
		dragStateRef.current = {
			startX: event.clientX,
			startRatio: splitRatio,
			width: rect.width,
		};
	};

	const handleResizerKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
		if (event.key === "ArrowLeft") {
			event.preventDefault();
			setSplitRatio((current) => Math.max(0.25, current - 0.03));
		}
		if (event.key === "ArrowRight") {
			event.preventDefault();
			setSplitRatio((current) => Math.min(0.75, current + 0.03));
		}
	};

	const showTokenSplit =
		bottomPanelMode === "token" && boundTokenCounts.length > 0;

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
