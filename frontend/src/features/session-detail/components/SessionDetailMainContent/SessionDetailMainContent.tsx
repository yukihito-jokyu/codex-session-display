import {
	type KeyboardEventHandler,
	lazy,
	type MutableRefObject,
	type PointerEventHandler,
	Suspense,
} from "react";
import type { dto } from "wailsjs/go/models";
import { FlowCanvas } from "../../../../components/ui/FlowCanvas/FlowCanvas";
import { BottomPanel } from "../BottomPanel/BottomPanel";
import { ConversationTimeline } from "../ConversationTimeline/ConversationTimeline";
import styles from "./SessionDetailMainContent.module.css";

const RightPanel = lazy(() =>
	import("../RightPanel/RightPanel").then((module) => ({
		default: module.RightPanel,
	})),
);

type SessionDetailMainContentProps = {
	sessionData: dto.SessionDetailResponse | null;
	selectedNode: dto.FlowNode | null;
	selectedFlowNodeId: string | null;
	selectedTimelineId: string | null;
	selectedTokenCountIndices: number[];
	timelineScrollTarget: { selectionId: string; timestamp: number } | null;
	splitRatio: number;
	zoomTarget: { nodeId: string; timestamp: number } | null;
	splitContainerRef: MutableRefObject<HTMLDivElement | null>;
	boundTokenCounts: dto.TokenCountEntry[];
	latestBoundToken: dto.TokenCountEntry["total_token_usage"];
	showTokenSplit: boolean;
	timelineWidth: number;
	timelineMinWidth: number;
	timelineMaxWidth: number;
	onStartTimelineResize: PointerEventHandler<HTMLDivElement>;
	onTimelineResizerKeyDown: KeyboardEventHandler<HTMLDivElement>;
	onTokenLogClick: (tokenIndex: number) => void;
	onCanvasNodeSelect: (node: dto.FlowNode | null) => void;
	onTokenBadgeClick: (node: dto.FlowNode) => void;
	onTimelineSelect: (item: dto.ConversationTimelineItem) => void;
	onTimelineFullText: (item: dto.ConversationTimelineItem) => void;
	onStartResize: PointerEventHandler<HTMLDivElement>;
	onResizerKeyDown: KeyboardEventHandler<HTMLDivElement>;
	onSubagentClick?: (threadId: string) => void;
};

export function SessionDetailMainContent({
	sessionData,
	selectedNode,
	selectedFlowNodeId,
	selectedTimelineId,
	selectedTokenCountIndices,
	timelineScrollTarget,
	splitRatio,
	zoomTarget,
	splitContainerRef,
	boundTokenCounts,
	latestBoundToken,
	showTokenSplit,
	timelineWidth,
	timelineMinWidth,
	timelineMaxWidth,
	onStartTimelineResize,
	onTimelineResizerKeyDown,
	onTokenLogClick,
	onCanvasNodeSelect,
	onTokenBadgeClick,
	onTimelineSelect,
	onTimelineFullText,
	onStartResize,
	onResizerKeyDown,
	onSubagentClick,
}: SessionDetailMainContentProps) {
	const timelineDetailSelected = selectedNode?.data?.category === "timeline";
	const selectedFlowNode = sessionData?.nodes?.find(
		(node) => node.id === selectedFlowNodeId,
	);

	return (
		<div className={styles.detailContent}>
			{sessionData && (
				<>
					<div className={styles.timelinePane} style={{ width: timelineWidth }}>
						<ConversationTimeline
							turns={sessionData.timeline || []}
							selectedSelectionId={selectedTimelineId}
							scrollTarget={timelineScrollTarget}
							onSelect={onTimelineSelect}
							onShowFullText={onTimelineFullText}
							onSubagentClick={onSubagentClick}
						/>
					</div>
					<div
						className={styles.timelineResizer}
						data-testid="timeline-resizer"
						role="separator"
						aria-orientation="vertical"
						aria-label="タイムラインの幅を変更"
						aria-valuemin={timelineMinWidth}
						aria-valuemax={timelineMaxWidth}
						aria-valuenow={timelineWidth}
						onPointerDown={onStartTimelineResize}
						onKeyDown={onTimelineResizerKeyDown}
						tabIndex={0}
					/>
				</>
			)}
			<div className={styles.mainArea}>
				<div className={styles.canvasWrapper}>
					{sessionData && (
						<FlowCanvas
							nodes={sessionData.nodes || []}
							edges={sessionData.edges || []}
							onNodeSelect={onCanvasNodeSelect}
							onTokenBadgeClick={onTokenBadgeClick}
							onSubagentClick={onSubagentClick}
							interactionLocked={Boolean(
								selectedFlowNodeId && !timelineDetailSelected,
							)}
							selectedNodeId={selectedFlowNodeId ?? undefined}
							selectedNodeType={selectedFlowNode?.type}
							zoomTarget={zoomTarget}
						/>
					)}
				</div>

				{selectedNode && (
					<BottomPanel
						node={selectedNode}
						showTokenSplit={showTokenSplit}
						splitRatio={splitRatio}
						splitContainerRef={splitContainerRef}
						boundTokenCounts={boundTokenCounts}
						latestBoundToken={latestBoundToken}
						onClose={() => onCanvasNodeSelect(null)}
						onStartResize={onStartResize}
						onResizerKeyDown={onResizerKeyDown}
					/>
				)}
			</div>

			{sessionData?.statistics && (
				<Suspense fallback={null}>
					<RightPanel
						sessionId={sessionData.id}
						subagents={sessionData.subagents}
						statistics={sessionData.statistics}
						transcriptStats={sessionData.transcript_stats}
						tokenCounts={sessionData.token_counts || []}
						nodes={sessionData.nodes || []}
						selectedTokenCountIndices={selectedTokenCountIndices}
						onTokenLogClick={onTokenLogClick}
					/>
				</Suspense>
			)}
		</div>
	);
}
