package usecase_test

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/usecase"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckUpdateUseCase_Execute_HasUpdate(t *testing.T) {
	// GitHub Releases API のモックレスポンス
	mockResponse := map[string]interface{}{
		"tag_name": "v1.1.0",
		"html_url": "https://github.com/owner/repo/releases/tag/v1.1.0",
		"assets": []map[string]interface{}{
			{
				"name":                 "codex-session-display.dmg",
				"browser_download_url": "https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.dmg",
			},
			{
				"name":                 "codex-session-display.zip",
				"browser_download_url": "https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.zip",
			},
		},
	}

	tests := []struct {
		name           string
		currentVersion string
		mockStatus     int
		mockBody       interface{}
		wantResult     *dto.UpdateResult
		wantErr        bool
	}{
		{
			name:           "最新バージョンが存在してアップデートがある場合",
			currentVersion: "1.0.0",
			mockStatus:     http.StatusOK,
			mockBody:       mockResponse,
			wantResult: &dto.UpdateResult{
				HasUpdate:   true,
				Current:     "1.0.0",
				Latest:      "1.1.0",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.1.0",
				DownloadURL: "https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.zip",
			},
			wantErr: false,
		},
		{
			name:           "最新バージョンが現在と同じ場合（アップデートなし）",
			currentVersion: "1.1.0",
			mockStatus:     http.StatusOK,
			mockBody:       mockResponse,
			wantResult: &dto.UpdateResult{
				HasUpdate:   false,
				Current:     "1.1.0",
				Latest:      "1.1.0",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.1.0",
				DownloadURL: "https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.zip",
			},
			wantErr: false,
		},
		{
			name:           "最新バージョンが現在より古い場合（アップデートなし）",
			currentVersion: "1.2.0",
			mockStatus:     http.StatusOK,
			mockBody:       mockResponse,
			wantResult: &dto.UpdateResult{
				HasUpdate:   false,
				Current:     "1.2.0",
				Latest:      "1.1.0",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.1.0",
				DownloadURL: "https://github.com/owner/repo/releases/download/v1.1.0/codex-session-display.zip",
			},
			wantErr: false,
		},
		{
			name:           "semverでの比較が正しく行われる場合（1.10.0 と 1.9.0）",
			currentVersion: "1.9.0",
			mockStatus:     http.StatusOK,
			mockBody: map[string]interface{}{
				"tag_name": "v1.10.0",
				"html_url": "https://github.com/owner/repo/releases/tag/v1.10.0",
				"assets": []map[string]interface{}{
					{
						"name":                 "codex-session-display.zip",
						"browser_download_url": "https://github.com/owner/repo/releases/download/v1.10.0/codex-session-display.zip",
					},
				},
			},
			wantResult: &dto.UpdateResult{
				HasUpdate:   true,
				Current:     "1.9.0",
				Latest:      "1.10.0",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.10.0",
				DownloadURL: "https://github.com/owner/repo/releases/download/v1.10.0/codex-session-display.zip",
			},
			wantErr: false,
		},
		{
			name:           "最新バージョンが文字列比較で大きくてもsemverで小さい場合（アップデートなし。1.10.0 -> 1.9.0）",
			currentVersion: "1.10.0",
			mockStatus:     http.StatusOK,
			mockBody: map[string]interface{}{
				"tag_name": "v1.9.0",
				"html_url": "https://github.com/owner/repo/releases/tag/v1.9.0",
				"assets": []map[string]interface{}{
					{
						"name":                 "codex-session-display.zip",
						"browser_download_url": "https://github.com/owner/repo/releases/download/v1.9.0/codex-session-display.zip",
					},
				},
			},
			wantResult: &dto.UpdateResult{
				HasUpdate:   false,
				Current:     "1.10.0",
				Latest:      "1.9.0",
				ReleaseURL:  "https://github.com/owner/repo/releases/tag/v1.9.0",
				DownloadURL: "https://github.com/owner/repo/releases/download/v1.9.0/codex-session-display.zip",
			},
			wantErr: false,
		},
		{
			name:           "APIエラーレスポンスの場合（エラーを返すこと）",
			currentVersion: "1.0.0",
			mockStatus:     http.StatusInternalServerError,
			mockBody:       nil,
			wantResult:     nil,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバーの起動
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				_ = json.NewEncoder(w).Encode(tt.mockBody)
			}))
			defer server.Close()

			uc := usecase.NewCheckUpdateUseCase(tt.currentVersion, server.URL)
			got, err := uc.Execute(context.Background())

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if got == nil {
					t.Fatal("Execute() returned nil result, but expected non-nil")
				}
				if got.HasUpdate != tt.wantResult.HasUpdate {
					t.Errorf("HasUpdate = %v, want %v", got.HasUpdate, tt.wantResult.HasUpdate)
				}
				if got.Current != tt.wantResult.Current {
					t.Errorf("Current = %q, want %q", got.Current, tt.wantResult.Current)
				}
				if got.Latest != tt.wantResult.Latest {
					t.Errorf("Latest = %q, want %q", got.Latest, tt.wantResult.Latest)
				}
				if got.ReleaseURL != tt.wantResult.ReleaseURL {
					t.Errorf("ReleaseURL = %q, want %q", got.ReleaseURL, tt.wantResult.ReleaseURL)
				}
				if got.DownloadURL != tt.wantResult.DownloadURL {
					t.Errorf("DownloadURL = %q, want %q", got.DownloadURL, tt.wantResult.DownloadURL)
				}
			}
		})
	}
}
