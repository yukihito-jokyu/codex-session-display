import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { SessionDetailError } from "./components/SessionDetailError/SessionDetailError";
import { SessionDetailHeader } from "./components/SessionDetailHeader/SessionDetailHeader";
import { SessionDetailMainContent } from "./components/SessionDetailMainContent/SessionDetailMainContent";
import { useSessionDetail } from "./hooks/useSessionDetail";
import { useTimelineResize } from "./hooks/useTimelineResize";
import styles from "./SessionDetailPage.module.css";

export function SessionDetailPage() {
	const { id } = useParams<{ id: string }>();
	const [searchParams] = useSearchParams();
	const provider =
		searchParams.get("provider") === "claude" ? "claude" : "codex";
	const navigate = useNavigate();
	const {
		sessionData,
		loading,
		error,
		selectedNode,
		selectedFlowNodeId,
		selectedTimelineId,
		selectedTokenCountIndices,
		timelineScrollTarget,
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
		handleTimelineSelect,
		handleTimelineFullText,
		handleOpenLogDirectory,
		handleCopyLogPath,
		startResize,
		handleResizerKeyDown,
	} = useSessionDetail(id, provider);
	const {
		timelineWidth,
		timelineMinWidth,
		timelineMaxWidth,
		startTimelineResize,
		handleTimelineResizerKeyDown,
	} = useTimelineResize();
	const handleBack = () => navigate("/");

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
			<SessionDetailError
				error={error}
				logActionMessage={logActionMessage}
				onRetry={retry}
				onBack={handleBack}
				onOpenLogDirectory={handleOpenLogDirectory}
				onCopyLogPath={handleCopyLogPath}
			/>
		);
	}

	const parentSessionId = sessionData?.parent_session_id ?? null;
	const handleBackToParent = () => {
		if (parentSessionId) {
			navigate(`/sessions/${parentSessionId}?provider=${provider}`);
		}
	};
	const handleSubagentClick = (subagentId: string) => {
		navigate(`/sessions/${subagentId}?provider=${provider}`);
	};

	return (
		<div className={styles.detailPage}>
			<SessionDetailHeader
				sessionId={id ?? ""}
				parentSessionId={parentSessionId}
				onBack={handleBack}
				onBackToParent={handleBackToParent}
			/>
			<SessionDetailMainContent
				sessionData={sessionData}
				selectedNode={selectedNode}
				selectedFlowNodeId={selectedFlowNodeId}
				selectedTimelineId={selectedTimelineId}
				selectedTokenCountIndices={selectedTokenCountIndices}
				timelineScrollTarget={timelineScrollTarget}
				splitRatio={splitRatio}
				zoomTarget={zoomTarget}
				splitContainerRef={splitContainerRef}
				boundTokenCounts={boundTokenCounts}
				latestBoundToken={latestBoundToken}
				showTokenSplit={showTokenSplit}
				timelineWidth={timelineWidth}
				timelineMinWidth={timelineMinWidth}
				timelineMaxWidth={timelineMaxWidth}
				onStartTimelineResize={startTimelineResize}
				onTimelineResizerKeyDown={handleTimelineResizerKeyDown}
				onTokenLogClick={handleTokenLogClick}
				onCanvasNodeSelect={handleCanvasNodeSelect}
				onTokenBadgeClick={handleTokenBadgeClick}
				onTimelineSelect={handleTimelineSelect}
				onTimelineFullText={handleTimelineFullText}
				onStartResize={startResize}
				onResizerKeyDown={handleResizerKeyDown}
				onSubagentClick={handleSubagentClick}
			/>
		</div>
	);
}
