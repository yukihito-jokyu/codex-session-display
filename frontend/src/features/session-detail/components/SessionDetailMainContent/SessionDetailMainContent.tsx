import {
	type KeyboardEventHandler,
	lazy,
	type MutableRefObject,
	type PointerEventHandler,
	Suspense,
} from "react";
import type { dto } from "wailsjs/go/models";
import { FlowCanvas } from "../../../../components/ui/FlowCanvas/FlowCanvas";
import styles from "./SessionDetailMainContent.module.css";

const RightPanel = lazy(() =>
	import("../RightPanel/RightPanel").then((module) => ({
		default: module.RightPanel,
	})),
);

type SessionDetailMainContentProps = {
	sessionData: dto.SessionDetailResponse | null;
	selectedNode: dto.FlowNode | null;
	splitRatio: number;
	zoomTarget: { nodeId: string; timestamp: number } | null;
	splitContainerRef: MutableRefObject<HTMLDivElement | null>;
	boundTokenCounts: dto.TokenCountEntry[];
	latestBoundToken: dto.TokenCountEntry["total_token_usage"];
	showTokenSplit: boolean;
	onTokenLogClick: (nodeId: string) => void;
	onCanvasNodeSelect: (node: dto.FlowNode | null) => void;
	onTokenBadgeClick: (node: dto.FlowNode) => void;
	onStartResize: PointerEventHandler<HTMLDivElement>;
	onResizerKeyDown: KeyboardEventHandler<HTMLDivElement>;
};

function formatNumber(value?: number) {
	return new Intl.NumberFormat().format(value ?? 0);
}

export function SessionDetailMainContent({
	sessionData,
	selectedNode,
	splitRatio,
	zoomTarget,
	splitContainerRef,
	boundTokenCounts,
	latestBoundToken,
	showTokenSplit,
	onTokenLogClick,
	onCanvasNodeSelect,
	onTokenBadgeClick,
	onStartResize,
	onResizerKeyDown,
}: SessionDetailMainContentProps) {
	return (
		<div className={styles.detailContent}>
			<div className={styles.mainArea}>
				<div className={styles.canvasWrapper}>
					{sessionData && (
						<FlowCanvas
							nodes={sessionData.nodes || []}
							edges={sessionData.edges || []}
							onNodeSelect={onCanvasNodeSelect}
							onTokenBadgeClick={onTokenBadgeClick}
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
								onClick={() => onCanvasNodeSelect(null)}
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
										onPointerDown={onStartResize}
										onKeyDown={onResizerKeyDown}
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
														{formatNumber(latestBoundToken.cached_input_tokens)}
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
																	{formatNumber(usage?.reasoning_output_tokens)}
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
						onTokenLogClick={onTokenLogClick}
					/>
				</Suspense>
			)}
		</div>
	);
}
