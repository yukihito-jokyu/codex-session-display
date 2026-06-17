package usecase

import (
	"codex-session-display/internal/domain/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// ErrAPICallFailed はGitHub APIの呼び出しに失敗した場合のエラーです。
var ErrAPICallFailed = errors.New("api call failed")

// CheckUpdateUseCase はアプリケーションのアップデートを確認するユースケースです。
type CheckUpdateUseCase struct {
	currentVersion string
	apiURL         string
	httpClient     *http.Client
}

// NewCheckUpdateUseCase は CheckUpdateUseCase を作成します。
func NewCheckUpdateUseCase(currentVersion, apiURL string) *CheckUpdateUseCase {
	return &CheckUpdateUseCase{
		currentVersion: currentVersion,
		apiURL:         apiURL,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

type gitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Execute はアップデートチェックを実行します。
func (uc *CheckUpdateUseCase) Execute(ctx context.Context) (*dto.UpdateResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uc.apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "codex-session-display")

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrAPICallFailed, resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(uc.currentVersion, "v")

	downloadURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".zip") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return &dto.UpdateResult{
		HasUpdate:   isNewer(latest, current),
		Current:     current,
		Latest:      latest,
		ReleaseURL:  release.HTMLURL,
		DownloadURL: downloadURL,
	}, nil
}

func isNewer(latest, current string) bool {
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}

	return semver.IsValid(latest) &&
		semver.IsValid(current) &&
		semver.Compare(latest, current) > 0
}
