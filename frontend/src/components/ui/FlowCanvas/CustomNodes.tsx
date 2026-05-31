import { Handle, type NodeProps, Position } from "@xyflow/react";
import { useState } from "react";
import styles from "./FlowCanvas.module.css";

export type CustomNodeProps = NodeProps & {
	data: {
		category: string;
		label: string;
		icon: string;
		summary: string;
		fullText?: string;
		meta?: Record<string, unknown>;
		batchIndex?: number;
		batchSize?: number;
		collapsed?: boolean;
		textLength?: number;
		turnIndex?: number;
		tokenBadge?: {
			totalTokens: number;
			tokenCountIndex: number;
			boundCount: number;
		};
	};
};

const getCategoryClass = (type: string, data: CustomNodeProps["data"]) => {
	// 孤立警告ノードの判定
	if (data.icon === "⚠️" || data.label?.includes("Orphan")) {
		return styles.nodeWarning;
	}

	switch (type) {
		case "sessionMeta":
			return styles.nodeMeta;
		case "turnContext":
			return styles.nodeTurnCtx;
		case "contextDoc":
			return styles.nodeContextDoc;
		case "developerMessage":
			return styles.nodeDeveloperMessage;
		case "userMessage":
			return styles.nodeUserMessage;
		case "userApiMessage":
			return styles.nodeUserApiMessage;
		case "agentMessage":
			return styles.nodeAgentMessage;
		case "reasoning":
			return styles.nodeReasoning;
		case "action":
			return styles.nodeAction;
		case "webSearchAction":
			return styles.nodeWebSearch;
		case "externalEvent":
			return styles.nodeExternalEvent;
		case "taskEvent":
			if (data.label?.includes("Started")) return styles.nodeTurnStart;
			if (data.label?.includes("Complete") || data.label?.includes("Aborted"))
				return styles.nodeTurnEnd;
			return styles.nodeTurnStart;
		case "itemCompleted":
			return styles.nodeItemCompleted;
		default:
			return styles.nodeGeneric;
	}
};

export const BaseCustomNode = ({ type, data, selected }: CustomNodeProps) => {
	const isOutOfTurn = data.turnIndex === -1;
	const isWarning = data.icon === "⚠️" || data.label?.includes("Orphan");

	let displayLabel = data.label;
	if (isOutOfTurn && !displayLabel.startsWith("[System]")) {
		displayLabel = `[System] ${displayLabel}`;
	}

	const classNames = [
		styles.customNode,
		getCategoryClass(type, data),
		isOutOfTurn ? styles.outOfTurn : "",
		isOutOfTurn ? "node-out-of-turn" : "",
		isWarning ? styles.nodeWarning : "",
		isWarning ? "node-warning" : "",
		selected ? styles.selected : "",
	]
		.filter(Boolean)
		.join(" ");

	// トークンバッジ情報のフォーマット
	const tokenBadge = data.tokenBadge;
	const formatTokens = (tokens: number): string => {
		if (tokens >= 1000000) {
			const val = tokens / 1000000;
			return `${val.toFixed(val % 1 === 0 ? 0 : 1)}M`;
		}
		if (tokens >= 1000) {
			const val = tokens / 1000;
			return `${val.toFixed(val % 1 === 0 ? 0 : 1)}K`;
		}
		return tokens.toString();
	};
	const totalTokensFormatted = tokenBadge
		? formatTokens(tokenBadge.totalTokens)
		: null;

	return (
		<div className={classNames}>
			<Handle type="target" position={Position.Top} id="t" />
			<Handle type="target" position={Position.Left} id="l" />

			<div className={styles.nodeHeader}>
				<div className={styles.headerLeft}>
					<span className={styles.nodeIcon}>
						{isWarning ? "⚠️" : data.icon || "📄"}
					</span>
					<span className={styles.nodeLabel}>{displayLabel}</span>
				</div>
				{tokenBadge && totalTokensFormatted && (
					<span className={styles.tokenBadge}>
						{totalTokensFormatted}
						{tokenBadge.boundCount >= 2 ? ` ×${tokenBadge.boundCount}` : ""}
					</span>
				)}
			</div>

			<div className={styles.nodeBody}>{data.summary}</div>

			<Handle type="source" position={Position.Bottom} id="b" />
			<Handle type="source" position={Position.Right} id="r" />
		</div>
	);
};

