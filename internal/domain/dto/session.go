package dto

import "codex-session-display/internal/domain/model"

// SessionSummary はセッション一覧内の単一のセッションエントリーを表します。
type SessionSummary struct {
	ID             string  `json:"id"`
	FilePath       string  `json:"file_path"`
	Cwd            *string `json:"cwd"`
	CliVersion     *string `json:"cli_version"`
	Originator     *string `json:"originator"`
	ModelProvider  *string `json:"model_provider"`
	Branch         *string `json:"branch"`
	Source         *string `json:"source"`
	Timestamp      *string `json:"timestamp"`
	FileSize       int64   `json:"file_size"`
	FileModifiedAt *string `json:"file_modified_at"`
	Parsed         bool    `json:"parsed"`
}

// TokenBadgeData は React Flow のノード上に表示するトークンバッジのデータを表します。
type TokenBadgeData struct {
	TotalTokens     int64 `json:"totalTokens"`
	TokenCountIndex int   `json:"tokenCountIndex"`
	BoundCount      int   `json:"boundCount"`
}

// FlowNode は React Flow 内のノードを表します。
type FlowNode struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
}

// Position は React Flow における座標を表します。
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeData は React Flow ノードのペイロードを表します。
type NodeData struct {
	Category   string                 `json:"category"`
	Label      string                 `json:"label"`
	Icon       string                 `json:"icon"`
	Summary    string                 `json:"summary"`
	FullText   string                 `json:"fullText,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
	BatchIndex int                    `json:"batchIndex,omitempty"`
	BatchSize  int                    `json:"batchSize,omitempty"`
	Collapsed  bool                   `json:"collapsed,omitempty"`
	TextLength int                    `json:"textLength,omitempty"`
	TurnIndex  int                    `json:"turnIndex,omitempty"`
	TokenBadge *TokenBadgeData        `json:"tokenBadge,omitempty"`
}

// FlowEdge は React Flow 内のエッジを表します。
type FlowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	Animated bool   `json:"animated"`
}

// TokenBreakdown はトークン消費の内訳を表します。
type TokenBreakdown struct {
	TotalTokens           int64 `json:"total_tokens"`
	InputTokens           int64 `json:"input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
}

// TurnStatistics はターンごとの統計情報を表します。
type TurnStatistics struct {
	Index                 int            `json:"index"`
	CollaborationModeKind string         `json:"collaboration_mode_kind"`
	DurationMs            int64          `json:"duration_ms"`
	TimeToFirstTokenMs    int64          `json:"time_to_first_token_ms"`
	TokenCountCount       int            `json:"token_count_count"`
	ConsumedTokens        TokenBreakdown `json:"consumed_tokens"`
}

// Statistics は SessionDetailResponse の統計情報です。
type Statistics struct {
	DurationMs        int64            `json:"duration_ms"`
	TotalTokens       int64            `json:"total_tokens"`
	ToolCallCount     int              `json:"tool_call_count"`
	TokenCountCount   int              `json:"token_count_count"`
	ContextWindowSize int64            `json:"context_window_size"`
	TurnCount         int              `json:"turn_count"`
	Turns             []TurnStatistics `json:"turns"`
}

// TokenCountEntry は詳細画面で管理されるトークンカウント情報を表します。
type TokenCountEntry struct {
	Index           int                `json:"index"`
	TurnIndex       int                `json:"turn_index"`
	BoundToNodeID   string             `json:"bound_to_node_id"`
	LastTokenUsage  *model.TokenDetail `json:"last_token_usage"`
	TotalTokenUsage *model.TokenDetail `json:"total_token_usage"`
}

// SessionDetailResponse はフロントエンドに返され、ディスクにキャッシュされるセッション詳細を表します。
type SessionDetailResponse struct {
	ID          string            `json:"id"`
	ParsedAt    string            `json:"parsed_at"`
	Nodes       []FlowNode        `json:"nodes"`
	Edges       []FlowEdge        `json:"edges"`
	Statistics  Statistics        `json:"statistics"`
	TokenCounts []TokenCountEntry `json:"token_counts"`
}
