import { useNavigate, useParams } from "react-router-dom";
import { SessionDetailError } from "./components/SessionDetailError/SessionDetailError";
import { SessionDetailHeader } from "./components/SessionDetailHeader/SessionDetailHeader";
import { SessionDetailMainContent } from "./components/SessionDetailMainContent/SessionDetailMainContent";
import { useSessionDetail } from "./hooks/useSessionDetail";
import { useTimelineResize } from "./hooks/useTimelineResize";
import styles from "./SessionDetailPage.module.css";

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
		handleTimelineFullText,
		handleOpenLogDirectory,
		handleCopyLogPath,
		startResize,
		handleResizerKeyDown,
	} = useSessionDetail(id);
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

	return (
		<div className={styles.detailPage}>
			<SessionDetailHeader sessionId={id ?? ""} onBack={handleBack} />
			<SessionDetailMainContent
				sessionData={sessionData}
				selectedNode={selectedNode}
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
				onTimelineFullText={handleTimelineFullText}
				onStartResize={startResize}
				onResizerKeyDown={handleResizerKeyDown}
			/>
		</div>
	);
}
