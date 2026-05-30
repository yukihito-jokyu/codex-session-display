package model

// Turn はユーザーの指示とAIエージェントの処理の一連の流れを表すドメインモデルです。
type Turn struct {
	Index                int
	TurnID               string
	TaskStarted          *EventMsgPayload
	TaskComplete         *EventMsgPayload
	Aborted              bool
	TurnContext          *TurnContextPayload
	Records              []*TypedRecord
	Batches              []*Batch
	DeveloperMessages    []*TypedRecord
	UserMessages         []*TypedRecord
	UserEventMsg         *TypedRecord
	AgentReasonings      []*ReasoningPair
	AgentMessages        []*TypedRecord
	ItemCompleted        []*TypedRecord
	TokenCounts          []*TokenCountWithBinding
	WebSearchRecords     []*TypedRecord
	ExternalEventRecords []*TypedRecord
	GenericRecords       []*TypedRecord
}

// ReasoningPair はエージェントの推論とその要約をペアリングしたドメインモデルです。
type ReasoningPair struct {
	AgentReasoning *TypedRecord
	Reasoning      *TypedRecord
}

// TokenCountWithBinding はトークン数情報とそれに関連付けられた直前のレコードを表すドメインモデルです。
type TokenCountWithBinding struct {
	Record        *TypedRecord
	BoundToRecord *TypedRecord
	BoundToNodeID string
	TurnIndex     int
}

// Batch はエージェントが複数のツール呼び出しをバッチ処理するパターンを表すドメインモデルです。
type Batch struct {
	CallRecords   []*TypedRecord
	OutputRecords []*TypedRecord
	MiddleMessage []*TypedRecord
}
