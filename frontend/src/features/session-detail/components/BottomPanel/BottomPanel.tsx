import type {
	KeyboardEventHandler,
	MutableRefObject,
	PointerEventHandler,
} from "react";
import type { dto } from "wailsjs/go/models";
import { NodeDetail } from "../NodeDetail/NodeDetail";
import { TokenDetail } from "../TokenDetail/TokenDetail";
import styles from "./BottomPanel.module.css";

type BottomPanelProps = {
	node: dto.FlowNode;
	showTokenSplit: boolean;
	splitRatio: number;
	splitContainerRef: MutableRefObject<HTMLDivElement | null>;
	boundTokenCounts: dto.TokenCountEntry[];
	latestBoundToken: dto.TokenCountEntry["total_token_usage"];
	onClose: () => void;
	onStartResize: PointerEventHandler<HTMLDivElement>;
	onResizerKeyDown: KeyboardEventHandler<HTMLDivElement>;
};

export function BottomPanel({
	node,
	showTokenSplit,
	splitRatio,
	splitContainerRef,
	boundTokenCounts,
	latestBoundToken,
	onClose,
	onStartResize,
	onResizerKeyDown,
}: BottomPanelProps) {
	return (
		<div className={styles.bottomPanel} data-testid="bottom-panel">
			<div className={styles.bottomPanelHeader}>
				<span className={styles.bottomPanelTitle}>
					Node Detail: {node.data?.label || node.id}
				</span>
				<button
					type="button"
					className={styles.panelCloseBtn}
					onClick={onClose}
				>
					✕
				</button>
			</div>
			<div
				ref={splitContainerRef}
				className={`${styles.bottomPanelContent} ${
					showTokenSplit ? styles.bottomPanelContentSplit : ""
				}`}
				data-testid={
					showTokenSplit ? "bottom-panel-split" : "bottom-panel-single"
				}
			>
				<NodeDetail
					key={node.id}
					node={node}
					splitRatio={showTokenSplit ? splitRatio : undefined}
				/>
				{showTokenSplit && (
					<>
						<div
							className={styles.panelResizer}
							data-testid="bottom-panel-resizer"
							onPointerDown={onStartResize}
							onKeyDown={onResizerKeyDown}
							role="separator"
							aria-orientation="vertical"
							aria-label="Resize token detail panel"
							aria-valuemin={25}
							aria-valuemax={75}
							aria-valuenow={Math.round(splitRatio * 100)}
							tabIndex={0}
						/>
						<TokenDetail
							entries={boundTokenCounts}
							latestUsage={latestBoundToken}
						/>
					</>
				)}
			</div>
		</div>
	);
}
