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
	rootDir           string
	claudeProjectsDir string
	cacheRepo         usecase.CacheRepository
}

// NewSessionFSRepository は新しい SessionFSRepository を作成します。
func NewSessionFSRepository(rootDir string, cacheRepo usecase.CacheRepository) *SessionFSRepository {
	return &SessionFSRepository{
		rootDir:   rootDir,
		cacheRepo: cacheRepo,
	}
}

// SetClaudeProjectsDir は Claude Code transcript の projects ディレクトリを設定します。
func (r *SessionFSRepository) SetClaudeProjectsDir(projectsDir string) {
	r.claudeProjectsDir = projectsDir
}

// ListSessions はルートディレクトリをスキャンして .jsonl ファイルを探し、指定された年月および検索クエリでフィルタリングして SessionSummary オブジェクトを構築します。
// year == 0 && month == 0 の場合は、自動的に最新のセッションが存在する年月を特定してその月分のみを返します。
func (r *SessionFSRepository) ListSessions(ctx context.Context, provider dto.SessionProvider, year, month int, query string) ([]dto.SessionSummary, error) {
	if provider == dto.SessionProviderClaude {
		return r.listClaudeSessions(ctx, year, month, query)
	}
	// ディレクトリが存在するか確認
	if _, err := os.Stat(r.rootDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil // 空のリストを正常に返す
	}

	type sessionFileInfo struct {
		path      string
		info      fs.FileInfo
		id        string
		timestamp string
		time      time.Time
	}

	var allFiles []sessionFileInfo
	var latestTime time.Time

	// 1回の WalkDir でファイルを走査し、必要な情報を収集
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

		id, timestamp, err := parseSessionFilenameFn(filename)
		if err != nil {
			return nil
		}

		t, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // ファイル情報を読み取れない場合はスキップ
		}

		allFiles = append(allFiles, sessionFileInfo{
			path:      path,
			info:      info,
			id:        id,
			timestamp: timestamp,
			time:      t,
		})

		if t.After(latestTime) {
			latestTime = t
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan session directory: %w", err)
	}

	var targetYear int
	var targetMonth int

	if year == 0 && month == 0 {
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

	for _, file := range allFiles {
		// 年月によるフィルタリング
		if file.time.Year() != targetYear || int(file.time.Month()) != targetMonth {
			continue
		}

		// rootDir からの相対パスを計算
		relPath, err := filepathRelFn(r.rootDir, file.path)
		if err != nil {
			relPath = file.path
		}

		modTimeStr := file.info.ModTime().UTC().Format(time.RFC3339)

		// キャッシュから詳細情報をマージ
		var sSummary dto.SessionSummary
		cached, err := r.cacheRepo.GetSessionSummary(ctx, dto.SessionProviderCodex, file.id)
		if err == nil && cached != nil {
			sSummary = *cached
			sSummary.Provider = dto.SessionProviderCodex
			sSummary.Parsed = true
		} else {
			sSummary = dto.SessionSummary{
				ID:       file.id,
				Provider: dto.SessionProviderCodex,
				Parsed:   false,
			}
			// 未解析の場合、ファイルから親セッションIDをすばやく読み取る
			if parentID, readErr := readParentThreadIDFromFile(file.path); readErr == nil && parentID != "" {
				sSummary.ParentSessionID = &parentID
			}
		}

		// ファイルスキャン側のメタデータを設定（キャッシュデータがなければこれを優先、あれば補完）
		sSummary.FilePath = relPath
		sSummary.FileSize = file.info.Size()
		sSummary.FileModifiedAt = &modTimeStr
		if sSummary.Timestamp == nil {
			sSummary.Timestamp = &file.timestamp
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
				continue // マッチしないのでスキップ
			}
		}

		sessions = append(sessions, sSummary)
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

func (r *SessionFSRepository) listClaudeSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error) {
	if r.claudeProjectsDir == "" {
		return nil, nil
	}
	if _, err := os.Stat(r.claudeProjectsDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	var allSessions []dto.SessionSummary
	var latestTime time.Time

	err := filepath.WalkDir(r.claudeProjectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}

		summary, err := r.readClaudeSessionSummary(ctx, path)
		if err != nil {
			return nil
		}
		if summary == nil {
			return nil
		}

		allSessions = append(allSessions, *summary)

		// 最新のタイムスタンプを追跡
		if summary.Timestamp != nil {
			var t time.Time
			var parseErr error
			t, parseErr = time.Parse(time.RFC3339, *summary.Timestamp)
			if parseErr != nil {
				t, parseErr = time.Parse(time.RFC3339Nano, *summary.Timestamp)
			}
			if parseErr == nil {
				if t.After(latestTime) {
					latestTime = t
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan Claude sessions directory: %w", err)
	}

	// 親子関係を解決
	childToParent := make(map[string]string)
	for i := range allSessions {
		for _, childID := range allSessions[i].ChildSessionIDs {
			childToParent[childID] = allSessions[i].ID
		}
		if allSessions[i].ParentSessionID != nil {
			childToParent[allSessions[i].ID] = *allSessions[i].ParentSessionID
		}
	}

	parentToChildren := make(map[string]map[string]bool)
	for i := range allSessions {
		parentToChildren[allSessions[i].ID] = make(map[string]bool)
		for _, cid := range allSessions[i].ChildSessionIDs {
			parentToChildren[allSessions[i].ID][cid] = true
		}
	}

	for cid, pid := range childToParent {
		if parentToChildren[pid] == nil {
			parentToChildren[pid] = make(map[string]bool)
		}
		parentToChildren[pid][cid] = true
	}

	for i := range allSessions {
		if parentID, ok := childToParent[allSessions[i].ID]; ok {
			pID := parentID
			allSessions[i].ParentSessionID = &pID
		}
		if children, ok := parentToChildren[allSessions[i].ID]; ok && len(children) > 0 {
			var childIDs []string
			for cid := range children {
				childIDs = append(childIDs, cid)
			}
			sort.Strings(childIDs)
			allSessions[i].ChildSessionIDs = childIDs
		}
	}

	var targetYear int
	var targetMonth int

	if year == 0 && month == 0 {
		if latestTime.IsZero() {
			return nil, nil // セッションが存在しない場合
		}
		targetYear = latestTime.Year()
		targetMonth = int(latestTime.Month())
	} else {
		targetYear = year
		targetMonth = month
	}

	var filtered []dto.SessionSummary
	for i := range allSessions {
		summary := &allSessions[i]
		// 年月によるフィルタリング
		if summary.Timestamp != nil {
			var t time.Time
			var parseErr error
			t, parseErr = time.Parse(time.RFC3339, *summary.Timestamp)
			if parseErr != nil {
				t, parseErr = time.Parse(time.RFC3339Nano, *summary.Timestamp)
			}
			if parseErr != nil || t.Year() != targetYear || int(t.Month()) != targetMonth {
				continue
			}
		} else {
			continue // タイムスタンプがないものは除外（通常はフォールバックがあるのでここには来ない）
		}

		// クエリによるフィルタリング
		if query != "" && !matchesSessionQuery(summary, query) {
			continue
		}

		filtered = append(filtered, *summary)
	}

	// 日時で降順ソート
	sort.Slice(filtered, func(i, j int) bool {
		t1, err1 := time.Parse(time.RFC3339, *filtered[i].Timestamp)
		if err1 != nil {
			t1, err1 = time.Parse(time.RFC3339Nano, *filtered[i].Timestamp)
		}
		t2, err2 := time.Parse(time.RFC3339, *filtered[j].Timestamp)
		if err2 != nil {
			t2, err2 = time.Parse(time.RFC3339Nano, *filtered[j].Timestamp)
		}
		if err1 != nil || err2 != nil {
			return false
		}
		return t1.After(t2)
	})

	return filtered, nil
}

func (r *SessionFSRepository) readClaudeSessionSummary(ctx context.Context, path string) (*dto.SessionSummary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	relPath, err := filepathRelFn(r.claudeProjectsDir, path)
	if err != nil {
		relPath = path
	}
	encodedProject := filepath.Base(filepath.Dir(path))
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	// ファイル名からではなく、ファイルに記録されている本来のセッションIDを読み出す
	if embeddedID, err := readClaudeSessionIDFromFile(path); err == nil && embeddedID != "" {
		sessionID = embeddedID
	}
	modTimeStr := info.ModTime().UTC().Format(time.RFC3339)

	var sSummary dto.SessionSummary
	cached, err := r.cacheRepo.GetSessionSummary(ctx, dto.SessionProviderClaude, sessionID)
	if err == nil && cached != nil {
		sSummary = *cached
		sSummary.Provider = dto.SessionProviderClaude
		sSummary.Parsed = true
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		var cwd *string
		var timestamp *string
		var gitBranch *string
		var version *string
		messageCount := 0
		toolCallCount := 0
		totalCostUSD := 0.0

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var record claudeTranscriptRecord
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				continue
			}
			if record.AgentID != "" {
				sessionID = record.AgentID
			} else if record.SessionID != "" {
				sessionID = record.SessionID
			}
			if cwd == nil && record.Cwd != "" {
				value := record.Cwd
				cwd = &value
			}
			if timestamp == nil && record.Timestamp != "" {
				value := record.Timestamp
				timestamp = &value
			}
			if gitBranch == nil && record.GitBranch != "" {
				value := record.GitBranch
				gitBranch = &value
			}
			if version == nil && record.Version != "" {
				value := record.Version
				version = &value
			}
			if record.Type == "user" || record.Type == "assistant" {
				messageCount++
			}
			totalCostUSD += record.CostUSD
			toolCallCount += countClaudeToolUses(record.Message.Content)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		sSummary = dto.SessionSummary{
			ID:            sessionID,
			Provider:      dto.SessionProviderClaude,
			Cwd:           cwd,
			Timestamp:     timestamp,
			Branch:        gitBranch,
			CliVersion:    version,
			Parsed:        false,
			MessageCount:  &messageCount,
			ToolCallCount: &toolCallCount,
			TotalCostUSD:  &totalCostUSD,
		}
	}

	sSummary.FilePath = relPath
	sSummary.FileSize = info.Size()
	sSummary.FileModifiedAt = &modTimeStr
	sSummary.EncodedProject = &encodedProject
	if sSummary.Timestamp == nil || *sSummary.Timestamp == "" {
		sSummary.Timestamp = &modTimeStr
	}

	if sSummary.ParentSessionID == nil || *sSummary.ParentSessionID == "" {
		parentID := getClaudeParentSessionID(path)
		if parentID != "" {
			sSummary.ParentSessionID = &parentID
		}
	}

	return &sSummary, nil
}

type claudeTranscriptRecord struct {
	Type      string  `json:"type"`
	SessionID string  `json:"sessionId"`
	AgentID   string  `json:"agentId"`
	Cwd       string  `json:"cwd"`
	Timestamp string  `json:"timestamp"`
	CostUSD   float64 `json:"costUSD"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	GitBranch string `json:"gitBranch"`
	Version   string `json:"version"`
}

func countClaudeToolUses(content json.RawMessage) int {
	if len(content) == 0 {
		return 0
	}

	var items []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(content, &items); err == nil {
		count := 0
		for _, item := range items {
			if item.Type == "tool_use" {
				count++
			}
		}
		return count
	}

	var item struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(content, &item); err == nil && item.Type == "tool_use" {
		return 1
	}
	return 0
}

func matchesSessionQuery(summary *dto.SessionSummary, query string) bool {
	lowerQuery := strings.ToLower(query)
	if strings.Contains(strings.ToLower(summary.ID), lowerQuery) {
		return true
	}
	if summary.Cwd != nil && strings.Contains(strings.ToLower(*summary.Cwd), lowerQuery) {
		return true
	}
	if summary.Branch != nil && strings.Contains(strings.ToLower(*summary.Branch), lowerQuery) {
		return true
	}
	if summary.ModelProvider != nil && strings.Contains(strings.ToLower(*summary.ModelProvider), lowerQuery) {
		return true
	}
	if summary.EncodedProject != nil && strings.Contains(strings.ToLower(*summary.EncodedProject), lowerQuery) {
		return true
	}
	return strings.Contains(string(summary.Provider), lowerQuery)
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

	if r.claudeProjectsDir != "" {
		err = filepath.WalkDir(r.claudeProjectsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
				return nil
			}
			if strings.TrimSuffix(d.Name(), ".jsonl") == sessionID {
				targetPath = path
				return errSessionFound
			}
			embeddedSessionID, readErr := readClaudeSessionIDFromFile(path)
			if readErr != nil || embeddedSessionID != sessionID {
				return nil
			}
			targetPath = path
			return errSessionFound
		})
		if errors.Is(err, errSessionFound) {
			return targetPath, nil
		}
		if err != nil {
			return "", fmt.Errorf("failed to scan Claude sessions directory: %w", err)
		}
	}

	return "", fmt.Errorf("%w: ID %s", ErrSessionNotFound, sessionID)
}

func readClaudeSessionIDFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record claudeTranscriptRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		if record.AgentID != "" {
			return record.AgentID, nil
		}
		if record.SessionID != "" {
			return record.SessionID, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
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

// getClaudeParentSessionID はパス構造から親セッションIDを抽出して返します。
// 例: ~/.claude/projects/project-name/parent-session-id/subagents/subagent-id.jsonl
func getClaudeParentSessionID(path string) string {
	path = filepath.Clean(path)
	dir := filepath.Dir(path) // .../parent-session-id/subagents
	if filepath.Base(dir) == "subagents" {
		parentDir := filepath.Dir(dir) // .../parent-session-id
		return filepath.Base(parentDir)
	}
	return ""
}
