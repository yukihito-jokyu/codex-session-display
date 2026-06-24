package dto

import "codex-session-display/internal/domain/model"

// SessionSummary はセッション一覧内の単一のセッションエントリーを表します。
type SessionSummary struct {
	ID              string   `json:"id"`
	FilePath        string   `json:"file_path"`
	Cwd             *string  `json:"cwd"`
	CliVersion      *string  `json:"cli_version"`
	Originator      *string  `json:"originator"`
	ModelProvider   *string  `json:"model_provider"`
	Branch          *string  `json:"branch"`
	Source          *string  `json:"source"`
	Timestamp       *string  `json:"timestamp"`
	FileSize        int64    `json:"file_size"`
	FileModifiedAt  *string  `json:"file_modified_at"`
	Parsed          bool     `json:"parsed"`
	ParentSessionID *string  `json:"parent_session_id,omitempty"`
	ChildSessionIDs []string `json:"child_session_ids,omitempty"`
	TotalTokens     *int64   `json:"total_tokens,omitempty"`
	InputTokens     *int64   `json:"input_tokens,omitempty"`
	OutputTokens    *int64   `json:"output_tokens,omitempty"`
	ReasoningTokens *int64   `json:"reasoning_tokens,omitempty"`
	TurnCount       *int     `json:"turn_count,omitempty"`
	StepCount       *int     `json:"step_count,omitempty"`
	DurationMs      *int64   `json:"duration_ms,omitempty"`
}

// TokenBadgeData は React Flow のノード上に表示するトークンバッジのデータを表します。
type TokenBadgeData struct {
	ConsumedTokens  int64 `json:"consumedTokens"`
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
	ToolCallCount         int            `json:"tool_call_count"`
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
	Index              int                `json:"index"`
	TurnIndex          int                `json:"turn_index"`
	BoundToNodeID      string             `json:"bound_to_node_id"`
	ModelContextWindow int64              `json:"model_context_window"`
	LastTokenUsage     *model.TokenDetail `json:"last_token_usage"`
	TotalTokenUsage    *model.TokenDetail `json:"total_token_usage"`
}

// TimelineItemDetail はタイムライン項目を展開した際の補足情報を表します。
type TimelineItemDetail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ConversationTimelineItem はタイムライン内の単一の表示項目を表します。
type ConversationTimelineItem struct {
	SelectionID       string               `json:"selection_id"`
	NodeID            string               `json:"node_id"`
	NodeIDs           []string             `json:"node_ids"`
	TokenCountIndices []int                `json:"token_count_indices"`
	Kind              string               `json:"kind"`
	Label             string               `json:"label"`
	Role              string               `json:"role"`
	Body              string               `json:"body"`
	Timestamp         string               `json:"timestamp"`
	RecordCount       int                  `json:"record_count"`
	Collapsible       bool                 `json:"collapsible"`
	Details           []TimelineItemDetail `json:"details"`
	LastTokenUsage    TokenBreakdown       `json:"last_token_usage"`
	TokenCountCount   int                  `json:"token_count_count"`
	TotalTokenUsage   *TokenBreakdown      `json:"total_token_usage"`
}

// ConversationTimelineTurn は会話タイムライン内のターンを表します。
type ConversationTimelineTurn struct {
	Index          int                        `json:"index"`
	TurnID         string                     `json:"turn_id"`
	Pseudo         bool                       `json:"pseudo"`
	DurationMs     int64                      `json:"duration_ms"`
	ConsumedTokens TokenBreakdown             `json:"consumed_tokens"`
	Items          []ConversationTimelineItem `json:"items"`
}

// CurrentSessionDetailCacheSchemaVersion は現行のセッション詳細キャッシュ形式を表します。
const CurrentSessionDetailCacheSchemaVersion = 8

// SessionDetailResponse はフロントエンドに返され、ディスクにキャッシュされるセッション詳細を表します。
type SessionDetailResponse struct {
	ID                 string                     `json:"id"`
	CacheSchemaVersion int                        `json:"cache_schema_version"`
	ParsedAt           string                     `json:"parsed_at"`
	Nodes              []FlowNode                 `json:"nodes"`
	Edges              []FlowEdge                 `json:"edges"`
	Statistics         Statistics                 `json:"statistics"`
	TokenCounts        []TokenCountEntry          `json:"token_counts"`
	Timeline           []ConversationTimelineTurn `json:"timeline"`
	ParentSessionID    *string                    `json:"parent_session_id,omitempty"`
	ChildSessionIDs    []string                   `json:"child_session_ids,omitempty"`
	Subagents          []SubagentDetail           `json:"subagents,omitempty"`
}

// SubagentDetail は子セッション（サブエージェント）のトークン情報を表します。
type SubagentDetail struct {
	ID           string `json:"id"`
	Nickname     string `json:"nickname"`
	TotalTokens  int64  `json:"total_tokens"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TurnCount    *int   `json:"turn_count,omitempty"`
	StepCount    *int   `json:"step_count,omitempty"`
	DurationMs   *int64 `json:"duration_ms,omitempty"`
}
