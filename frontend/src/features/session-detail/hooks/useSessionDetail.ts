import {
	type KeyboardEvent as ReactKeyboardEvent,
	type PointerEvent as ReactPointerEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import {
	GetSessionDetailByProvider,
	OpenLogDirectory,
} from "wailsjs/go/main/App";
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

export function useSessionDetail(
	id: string | undefined,
	provider: "codex" | "claude" = "codex",
) {
	const [sessionData, setSessionData] =
		useState<dto.SessionDetailResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [selectedNode, setSelectedNode] = useState<dto.FlowNode | null>(null);
	const [selectedFlowNodeId, setSelectedFlowNodeId] = useState<string | null>(
		null,
	);
	const [selectedTimelineId, setSelectedTimelineId] = useState<string | null>(
		null,
	);
	const [selectedTokenCountIndices, setSelectedTokenCountIndices] = useState<
		number[]
	>([]);
	const [timelineScrollTarget, setTimelineScrollTarget] = useState<{
		selectionId: string;
		timestamp: number;
	} | null>(null);
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

	const fetchSessionDetail = useCallback(
		(targetId: string, targetProvider: "codex" | "claude") => {
			setLoading(true);
			setError(null);
			GetSessionDetailByProvider(targetProvider, targetId)
				.then((data) => {
					setSessionData(data);
					setLoading(false);
				})
				.catch((err) => {
					console.error(err);
					setError(`Failed to fetch session detail: ${err.message || err}`);
					setLoading(false);
				});
		},
		[],
	);

	useEffect(() => {
		if (!id) {
			setSessionData(null);
			setError("Session ID is missing.");
			setLoading(false);
			return;
		}
		fetchSessionDetail(id, provider);
	}, [id, provider, fetchSessionDetail]);

	const retry = useCallback(() => {
		if (id) {
			fetchSessionDetail(id, provider);
		}
	}, [id, provider, fetchSessionDetail]);

	const timelineItems = useMemo(
		() =>
			(sessionData?.timeline ?? []).flatMap((turn) =>
				(turn.items ?? []).map((item) => item),
			),
		[sessionData?.timeline],
	);

	const clearSelection = useCallback(() => {
		setSelectedNode(null);
		setSelectedFlowNodeId(null);
		setSelectedTimelineId(null);
		setSelectedTokenCountIndices([]);
		setBottomPanelMode("node");
	}, []);

	const findTimelineItemByNodeId = useCallback(
		(nodeId: string) =>
			timelineItems.find(
				(item) =>
					item.node_id === nodeId || (item.node_ids ?? []).includes(nodeId),
			),
		[timelineItems],
	);

	const applyTimelineSelection = useCallback(
		(
			item: dto.ConversationTimelineItem | undefined,
			node: dto.FlowNode | null,
			mode: BottomPanelMode,
			scrollTimeline: boolean,
			zoomCanvas: boolean,
		) => {
			const selectionId = item?.selection_id ?? null;
			const flowNodeId = node?.id || item?.node_id || null;
			setSelectedNode(node);
			setSelectedFlowNodeId(flowNodeId);
			setSelectedTimelineId(selectionId);
			setSelectedTokenCountIndices(item?.token_count_indices ?? []);
			setBottomPanelMode(mode);
			if (selectionId && scrollTimeline) {
				setTimelineScrollTarget({
					selectionId,
					timestamp: Date.now(),
				});
			}
			if (flowNodeId && zoomCanvas) {
				setZoomTarget({ nodeId: flowNodeId, timestamp: Date.now() });
			}
		},
		[],
	);

	const handleTokenLogClick = useCallback(
		(tokenIndex: number) => {
			const item = timelineItems.find((candidate) =>
				(candidate.token_count_indices ?? []).includes(tokenIndex),
			);
			const tokenCount = sessionData?.token_counts?.find(
				(entry) => entry.index === tokenIndex,
			);
			const nodeId = item?.node_id || tokenCount?.bound_to_node_id;
			const node =
				sessionData?.nodes?.find((candidate) => candidate.id === nodeId) ??
				null;
			applyTimelineSelection(item, node, "token", true, true);
		},
		[
			applyTimelineSelection,
			sessionData?.nodes,
			sessionData?.token_counts,
			timelineItems,
		],
	);

	const handleCanvasNodeSelect = useCallback(
		(node: dto.FlowNode | null) => {
			if (!node) {
				clearSelection();
				return;
			}
			applyTimelineSelection(
				findTimelineItemByNodeId(node.id),
				node,
				"node",
				true,
				false,
			);
		},
		[applyTimelineSelection, clearSelection, findTimelineItemByNodeId],
	);

	const handleTokenBadgeClick = useCallback(
		(node: dto.FlowNode) => {
			applyTimelineSelection(
				findTimelineItemByNodeId(node.id),
				node,
				"token",
				true,
				false,
			);
		},
		[applyTimelineSelection, findTimelineItemByNodeId],
	);

	const handleTimelineSelect = useCallback(
		(item: dto.ConversationTimelineItem) => {
			if (item.selection_id && item.selection_id === selectedTimelineId) {
				clearSelection();
				return;
			}
			const node =
				sessionData?.nodes?.find(
					(candidate) => candidate.id === item.node_id,
				) ?? null;
			applyTimelineSelection(item, node, "node", false, true);
		},
		[
			applyTimelineSelection,
			clearSelection,
			selectedTimelineId,
			sessionData?.nodes,
		],
	);

	const handleTimelineFullText = useCallback(
		(item: dto.ConversationTimelineItem) => {
			setSelectedNode(
				dto.FlowNode.createFrom({
					id: item.selection_id || `timeline-${item.kind}-${item.timestamp}`,
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
			setSelectedFlowNodeId(item.node_id || null);
			setSelectedTimelineId(item.selection_id || null);
			setSelectedTokenCountIndices(item.token_count_indices ?? []);
			setBottomPanelMode("node");
			if (item.node_id) {
				setZoomTarget({ nodeId: item.node_id, timestamp: Date.now() });
			}
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
		if (!sessionData?.token_counts) {
			return [];
		}
		if (selectedTokenCountIndices.length > 0) {
			const selectedIndices = new Set(selectedTokenCountIndices);
			return sessionData.token_counts.filter((entry) =>
				selectedIndices.has(entry.index),
			);
		}
		if (!selectedFlowNodeId) {
			return [];
		}
		return sessionData.token_counts.filter(
			(entry) => entry.bound_to_node_id === selectedFlowNodeId,
		);
	}, [
		selectedFlowNodeId,
		selectedTokenCountIndices,
		sessionData?.token_counts,
	]);

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
	};
}
