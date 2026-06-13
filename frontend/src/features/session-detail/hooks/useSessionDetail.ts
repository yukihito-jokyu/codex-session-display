import {
	type KeyboardEvent as ReactKeyboardEvent,
	type PointerEvent as ReactPointerEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { GetSessionDetail, OpenLogDirectory } from "wailsjs/go/main/App";
import { dto } from "wailsjs/go/models";
import { getTimelineItemFullText } from "../components/ConversationTimeline/timelineItemText";

type BottomPanelMode = "node" | "token";

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

export function useSessionDetail(id: string | undefined) {
	const [sessionData, setSessionData] =
		useState<dto.SessionDetailResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [selectedNode, setSelectedNode] = useState<dto.FlowNode | null>(null);
	const [bottomPanelMode, setBottomPanelMode] =
		useState<BottomPanelMode>("node");
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

	const fetchSessionDetail = useCallback((targetId: string) => {
		setLoading(true);
		setError(null);
		GetSessionDetail(targetId)
			.then((data) => {
				setSessionData(data);
				setLoading(false);
			})
			.catch((err) => {
				console.error(err);
				setError(`Failed to fetch session detail: ${err.message || err}`);
				setLoading(false);
			});
	}, []);

	useEffect(() => {
		if (!id) {
			setSessionData(null);
			setError("Session ID is missing.");
			setLoading(false);
			return;
		}
		fetchSessionDetail(id);
	}, [id, fetchSessionDetail]);

	const retry = useCallback(() => {
		if (id) {
			fetchSessionDetail(id);
		}
	}, [id, fetchSessionDetail]);

	const handleTokenLogClick = useCallback(
		(nodeId: string) => {
			const node = sessionData?.nodes?.find(
				(candidate) => candidate.id === nodeId,
			);
			if (!node) {
				return;
			}
			setSelectedNode(node);
			setBottomPanelMode("token");
			setZoomTarget({ nodeId, timestamp: Date.now() });
		},
		[sessionData?.nodes],
	);

	const handleCanvasNodeSelect = useCallback((node: dto.FlowNode | null) => {
		setSelectedNode(node);
		if (node) {
			setBottomPanelMode("node");
		}
	}, []);

	const handleTokenBadgeClick = useCallback((node: dto.FlowNode) => {
		setSelectedNode(node);
		setBottomPanelMode("token");
	}, []);

	const handleTimelineFullText = useCallback(
		(item: dto.ConversationTimelineItem) => {
			setSelectedNode(
				dto.FlowNode.createFrom({
					id: `timeline-${item.kind}-${item.timestamp}`,
					type: "generic",
					position: { x: 0, y: 0 },
					data: {
						category: "timeline",
						label: item.label || item.kind,
						icon: "",
						summary: item.body,
						fullText: getTimelineItemFullText(item),
					},
				}),
			);
			setBottomPanelMode("node");
		},
		[],
	);

	const handleOpenLogDirectory = useCallback(async () => {
		try {
			await OpenLogDirectory();
			setLogActionMessage("ログフォルダを開きました");
		} catch (openError) {
			setLogActionMessage(
				`ログフォルダを開けませんでした: ${String(openError)}`,
			);
		}
	}, []);

	const handleCopyLogPath = useCallback(async () => {
		try {
			const logPath = await getLogFilePath();
			await navigator.clipboard.writeText(logPath);
			setLogActionMessage("ログパスをコピーしました");
		} catch (copyError) {
			setLogActionMessage(
				`ログパスをコピーできませんでした: ${String(copyError)}`,
			);
		}
	}, []);

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
	const showTokenSplit =
		bottomPanelMode === "token" && boundTokenCounts.length > 0;

	const startResize = useCallback(
		(event: ReactPointerEvent<HTMLDivElement>) => {
			if (!splitContainerRef.current) {
				return;
			}
			const rect = splitContainerRef.current.getBoundingClientRect();
			dragStateRef.current = {
				startX: event.clientX,
				startRatio: splitRatio,
				width: rect.width,
			};
		},
		[splitRatio],
	);

	const handleResizerKeyDown = useCallback(
		(event: ReactKeyboardEvent<HTMLDivElement>) => {
			if (event.key === "ArrowLeft") {
				event.preventDefault();
				setSplitRatio((current) => Math.max(0.25, current - 0.03));
			}
			if (event.key === "ArrowRight") {
				event.preventDefault();
				setSplitRatio((current) => Math.min(0.75, current + 0.03));
			}
		},
		[],
	);

	return {
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
	};
}
