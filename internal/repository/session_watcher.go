package repository

import (
	"codex-session-display/internal/utils/logger"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type fileState struct {
	modTime time.Time
	size    int64
}

// SessionWatcher は指定されたディレクトリを定期的に走査し、ロールアウトファイルの変更・新規追加を検知します。
type SessionWatcher struct {
	rootDir   string
	interval  time.Duration
	onChanged func(filePath string)

	mu    sync.Mutex
	files map[string]fileState
	stop  chan struct{}
}

// NewSessionWatcher は新しい SessionWatcher を作成します。
func NewSessionWatcher(rootDir string, interval time.Duration, onChanged func(filePath string)) *SessionWatcher {
	return &SessionWatcher{
		rootDir:   rootDir,
		interval:  interval,
		onChanged: onChanged,
		files:     make(map[string]fileState),
		stop:      make(chan struct{}),
	}
}

// Start はバックグラウンドでのファイル監視を開始します。
func (w *SessionWatcher) Start(ctx context.Context) {
	logger.Info("Starting session watcher...", "dir", w.rootDir, "interval", w.interval)

	// 初回スキャンを実行して既存ファイルを登録（起動時の通知は行わない）
	w.scan(true)

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.scan(false)
			case <-w.stop:
				logger.Info("Session watcher stopped via channel")
				return
			case <-ctx.Done():
				logger.Info("Session watcher stopped via context done")
				return
			}
		}
	}()
}

// Stop はファイル監視を停止します。
func (w *SessionWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case <-w.stop:
		// すでにクローズされている場合は何もしない
	default:
		close(w.stop)
	}
}

func (w *SessionWatcher) scan(isInitial bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// ディレクトリが存在しない場合は処理をスキップ
	if _, err := os.Stat(w.rootDir); os.IsNotExist(err) {
		return
	}

	currentFiles := make(map[string]fileState)

	err := filepath.WalkDir(w.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Warn("Failed to walk path in session watcher", "path", path, "error", err)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		filename := d.Name()
		// セッションログファイル (rollout-*.jsonl) のみを監視
		if !strings.HasPrefix(filename, "rollout-") || !strings.HasSuffix(filename, ".jsonl") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			logger.Warn("Failed to get file info in session watcher", "path", path, "error", err)
			return nil
		}

		currentFiles[path] = fileState{
			modTime: info.ModTime(),
			size:    info.Size(),
		}

		return nil
	})
	if err != nil {
		logger.Error("WalkDir failed in session watcher", "dir", w.rootDir, "error", err)
		return
	}

	// 新規ファイルおよび変更ファイルの検知
	for path, state := range currentFiles {
		prevState, exists := w.files[path]
		if !exists {
			// 新規ファイル検知
			w.files[path] = state
			if !isInitial {
				logger.Info("New session log detected", "path", path)
				w.onChanged(path)
			}
		} else if state.modTime.After(prevState.modTime) || state.size != prevState.size {
			// ファイルの更新検知
			w.files[path] = state
			if !isInitial {
				logger.Info("Session log modification detected", "path", path)
				w.onChanged(path)
			}
		}
	}

	// 削除されたファイルを管理対象から除外
	for path := range w.files {
		if _, exists := currentFiles[path]; !exists {
			delete(w.files, path)
		}
	}
}
