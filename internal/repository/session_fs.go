package repository

import (
	"bufio"
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/usecase"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	parseSessionFilenameFn = parseSessionFilename
	filepathRelFn          = filepath.Rel
	ErrInvalidFilename     = errors.New("invalid filename format")
	ErrInvalidTimestamp    = errors.New("invalid timestamp format")
	ErrSessionNotFound     = errors.New("session file not found")
)

// SessionFSRepository はローカルファイルシステムを使用して usecase.SessionRepository を実装します。
type SessionFSRepository struct {
	rootDir   string
	cacheRepo usecase.CacheRepository
}

// NewSessionFSRepository は新しい SessionFSRepository を作成します。
func NewSessionFSRepository(rootDir string, cacheRepo usecase.CacheRepository) *SessionFSRepository {
	return &SessionFSRepository{
		rootDir:   rootDir,
		cacheRepo: cacheRepo,
	}
}

// ListSessions はルートディレクトリをスキャンして .jsonl ファイルを探し、指定された年月および検索クエリでフィルタリングして SessionSummary オブジェクトを構築します。
// year == 0 && month == 0 の場合は、自動的に最新のセッションが存在する年月を特定してその月分のみを返します。
func (r *SessionFSRepository) ListSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error) {
	// ディレクトリが存在するか確認
	if _, err := os.Stat(r.rootDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil // 空のリストを正常に返す
	}

	var targetYear int
	var targetMonth int

	if year == 0 && month == 0 {
		// ディレクトリ内の最新のセッションログの日付を特定する
		var latestTime time.Time
		scanErr := filepath.WalkDir(r.rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			filename := d.Name()
			if !strings.HasPrefix(filename, "rollout-") || !strings.HasSuffix(filename, ".jsonl") {
				return nil
			}
			_, timestamp, err := parseSessionFilenameFn(filename)
			if err != nil {
				return nil
			}
			t, err := time.Parse(time.RFC3339, timestamp)
			if err == nil {
				if t.After(latestTime) {
					latestTime = t
				}
			}
			return nil
		})

		if scanErr != nil {
			return nil, scanErr
		}

		if latestTime.IsZero() {
			return nil, nil // セッションが存在しない場合は空リストを返す
		}
		targetYear = latestTime.Year()
		targetMonth = int(latestTime.Month())
	} else {
		targetYear = year
		targetMonth = month
	}

	var sessions []dto.SessionSummary

	err := filepath.WalkDir(r.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// rollout-*.jsonl ファイルのみを処理する
		filename := d.Name()
		if !strings.HasPrefix(filename, "rollout-") || !strings.HasSuffix(filename, ".jsonl") {
			return nil
		}

		id, timestamp, err := parseSessionFilenameFn(filename)
		if err != nil {
			// 無効なファイル名はスキップ
			return nil
		}

		// 年月によるフィルタリング
		parsedTime, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil
		}
		if parsedTime.Year() != targetYear || int(parsedTime.Month()) != targetMonth {
			return nil // 対象の年月でないためスキップ
		}

		// ファイルの詳細を取得
		info, err := d.Info()
		if err != nil {
			return nil // 情報を読み取れない場合はファイルをスキップ
		}

		// rootDir からの相対パスを計算
		relPath, err := filepathRelFn(r.rootDir, path)
		if err != nil {
			relPath = path
		}

		modTimeStr := info.ModTime().UTC().Format(time.RFC3339)

		// キャッシュから詳細情報をマージ
		var sSummary dto.SessionSummary
		cached, err := r.cacheRepo.GetSessionSummary(ctx, id)
		if err == nil && cached != nil {
			sSummary = *cached
			sSummary.Parsed = true
		} else {
			sSummary = dto.SessionSummary{
				ID:     id,
				Parsed: false,
			}
			// 未解析の場合、ファイルから親セッションIDをすばやく読み取る
			if parentID, readErr := readParentThreadIDFromFile(path); readErr == nil && parentID != "" {
				sSummary.ParentSessionID = &parentID
			}
		}

		// ファイルスキャン側のメタデータを設定（キャッシュデータがなければこれを優先、あれば補完）
		sSummary.FilePath = relPath
		sSummary.FileSize = info.Size()
		sSummary.FileModifiedAt = &modTimeStr
		if sSummary.Timestamp == nil {
			sSummary.Timestamp = &timestamp
		}

		// 検索クエリによるフィルタリング
		if query != "" {
			lowerQuery := strings.ToLower(query)
			idMatch := strings.Contains(strings.ToLower(sSummary.ID), lowerQuery)
			cwdMatch := false
			if sSummary.Cwd != nil {
				cwdMatch = strings.Contains(strings.ToLower(*sSummary.Cwd), lowerQuery)
			}
			branchMatch := false
			if sSummary.Branch != nil {
				branchMatch = strings.Contains(strings.ToLower(*sSummary.Branch), lowerQuery)
			}
			providerMatch := false
			if sSummary.ModelProvider != nil {
				providerMatch = strings.Contains(strings.ToLower(*sSummary.ModelProvider), lowerQuery)
			}

			if !idMatch && !cwdMatch && !branchMatch && !providerMatch {
				return nil // マッチしないのでスキップ
			}
		}

		sessions = append(sessions, sSummary)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan session directory: %w", err)
	}

	// 親子関係を解決
	childToParent := make(map[string]string)
	for i := range sessions {
		// キャッシュから読み込まれた ChildSessionIDs
		for _, childID := range sessions[i].ChildSessionIDs {
			childToParent[childID] = sessions[i].ID
		}
		// ファイルまたはキャッシュから読み込まれた ParentSessionID
		if sessions[i].ParentSessionID != nil {
			childToParent[sessions[i].ID] = *sessions[i].ParentSessionID
		}
	}

	parentToChildren := make(map[string]map[string]bool)
	for i := range sessions {
		parentToChildren[sessions[i].ID] = make(map[string]bool)
		for _, cid := range sessions[i].ChildSessionIDs {
			parentToChildren[sessions[i].ID][cid] = true
		}
	}

	for cid, pid := range childToParent {
		if parentToChildren[pid] == nil {
			parentToChildren[pid] = make(map[string]bool)
		}
		parentToChildren[pid][cid] = true
	}

	// 抽出された親子関係を双方向にマージして設定
	for i := range sessions {
		if parentID, ok := childToParent[sessions[i].ID]; ok {
			pID := parentID
			sessions[i].ParentSessionID = &pID
		}
		if children, ok := parentToChildren[sessions[i].ID]; ok && len(children) > 0 {
			var childIDs []string
			for cid := range children {
				childIDs = append(childIDs, cid)
			}
			sort.Strings(childIDs)
			sessions[i].ChildSessionIDs = childIDs
		}
	}

	return sessions, nil
}

