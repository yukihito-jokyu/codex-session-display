package usecase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ErrDownloadFailed はアップデートパッケージのダウンロードに失敗した際のエラーです。
	ErrDownloadFailed = errors.New("download failed")
	// ErrAppNotFound は展開されたパッケージ内に.appフォルダが見つからない際のエラーです。
	ErrAppNotFound = errors.New("no .app found in the extracted files")
	// ErrIllegalFilePath はZIPスリップなどの不正なファイルパスを検知した際のエラーです。
	ErrIllegalFilePath = errors.New("illegal file path")
)

// AppEventsEmitter はアプリケーションのイベント送信インターフェースです。
type AppEventsEmitter interface {
	Emit(eventName string, optionalData ...interface{})
}

// ApplyUpdateUseCase はアプリケーションを最新版に更新するユースケースです。
type ApplyUpdateUseCase struct {
	emitter        AppEventsEmitter
	tempDir        string
	httpClient     *http.Client
	CurrentAppPath string
	ExitFunc       func(int)
	CmdStartFunc   func(*exec.Cmd) error
}

// NewApplyUpdateUseCase は ApplyUpdateUseCase を作成します。
func NewApplyUpdateUseCase(emitter AppEventsEmitter, tempDir string) *ApplyUpdateUseCase {
	appPath, _ := detectAppPath()
	return &ApplyUpdateUseCase{
		emitter:        emitter,
		tempDir:        tempDir,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		CurrentAppPath: appPath,
		ExitFunc:       os.Exit,
		CmdStartFunc: func(cmd *exec.Cmd) error {
			return cmd.Start()
		},
	}
}

func detectAppPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "MacOS" {
		contentsDir := filepath.Dir(dir)
		if filepath.Base(contentsDir) == "Contents" {
			appDir := filepath.Dir(contentsDir)
			if strings.HasSuffix(appDir, ".app") {
				return appDir, nil
			}
		}
	}
	return dir, nil
}

// progressWriter は進捗を計算して通知するための io.Writer ラッパーです。
type progressWriter struct {
	emitter AppEventsEmitter
	total   int64
	written int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	if pw.total > 0 {
		progress := float64(pw.written) / float64(pw.total) * 100
		pw.emitter.Emit("update-progress", map[string]interface{}{
			"status":   "downloading",
			"progress": progress,
		})
	}
	return n, nil
}

// Execute はアップデートのダウンロードと適用を実行します。
func (uc *ApplyUpdateUseCase) Execute(ctx context.Context, downloadURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrDownloadFailed, resp.StatusCode)
	}

	totalSize := resp.ContentLength

	// 一時保存先ファイル
	if err := os.MkdirAll(uc.tempDir, 0o755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	tempFilePath := filepath.Join(uc.tempDir, "update.zip")
	out, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer out.Close()

	pw := &progressWriter{
		emitter: uc.emitter,
		total:   totalSize,
	}

	// データをコピーしながら進捗を出力
	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	if err != nil {
		return fmt.Errorf("failed to save download file: %w", err)
	}

	// ダウンロード完了通知
	uc.emitter.Emit("update-progress", map[string]interface{}{
		"status":   "download_complete",
		"progress": 100.0,
	})

	// 解凍先のパス
	extractedDir := filepath.Join(uc.tempDir, "extracted")
	if err := os.MkdirAll(extractedDir, 0o755); err != nil {
		return fmt.Errorf("failed to create extracted dir: %w", err)
	}

	uc.emitter.Emit("update-progress", map[string]interface{}{
		"status":   "extracting",
		"progress": 100.0,
	})

	if err := unzip(tempFilePath, extractedDir); err != nil {
		return fmt.Errorf("failed to unzip: %w", err)
	}

	// 解凍された .app を探す
	var extractedAppPath string
	err = filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasSuffix(info.Name(), ".app") {
			extractedAppPath = path
			return filepath.SkipDir // 最初の .app が見つかれば走査終了
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error searching for extracted app: %w", err)
	}

	if extractedAppPath == "" {
		return ErrAppNotFound
	}

	// シェルスクリプトの作成
	scriptPath := filepath.Join(uc.tempDir, "update.sh")
	pid := os.Getpid()
	if err := createUpdateScript(scriptPath, uc.CurrentAppPath, extractedAppPath, pid, uc.tempDir); err != nil {
		return fmt.Errorf("failed to create update script: %w", err)
	}

	// 外部プロセスの起動
	cmd := exec.Command("/bin/sh", scriptPath)
	if err := uc.CmdStartFunc(cmd); err != nil {
		return fmt.Errorf("failed to start update script: %w", err)
	}

	uc.emitter.Emit("update-progress", map[string]interface{}{
		"status":   "restarting",
		"progress": 100.0,
	})

	// 自身を終了
	uc.ExitFunc(0)

	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// zip slip 対策
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%w: %s", ErrIllegalFilePath, fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func createUpdateScript(scriptPath, currentApp, newApp string, pid int, tempDir string) error {
	scriptContent := fmt.Sprintf(`#!/bin/sh
# 呼び出し元プロセスが終了するのを待つ
while kill -0 %d 2>/dev/null; do
  sleep 0.5
done

# コピー先が存在することを確認（古いアプリを削除して置き換え）
if [ -d "%s" ]; then
  rm -rf "%s"
fi

cp -R "%s" "%s"

# 新しいアプリを起動する
open "%s"

# 一時作業ディレクトリをクリーンアップ
rm -rf "%s"
`, pid, currentApp, currentApp, newApp, currentApp, currentApp, tempDir)

	return os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
}
