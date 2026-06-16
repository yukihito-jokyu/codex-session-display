package model

import (
	"encoding/json"
)

// RecordEnvelope は JSONL レコードのエンベロープ（外枠）を表します。
type RecordEnvelope struct {
	Type      string          `json:"type"`
	Timestamp interface{}     `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// SessionMetaPayload は session_meta レコードのペイロードです。
type SessionMetaPayload struct {
	ID               string        `json:"id"`
	Cwd              string        `json:"cwd"`
	CliVersion       string        `json:"cli_version"`
	Originator       string        `json:"originator"`
	Source           string        `json:"source"`
	ThreadSource     string        `json:"thread_source"`
	ParentThreadID   string        `json:"parent_thread_id,omitempty"`
	ModelProvider    string        `json:"model_provider"`
	BaseInstructions *Instructions `json:"base_instructions"` // システム指示
	Git              *GitInfo      `json:"git"`
	Timestamp        string        `json:"timestamp"`     // ペイロード内
	TopTimestamp     string        `json:"top_timestamp"` // レコード書き込み
}

type Instructions struct {
	Text string `json:"text"`
}

type GitInfo struct {
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	URL    string `json:"url"`
}

// TurnContextPayload は turn_context レコードのペイロードです。
type TurnContextPayload struct {
	TurnID            string             `json:"turn_id"`
	Model             string             `json:"model"`
	ApprovalPolicy    string             `json:"approval_policy"`
	CollaborationMode *CollaborationMode `json:"collaboration_mode"`
	UserInstructions  string             `json:"user_instructions"`
	Effort            string             `json:"effort"`
	Personality       string             `json:"personality"`
}

type CollaborationMode struct {
	Mode                  string `json:"mode"`
	DeveloperInstructions string `json:"developer_instructions"`
}

// EventMsgPayload は event_msg レコードのペイロードです。
type EventMsgPayload struct {
	Type                  string            `json:"type"` // サブタイプ (task_started, task_complete 等)
	Message               string            `json:"message,omitempty"`
	Text                  string            `json:"text,omitempty"`
	TurnID                string            `json:"turn_id,omitempty"`
	StartedAt             int64             `json:"started_at,omitempty"`
	ModelContextWindow    int64             `json:"model_context_window,omitempty"`
	CollaborationModeKind string            `json:"collaboration_mode_kind,omitempty"`
	CompletedAt           int64             `json:"completed_at,omitempty"`
	DurationMs            int64             `json:"duration_ms,omitempty"`
	TimeToFirstTokenMs    int64             `json:"time_to_first_token_ms,omitempty"`
	LastAgentMessage      string            `json:"last_agent_message,omitempty"`
	Reason                string            `json:"reason,omitempty"`
	Item                  *CompletedItem    `json:"item,omitempty"`
	CompletedAtMs         int64             `json:"completed_at_ms,omitempty"`
	ThreadID              string            `json:"thread_id,omitempty"`
	ThreadName            string            `json:"thread_name,omitempty"`
	CallID                string            `json:"call_id,omitempty"`
	Command               []string          `json:"command,omitempty"`
	Cwd                   string            `json:"cwd,omitempty"`
	ExitCode              *int              `json:"exit_code,omitempty"`
	Duration              *CommandDuration  `json:"duration,omitempty"`
	AggregatedOutput      string            `json:"aggregated_output,omitempty"`
	ProcessID             int               `json:"process_id,omitempty"`
	CodexErrorInfo        *ErrorInfo        `json:"codex_error_info,omitempty"`
	Query                 string            `json:"query,omitempty"`
	Action                *SearchAction     `json:"action,omitempty"`
	SenderThreadID        string            `json:"sender_thread_id,omitempty"`
	NewThreadID           string            `json:"new_thread_id,omitempty"`
	NewAgentNickname      string            `json:"new_agent_nickname,omitempty"`
	NewAgentRole          string            `json:"new_agent_role,omitempty"`
	Prompt                string            `json:"prompt,omitempty"`
	Model                 string            `json:"model,omitempty"`
	ReasoningEffort       string            `json:"reasoning_effort,omitempty"`
	Status                string            `json:"status,omitempty"`
	ReceiverThreadID      string            `json:"receiver_thread_id,omitempty"`
	ReceiverAgentNickname string            `json:"receiver_agent_nickname,omitempty"`
	Statuses              map[string]string `json:"statuses,omitempty"`
	Invocation            *MCPInvocation    `json:"invocation,omitempty"`
	Result                string            `json:"result,omitempty"`
	Path                  string            `json:"path,omitempty"`
	Info                  *TokenInfo        `json:"info,omitempty"` // token_count 情報
}

type CompletedItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type CommandDuration struct {
	Secs  int64 `json:"secs"`
	Nanos int64 `json:"nanos"`
}

type ErrorInfo struct {
	Type string `json:"type"`
}

type SearchAction struct {
	Type    string   `json:"type"`
	Query   string   `json:"query"`
	Queries []string `json:"queries"`
}

type MCPInvocation struct {
	Server    string          `json:"server"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

type TokenDetail struct {
	TotalTokens           int64 `json:"total_tokens"`
	InputTokens           int64 `json:"input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
}

type TokenInfo struct {
	TotalTokenUsage    *TokenDetail `json:"total_token_usage"`
	LastTokenUsage     *TokenDetail `json:"last_token_usage"`
	ModelContextWindow int64        `json:"model_context_window"`
}

// ResponseItemPayload は response_item レコードのペイロードです。
type ResponseItemPayload struct {
	Type             string           `json:"type"` // サブタイプ (message, reasoning, function_call 等)
	Role             string           `json:"role,omitempty"`
	Content          []MessageContent `json:"content,omitempty"`
	Summary          []MessageContent `json:"summary,omitempty"` // reasoning のサマリー
	EncryptedContent string           `json:"encrypted_content,omitempty"`
	Name             string           `json:"name,omitempty"`
	Arguments        string           `json:"arguments,omitempty"`
	Input            string           `json:"input,omitempty"`
	CallID           string           `json:"call_id,omitempty"`
	Output           string           `json:"output,omitempty"`
	Action           *SearchAction    `json:"action,omitempty"`
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TypedRecord はパースされた1レコードのラッパーです。
type TypedRecord struct {
	LineNumber   int                  `json:"line_number"`
	Type         string               `json:"type"`     // session_meta, turn_context, event_msg, response_item
	SubType      string               `json:"sub_type"` // event_msg や response_item の payload.type
	Raw          string               `json:"raw"`      // 元の行の生JSON文字列
	Envelope     RecordEnvelope       `json:"envelope"`
	SessionMeta  *SessionMetaPayload  `json:"session_meta,omitempty"`
	TurnContext  *TurnContextPayload  `json:"turn_context,omitempty"`
	EventMsg     *EventMsgPayload     `json:"event_msg,omitempty"`
	ResponseItem *ResponseItemPayload `json:"response_item,omitempty"`
}
