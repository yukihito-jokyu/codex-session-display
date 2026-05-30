package usecase

import (
	"context"

	"codex-session-display/internal/domain/dto"
)

// SessionRepository はセッションログをスキャンするためのインターフェースを定義します。
type SessionRepository interface {
	ListSessions(ctx context.Context, year int, month int, query string) ([]dto.SessionSummary, error)
}

// CacheRepository は解析されたセッションキャッシュの読み書きを行うためのインターフェースを定義します。
type CacheRepository interface {
	GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error)
}
