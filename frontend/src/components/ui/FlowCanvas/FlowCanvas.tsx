import {
	Background,
	BackgroundVariant,
	Controls,
	type Node,
	Position,
	ReactFlow,
	ReactFlowProvider,
	useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useCallback, useEffect, useMemo } from "react";
import type { dto } from "wailsjs/go/models";
import { nodeTypes } from "./CustomNodes";
import styles from "./FlowCanvas.module.css";

const CUSTOM_NODE_INITIAL_WIDTH = 320;
const CUSTOM_NODE_INITIAL_HEIGHT = 80;
const CUSTOM_NODE_INITIAL_HANDLES: NonNullable<Node["handles"]> = [
	{
		id: "t",
		type: "target",
		position: Position.Top,
		x: CUSTOM_NODE_INITIAL_WIDTH / 2,
		y: 0,
	},
	{
		id: "l",
		type: "target",
		position: Position.Left,
		x: 0,
		y: CUSTOM_NODE_INITIAL_HEIGHT / 2,
	},
	{
		id: "b",
		type: "source",
		position: Position.Bottom,
		x: CUSTOM_NODE_INITIAL_WIDTH / 2,
		y: CUSTOM_NODE_INITIAL_HEIGHT,
	},
	{
		id: "r",
		type: "source",
		position: Position.Right,
		x: CUSTOM_NODE_INITIAL_WIDTH,
		y: CUSTOM_NODE_INITIAL_HEIGHT / 2,
	},
];

interface FlowCanvasProps {
	nodes: dto.FlowNode[];
	edges: dto.FlowEdge[];
	onNodeSelect: (node: dto.FlowNode | null) => void;
	onTokenBadgeClick?: (node: dto.FlowNode) => void;
	interactionLocked?: boolean;
	selectedNodeId?: string;
	selectedNodeType?: string;
	zoomTarget?: { nodeId: string; timestamp: number } | null;
}

function FlowCanvasInner({
	nodes,
	edges,
	onNodeSelect,
	onTokenBadgeClick,
	interactionLocked,
	selectedNodeId,
	selectedNodeType,
	zoomTarget,
}: FlowCanvasProps) {
	const { fitView } = useReactFlow();

	const handleTokenBadgeClick = useCallback(
		(nodeId: string) => {
			const node = nodes.find((candidate) => candidate.id === nodeId);
			if (node) {
				onTokenBadgeClick?.(node);
			}
		},
		[nodes, onTokenBadgeClick],
	);

	const nodeDataById = useMemo(() => {
		return new Map(
			nodes.map((node) => [
				node.id,
				{
					...node.data,
					onTokenBadgeClick: handleTokenBadgeClick,
				},
			]),
		);
	}, [handleTokenBadgeClick, nodes]);

	// React Flowのノード表現に合わせ、選択状態をマッピング
	const flowNodes = useMemo(() => {
		return nodes.map((node) => ({
			...node,
			className: [
				interactionLocked && node.id !== selectedNodeId
					? "node-pass-through"
					: undefined,
				interactionLocked && node.id === selectedNodeId
					? "node-selected-interactive"
					: undefined,
			]
				.filter(Boolean)
				.join(" "),
			data: nodeDataById.get(node.id),
			initialWidth: CUSTOM_NODE_INITIAL_WIDTH,
			initialHeight: CUSTOM_NODE_INITIAL_HEIGHT,
			handles: CUSTOM_NODE_INITIAL_HANDLES,
			selected: node.id === selectedNodeId,
		})) as unknown as Node[];
	}, [interactionLocked, nodeDataById, nodes, selectedNodeId]);

	const onNodeClick = (_event: React.MouseEvent, node: Node) => {
		if (
			selectedNodeId &&
			selectedNodeType !== "contextDoc" &&
			node.id !== selectedNodeId
		) {
			onNodeSelect(null);
			return;
		}

		const originalNode = nodes.find((n) => n.id === node.id);
		onNodeSelect(originalNode as dto.FlowNode);
	};

	const onPaneClick = () => {
		onNodeSelect(null);
	};

	// zoomTargetが更新されたら、該当ノードへズームインする
	useEffect(() => {
		if (zoomTarget) {
			fitView({
				nodes: [{ id: zoomTarget.nodeId }],
				duration: 800,
				maxZoom: 1.2, // 適度なズーム倍率に制限
			});
		}
	}, [zoomTarget, fitView]);

	return (
		<div
			className={`${styles.canvasContainer} ${
				interactionLocked ? styles.interactionLocked : ""
			}`}
			onClickCapture={(event) => {
				if (!interactionLocked) {
					return;
				}
				const target = event.target;
				if (!(target instanceof HTMLElement)) {
					return;
				}
				if (target.closest(".react-flow__controls")) {
					return;
				}
				if (target.closest(".node-selected-interactive")) {
					return;
				}
				event.preventDefault();
				event.stopPropagation();
				onNodeSelect(null);
			}}
		>
			<ReactFlow
				nodes={flowNodes}
				edges={edges}
				nodeTypes={nodeTypes}
				onNodeClick={onNodeClick}
				onPaneClick={onPaneClick}
				nodesDraggable={false}
				nodesConnectable={false}
				elementsSelectable={false}
				panOnDrag={!interactionLocked}
				zoomOnScroll={!interactionLocked}
				zoomOnPinch={!interactionLocked}
				zoomOnDoubleClick={!interactionLocked}
				minZoom={0.1}
				maxZoom={2.0}
				onlyRenderVisibleElements
				fitView
				fitViewOptions={{ padding: 0.2 }}
			>
				<Background
					variant={BackgroundVariant.Dots}
					gap={16}
					size={1.5}
					color="var(--grid-dot)"
				/>
				<Controls showInteractive={false} />
			</ReactFlow>
		</div>
	);
}

export function FlowCanvas(props: FlowCanvasProps) {
	return (
		<ReactFlowProvider>
			<FlowCanvasInner {...props} />
		</ReactFlowProvider>
	);
}

export default FlowCanvas;
