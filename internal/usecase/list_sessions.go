package usecase

import (
	"context"
	"sort"

	"codex-session-display/internal/domain/dto"
)

// ListSessionsUseCase はセッション一覧を取得するユースケースを実装します。
type ListSessionsUseCase struct {
	sessionRepo SessionRepository
	cacheRepo   CacheRepository
}

// NewListSessionsUseCase は新しい ListSessionsUseCase を作成します。
func NewListSessionsUseCase(sessionRepo SessionRepository, cacheRepo CacheRepository) *ListSessionsUseCase {
	return &ListSessionsUseCase{
		sessionRepo: sessionRepo,
		cacheRepo:   cacheRepo,
	}
}

// Execute はセッションをスキャンし、指定された年月と検索クエリでフィルタリングして、それらをタイムスタンプの降順でソートして返します。
func (uc *ListSessionsUseCase) Execute(ctx context.Context, query string, year int, month int) ([]dto.SessionSummary, error) {
	summaries, err := uc.sessionRepo.ListSessions(ctx, year, month, query)
	if err != nil {
		return nil, err
	}

	// タイムスタンプの降順でソート
	sort.Slice(summaries, func(i, j int) bool {
		ti := ""
		if summaries[i].Timestamp != nil {
			ti = *summaries[i].Timestamp
		}
		tj := ""
		if summaries[j].Timestamp != nil {
			tj = *summaries[j].Timestamp
		}
		return ti > tj
	})

	return summaries, nil
}
