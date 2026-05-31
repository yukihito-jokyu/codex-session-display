import {
	Background,
	BackgroundVariant,
	Controls,
	type Node,
	ReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useMemo } from "react";
import type { dto } from "wailsjs/go/models";
import { nodeTypes } from "./CustomNodes";
import styles from "./FlowCanvas.module.css";

interface FlowCanvasProps {
	nodes: dto.FlowNode[];
	edges: dto.FlowEdge[];
	onNodeSelect?: (node: dto.FlowNode | null) => void;
	selectedNodeId?: string;
}

export function FlowCanvas({
	nodes,
	edges,
	onNodeSelect,
	selectedNodeId,
}: FlowCanvasProps) {
	// React Flowのノード表現に合わせ、選択状態をマッピング
	const flowNodes = useMemo(() => {
		return nodes.map((node) => ({
			...node,
			selected: node.id === selectedNodeId,
		})) as unknown as Node[];
	}, [nodes, selectedNodeId]);

	const onNodeClick = (_event: React.MouseEvent, node: Node) => {
		if (onNodeSelect) {
			const originalNode = nodes.find((n) => n.id === node.id);
			onNodeSelect(originalNode || (node as unknown as dto.FlowNode));
		}
	};

	const onPaneClick = () => {
		if (onNodeSelect) {
			onNodeSelect(null);
		}
	};

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
export default FlowCanvas;
