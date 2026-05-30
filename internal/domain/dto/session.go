package dto

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
}

// FlowEdge は React Flow 内のエッジを表します。
type FlowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	Animated bool   `json:"animated"`
}

// Statistics は SessionDetailResponse の統計情報のプレースホルダーです。
type Statistics struct {
	DurationMs      int64 `json:"duration_ms"`
	TotalTokens     int64 `json:"total_tokens"`
	ToolCallCount   int   `json:"tool_call_count"`
	TokenCountCount int   `json:"token_count_count"`
	TurnCount       int   `json:"turn_count"`
}

// SessionDetailResponse はフロントエンドに返され、ディスクにキャッシュされるセッション詳細を表します。
type SessionDetailResponse struct {
	ID          string        `json:"id"`
	ParsedAt    string        `json:"parsed_at"`
	Nodes       []FlowNode    `json:"nodes"`
	Edges       []FlowEdge    `json:"edges"`
	Statistics  Statistics    `json:"statistics"`
	TokenCounts []interface{} `json:"token_counts"` // プレースホルダー
}
