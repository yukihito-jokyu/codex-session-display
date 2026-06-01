import {
	Background,
	BackgroundVariant,
	Controls,
	type Node,
	ReactFlow,
	ReactFlowProvider,
	useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useEffect, useMemo } from "react";
import type { dto } from "wailsjs/go/models";
import { nodeTypes } from "./CustomNodes";
import styles from "./FlowCanvas.module.css";

interface FlowCanvasProps {
	nodes: dto.FlowNode[];
	edges: dto.FlowEdge[];
	onNodeSelect: (node: dto.FlowNode | null) => void;
	selectedNodeId?: string;
	selectedNodeType?: string;
	zoomTarget?: { nodeId: string; timestamp: number } | null;
}

function FlowCanvasInner({
	nodes,
	edges,
	onNodeSelect,
	selectedNodeId,
	selectedNodeType,
	zoomTarget,
}: FlowCanvasProps) {
	const { fitView } = useReactFlow();

	// React Flowのノード表現に合わせ、選択状態をマッピング
	const flowNodes = useMemo(() => {
		return nodes.map((node) => ({
			...node,
			selected: node.id === selectedNodeId,
		})) as unknown as Node[];
	}, [nodes, selectedNodeId]);

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
		<div className={styles.canvasContainer}>
			<ReactFlow
				nodes={flowNodes}
				edges={edges}
				nodeTypes={nodeTypes}
				onNodeClick={onNodeClick}
				onPaneClick={onPaneClick}
				nodesDraggable={false}
				nodesConnectable={false}
				elementsSelectable={true}
				minZoom={0.1}
				maxZoom={2.0}
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