export const ContextDocNode = ({ data, selected }: CustomNodeProps) => {
	const [isExpanded, setIsExpanded] = useState(false);

	const isOutOfTurn = data.turnIndex === -1;
	const isWarning = data.icon === "⚠️" || data.label?.includes("Orphan");

	let displayLabel = data.label;
	if (isOutOfTurn && !displayLabel.startsWith("[System]")) {
		displayLabel = `[System] ${displayLabel}`;
	}

	const classNames = [
		styles.customNode,
		styles.nodeContextDoc,
		isOutOfTurn ? styles.outOfTurn : "",
		isOutOfTurn ? "node-out-of-turn" : "",
		isWarning ? styles.nodeWarning : "",
		isWarning ? "node-warning" : "",
		selected ? styles.selected : "",
	]
		.filter(Boolean)
		.join(" ");

	const textLength =
		data.textLength || (data.fullText ? data.fullText.length : 0);
	const lengthDisplay =
		textLength > 1000 ? `${(textLength / 1000).toFixed(0)}K` : textLength;

	return (
		<div className={classNames}>
			<Handle type="target" position={Position.Top} id="t" />
			<Handle type="target" position={Position.Left} id="l" />

			<div
				className={styles.nodeHeader}
				onClick={() => setIsExpanded(!isExpanded)}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.key === " ") setIsExpanded(!isExpanded);
				}}
				tabIndex={0}
				style={{ cursor: "pointer", userSelect: "none" }}
				role="button"
			>
				<div className={styles.headerLeft}>
					<span className={styles.nodeIcon}>
						{isWarning ? "⚠️" : data.icon || "📄"}
					</span>
					<span className={styles.nodeLabel}>
						{isExpanded ? `▾ ${data.label}` : displayLabel}
					</span>
				</div>
			</div>

			<div className={styles.nodeBody}>
				{!isExpanded ? (
					<div
						className={styles.collapsedDoc}
						onClick={() => setIsExpanded(true)}
						onKeyDown={(e) => {
							if (e.key === "Enter" || e.key === " ") setIsExpanded(true);
						}}
						tabIndex={0}
						role="button"
					>
						<span>▸ クリックして展開</span>
						{textLength > 0 && <span>({lengthDisplay})</span>}
					</div>
				) : (
					<div className={styles.expandedDoc}>
						<div
							className={styles.expandedDocTitle}
							onClick={() => setIsExpanded(false)}
							onKeyDown={(e) => {
								if (e.key === "Enter" || e.key === " ") setIsExpanded(false);
							}}
							tabIndex={0}
							role="button"
						>
							<span>▾ 折りたたむ</span>
						</div>
						<div className={styles.expandedDocContent}>{data.fullText}</div>
					</div>
				)}
			</div>

			<Handle type="source" position={Position.Bottom} id="b" />
			<Handle type="source" position={Position.Right} id="r" />
		</div>
	);
};

export const nodeTypes = {
	sessionMeta: BaseCustomNode,
	taskEvent: BaseCustomNode,
	turnContext: BaseCustomNode,
	contextDoc: ContextDocNode,
	developerMessage: BaseCustomNode,
	userApiMessage: BaseCustomNode,
	userMessage: BaseCustomNode,
	agentMessage: BaseCustomNode,
	reasoning: BaseCustomNode,
	action: BaseCustomNode,
	webSearchAction: BaseCustomNode,
	externalEvent: BaseCustomNode,
	generic: BaseCustomNode,
};
