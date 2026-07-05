package dto

// AnalyzeOptions はコーパス分析の実行オプションを表します。
type AnalyzeOptions struct {
	ProjectSource string `json:"projectSource"` // "config" または "home"
}

// ParseErrorInfo はパースエラーの情報を表します。
type ParseErrorInfo struct {
	FileID string `json:"fileId"` // 匿名化されたファイルID (ハッシュ値等)
	Count  int64  `json:"count"`  // パースエラー行数
}

// TypeCount は各型の出現回数を表します。
type TypeCount map[string]int64

// PrivacyMetrics はプライバシー関連の指標（生データを出さず、hash、長さ、bucket、頻度で表す）を表します。
type PrivacyMetrics struct {
	TextLengthDist     map[string]int64 `json:"textLengthDist"`     // textの長さのバケット分布 (例: "0": count, "1-100": count, ...)
	ThinkingLengthDist map[string]int64 `json:"thinkingLengthDist"` // thinkingの長さのバケット分布
	CommandHashDist    map[string]int64 `json:"commandHashDist"`    // 実行されたコマンドのSHA256ハッシュ -> 頻度 (本文は返さない)
	ToolOutputDist     map[string]int64 `json:"toolOutputDist"`     // ツール出力のSHA256ハッシュ -> 頻度 (本文は返さない)
}

// AnalyzeResult はコーパス分析結果を表します。
type AnalyzeResult struct {
	TotalFiles     int                  `json:"totalFiles"`
	TotalLines     int64                `json:"totalLines"`
	ParseErrors    []ParseErrorInfo     `json:"parseErrors"`
	FieldPaths     map[string]TypeCount `json:"fieldPaths"`   // field path -> {type -> count}
	ContentTypes   map[string]int64     `json:"contentTypes"` // content type (text, thinking, tool_use, etc.) -> count
	ToolNames      map[string]int64     `json:"toolNames"`    // tool name -> count
	UsageKeys      map[string]int64     `json:"usageKeys"`    // usage key (input_tokens, etc.) -> count
	PrivacyMetrics PrivacyMetrics       `json:"privacyMetrics"`
}
