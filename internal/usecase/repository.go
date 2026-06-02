package usecase

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"context"
	"time"
)

// SessionParser はセッションログファイルをパースするためのインターフェースを定義します。
type SessionParser interface {
	ParseSessionFile(ctx context.Context, filePath string) ([]*model.TypedRecord, error)
}

// SessionRepository はセッションログをスキャンするためのインターフェースを定義します。
type SessionRepository interface {
	ListSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error)
	GetSessionFilePath(ctx context.Context, sessionID string) (string, error)
	GetSessionIDByFilePath(ctx context.Context, filePath string) (string, error)
	GetSessionModTime(ctx context.Context, sessionID string) (time.Time, error)
}

// CacheRepository は解析されたセッションキャッシュの読み書きを行うためのインターフェースを定義します。
type CacheRepository interface {
	GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error)
	GetSessionDetail(ctx context.Context, sessionID string) (*dto.SessionDetailResponse, error)
	GetSessionDetailModTime(ctx context.Context, sessionID string) (time.Time, error)
	SaveSessionDetail(ctx context.Context, sessionID string, detail *dto.SessionDetailResponse) error
}
