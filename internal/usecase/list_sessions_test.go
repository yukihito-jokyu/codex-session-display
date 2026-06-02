package usecase

import (
	"codex-session-display/internal/domain/dto"
	"context"
	"errors"
	"testing"
	"time"
)

type mockSessionRepository struct {
	sessions  []dto.SessionSummary
	err       error
	cacheRepo CacheRepository
}

func (m *mockSessionRepository) ListSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error) {
	if m.err != nil {
		return nil, m.err
	}

	var results []dto.SessionSummary
	for _, s := range m.sessions {
		if year > 0 && month > 0 && s.Timestamp != nil {
			t, err := time.Parse(time.RFC3339, *s.Timestamp)
			if err == nil && (t.Year() != year || int(t.Month()) != month) {
				continue
			}
		}

		sSummary := s
		if m.cacheRepo != nil {
			cached, err := m.cacheRepo.GetSessionSummary(ctx, s.ID)
			if err == nil && cached != nil {
				sSummary.Cwd = cached.Cwd
				sSummary.CliVersion = cached.CliVersion
				sSummary.Originator = cached.Originator
				sSummary.ModelProvider = cached.ModelProvider
				sSummary.Branch = cached.Branch
				sSummary.Source = cached.Source
				if cached.Timestamp != nil {
					sSummary.Timestamp = cached.Timestamp
				}
				sSummary.Parsed = true
			} else {
				sSummary.Parsed = false
			}
		}

		results = append(results, sSummary)
	}

	return results, nil
}

func (m *mockSessionRepository) GetSessionFilePath(ctx context.Context, sessionID string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSessionRepository) GetSessionIDByFilePath(ctx context.Context, filePath string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSessionRepository) GetSessionModTime(ctx context.Context, sessionID string) (time.Time, error) {
	return time.Time{}, errors.New("not implemented")
}

type mockCacheRepository struct {
	cache map[string]*dto.SessionSummary
	err   error
}

func (m *mockCacheRepository) GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	if s, ok := m.cache[sessionID]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCacheRepository) GetSessionDetail(ctx context.Context, sessionID string) (*dto.SessionDetailResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCacheRepository) GetSessionDetailModTime(ctx context.Context, sessionID string) (time.Time, error) {
	return time.Time{}, errors.New("not implemented")
}

func (m *mockCacheRepository) SaveSessionDetail(ctx context.Context, sessionID string, detail *dto.SessionDetailResponse) error {
	return nil
}

func TestNewListSessionsUseCase(t *testing.T) {
	sessionRepo := &mockSessionRepository{}
	cacheRepo := &mockCacheRepository{}

	tests := []struct {
		name        string
		sessionRepo SessionRepository
		cacheRepo   CacheRepository
	}{
		{
			name:        "success constructor",
			sessionRepo: sessionRepo,
			cacheRepo:   cacheRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewListSessionsUseCase(tt.sessionRepo, tt.cacheRepo)
			if uc.sessionRepo != tt.sessionRepo {
				t.Errorf("expected sessionRepo to be %v, got %v", tt.sessionRepo, uc.sessionRepo)
			}
			if uc.cacheRepo != tt.cacheRepo {
				t.Errorf("expected cacheRepo to be %v, got %v", tt.cacheRepo, uc.cacheRepo)
			}
		})
	}
}

func TestListSessionsUseCase_Execute(t *testing.T) {
	t12 := "2026-05-23T12:00:00Z"
	t13 := "2026-05-23T13:00:00Z"

	cwd1 := "/cwd1"
	cwd2 := "/cwd2"

	tests := []struct {
		name            string
		sessionSessions []dto.SessionSummary
		sessionErr      error
		cacheMap        map[string]*dto.SessionSummary
		cacheErr        error
		wantErr         bool
		verify          func(t *testing.T, results []dto.SessionSummary)
	}{
		{
			name:       "session repo error",
			sessionErr: errors.New("session repo error"),
			wantErr:    true,
		},
		{
			name: "successful execution with mixed cached, uncached, nil timestamp, and sorting",
			sessionSessions: []dto.SessionSummary{
				{ID: "session_nil_ts", Timestamp: nil},
				{ID: "session_12", Timestamp: &t12},
				{ID: "session_13", Timestamp: &t13},
			},
			cacheMap: map[string]*dto.SessionSummary{
				"session_nil_ts": {
					ID:        "session_nil_ts",
					Cwd:       &cwd1,
					Timestamp: nil, // キャッシュされたタイムスタンプが nil
				},
				"session_12": {
					ID:        "session_12",
					Cwd:       &cwd2,
					Timestamp: &t12, // キャッシュされたタイムスタンプが nil ではない
				},
				// session_13 はキャッシュエントリーなし（not found が返される）
			},
			wantErr: false,
			verify: func(t *testing.T, results []dto.SessionSummary) {
				if len(results) != 3 {
					t.Fatalf("expected 3 results, got %d", len(results))
				}

				// タイムスタンプの降順でソート: session_13 (13:00), session_12 (12:00), session_nil_ts (nil)
				if results[0].ID != "session_13" {
					t.Errorf("expected index 0 to be session_13, got %s", results[0].ID)
				}
				if results[1].ID != "session_12" {
					t.Errorf("expected index 1 to be session_12, got %s", results[1].ID)
				}
				if results[2].ID != "session_nil_ts" {
					t.Errorf("expected index 2 to be session_nil_ts, got %s", results[2].ID)
				}

				// session_nil_ts の検証（キャッシュあり、タイムスタンプは nil）
				if !results[2].Parsed {
					t.Errorf("expected session_nil_ts Parsed to be true")
				}
				if results[2].Cwd == nil || *results[2].Cwd != "/cwd1" {
					t.Errorf("expected Cwd to be /cwd1, got %v", results[2].Cwd)
				}
				if results[2].Timestamp != nil {
					t.Errorf("expected Timestamp to be nil, got %v", *results[2].Timestamp)
				}

				// session_12 の検証（キャッシュあり、タイムスタンプあり）
				if !results[1].Parsed {
					t.Errorf("expected session_12 Parsed to be true")
				}
				if results[1].Cwd == nil || *results[1].Cwd != "/cwd2" {
					t.Errorf("expected Cwd to be /cwd2, got %v", results[1].Cwd)
				}
				if results[1].Timestamp == nil || *results[1].Timestamp != t12 {
					t.Errorf("expected Timestamp to be %s, got %v", t12, results[1].Timestamp)
				}

				// session_13 の検証（キャッシュなし）
				if results[0].Parsed {
					t.Errorf("expected session_13 Parsed to be false")
				}
				if results[0].Cwd != nil {
					t.Errorf("expected session_13 Cwd to be nil, got %v", results[0].Cwd)
				}
			},
		},
		{
			name: "cache repository returns general error",
			sessionSessions: []dto.SessionSummary{
				{ID: "session_12", Timestamp: &t12},
			},
			cacheErr: errors.New("cache store error"),
			wantErr:  false,
			verify: func(t *testing.T, results []dto.SessionSummary) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Parsed {
					t.Errorf("expected Parsed to be false due to cache error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheRepo := &mockCacheRepository{
				cache: tt.cacheMap,
				err:   tt.cacheErr,
			}
			sessionRepo := &mockSessionRepository{
				sessions:  tt.sessionSessions,
				err:       tt.sessionErr,
				cacheRepo: cacheRepo,
			}

			uc := NewListSessionsUseCase(sessionRepo, cacheRepo)
			results, err := uc.Execute(context.Background(), "", 0, 0)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.verify != nil {
				tt.verify(t, results)
			}
		})
	}
}
