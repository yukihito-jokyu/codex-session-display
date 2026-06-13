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