var errSessionFound = errors.New("session found")

// GetSessionFilePath はセッションIDに合致するセッションファイルの絶対パスを返します。
func (r *SessionFSRepository) GetSessionFilePath(ctx context.Context, sessionID string) (string, error) {
	var targetPath string
	err := filepath.WalkDir(r.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		filename := d.Name()
		if !strings.HasPrefix(filename, "rollout-") || !strings.HasSuffix(filename, ".jsonl") {
			return nil
		}
		id, _, err := parseSessionFilenameFn(filename)
		if err == nil && id == sessionID {
			targetPath = path
			return errSessionFound
		}
		return nil
	})
	if errors.Is(err, errSessionFound) {
		return targetPath, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to scan for session file path: %w", err)
	}
	return "", fmt.Errorf("%w: ID %s", ErrSessionNotFound, sessionID)
}

// GetSessionIDByFilePath はセッションファイルパスに対応するセッションIDを返します。
func (r *SessionFSRepository) GetSessionIDByFilePath(ctx context.Context, filePath string) (string, error) {
	targetPath := filepath.Clean(filePath)
	var sessionID string

	err := filepath.WalkDir(r.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		filename := d.Name()
		if !strings.HasPrefix(filename, "rollout-") || !strings.HasSuffix(filename, ".jsonl") {
			return nil
		}
		if filepath.Clean(path) != targetPath {
			return nil
		}

		id, _, parseErr := parseSessionFilenameFn(filename)
		if parseErr != nil {
			return parseErr
		}
		sessionID = id
		return errSessionFound
	})
	if errors.Is(err, errSessionFound) {
		return sessionID, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to scan for session id: %w", err)
	}
	return "", fmt.Errorf("%w: path %s", ErrSessionNotFound, filePath)
}

// GetSessionModTime はセッションIDに合致するセッションファイルの最終更新日時を返します。
func (r *SessionFSRepository) GetSessionModTime(ctx context.Context, sessionID string) (time.Time, error) {
	filePath, err := r.GetSessionFilePath(ctx, sessionID)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to stat session file: %w", err)
	}
	return info.ModTime(), nil
}

var sessionFilenameRegex = regexp.MustCompile(`^rollout-(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2})-([a-fA-F0-9-]{36})\.jsonl$`)

// parseSessionFilename はセッションIDとJSTから変換されたUTC ISO8601タイムスタンプを抽出します。
func parseSessionFilename(filename string) (id, timestamp string, err error) {
	matches := sessionFilenameRegex.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidFilename, filename)
	}

	timestampPart := matches[1]
	uuidPart := matches[2]

	// 'T' の後のタイムスタンプのハイフンをコロンに変換
	// 正規表現により timestampPart は必ず "YYYY-MM-DDThh-mm-ss" の形式 (19文字)
	datePart := timestampPart[:10]
	timePart := timestampPart[11:]
	timePart = strings.ReplaceAll(timePart, "-", ":")

	localTimeStr := datePart + "T" + timePart

	// ローカル時間を解析
	loc := time.Local
	t, err := time.ParseInLocation("2006-01-02T15:04:05", localTimeStr, loc)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse local time '%s': %w", localTimeStr, err)
	}

	utcTimeStr := t.UTC().Format(time.RFC3339)
	return uuidPart, utcTimeStr, nil
}

// readParentThreadIDFromFile はセッションファイルの最初の行（session_meta レコード）を高速に読み込み、
// 親セッションID（parent_thread_id）が存在すればそれを返します。
func readParentThreadIDFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		return "", err
	}

	if env.Type == "session_meta" {
		var meta struct {
			ParentThreadID string `json:"parent_thread_id"`
		}
		if err := json.Unmarshal(env.Payload, &meta); err == nil {
			return meta.ParentThreadID, nil
		}
	}

	return "", nil
}
