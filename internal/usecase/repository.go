package usecase

import (
	"codex-session-display/internal/domain/dto"
	"context"
)

// SessionRepository はセッションログをスキャンするためのインターフェースを定義します。
type SessionRepository interface {
	ListSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error)
}

// CacheRepository は解析されたセッションキャッシュの読み書きを行うためのインターフェースを定義します。
type CacheRepository interface {
	GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error)
}
