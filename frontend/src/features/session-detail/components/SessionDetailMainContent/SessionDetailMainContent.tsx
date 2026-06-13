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
	onTokenLogClick: (nodeId: string) => void;
	onCanvasNodeSelect: (node: dto.FlowNode | null) => void;
	onTokenBadgeClick: (node: dto.FlowNode) => void;
	onTimelineFullText: (item: dto.ConversationTimelineItem) => void;
	onStartResize: PointerEventHandler<HTMLDivElement>;
	onResizerKeyDown: KeyboardEventHandler<HTMLDivElement>;
};

export function SessionDetailMainContent({
	sessionData,
	selectedNode,
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
	onTimelineFullText,
	onStartResize,
	onResizerKeyDown,
}: SessionDetailMainContentProps) {
	const timelineDetailSelected = selectedNode?.data?.category === "timeline";

	return (
		<div className={styles.detailContent}>
			{sessionData && (
				<>
					<div className={styles.timelinePane} style={{ width: timelineWidth }}>
						<ConversationTimeline
							turns={sessionData.timeline || []}
							onShowFullText={onTimelineFullText}
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
							interactionLocked={Boolean(
								selectedNode && !timelineDetailSelected,
							)}
							selectedNodeId={
								timelineDetailSelected ? undefined : selectedNode?.id
							}
							selectedNodeType={
								timelineDetailSelected ? undefined : selectedNode?.type
							}
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
