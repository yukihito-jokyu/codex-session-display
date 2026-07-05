package usecase

import (
	"codex-session-display/internal/domain/dto"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeClaudeCorpus(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string // filename -> content
		verify  func(t *testing.T, res *dto.AnalyzeResult)
		wantErr bool
	}{
		{
			name: "detailed_metrics_and_privacy_check",
			files: map[string]string{
				"projectA/session-1.jsonl": `{"type":"user","sessionId":"session-1","cwd":"/path/to/projectA","timestamp":"2026-07-05T06:00:00Z","message":{"id":"msg-1","role":"user","content":[{"type":"text","text":"hello world"}]}}` + "\n" +
					`{"type":"assistant","sessionId":"session-1","cwd":"/path/to/projectA","timestamp":"2026-07-05T06:00:05Z","costUSD":0.0015,"message":{"id":"msg-2","role":"assistant","content":[{"type":"thinking","thinking":"let me think"},{"type":"tool_use","id":"call-1","name":"bash","input":{"command":"echo test"}}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":5}}}` + "\n" +
					`{"type":"user","sessionId":"session-1","cwd":"/path/to/projectA","timestamp":"2026-07-05T06:00:10Z","message":{"id":"msg-3","role":"user","content":[{"type":"tool_result","tool_use_id":"call-1","content":"test output"}]}}` + "\n",
				"projectA/session-invalid.jsonl": `{"type":"user","sessionId":"session-2",` + "\n", // Parse error line
			},
			verify: func(t *testing.T, res *dto.AnalyzeResult) {
				if res.TotalFiles != 2 {
					t.Errorf("expected 2 files, got %d", res.TotalFiles)
				}
				if res.TotalLines != 4 {
					t.Errorf("expected 4 lines, got %d", res.TotalLines)
				}

				// パースエラーのチェック
				if len(res.ParseErrors) != 1 {
					t.Fatalf("expected 1 parse error file info, got %d", len(res.ParseErrors))
				}
				if res.ParseErrors[0].Count != 1 {
					t.Errorf("expected 1 parse error count in invalid file, got %d", res.ParseErrors[0].Count)
				}
				if res.ParseErrors[0].FileID == "" {
					t.Errorf("expected non-empty FileID for parse error")
				}

				// fieldPaths のスキーマチェック
				// message.role は string
				if counts, ok := res.FieldPaths["message.role"]; ok {
					if counts["string"] != 3 {
						t.Errorf("expected message.role string count to be 3, got %d", counts["string"])
					}
				} else {
					t.Errorf("expected field path message.role to be tracked")
				}

				// message.content[].type は string
				if counts, ok := res.FieldPaths["message.content[].type"]; ok {
					if counts["string"] != 4 { // text, thinking, tool_use, tool_result
						t.Errorf("expected message.content[].type string count to be 4, got %d", counts["string"])
					}
				} else {
					t.Errorf("expected field path message.content[].type to be tracked")
				}

				// contentTypes のチェック
				if res.ContentTypes["text"] != 1 {
					t.Errorf("expected 1 text content type, got %d", res.ContentTypes["text"])
				}
				if res.ContentTypes["thinking"] != 1 {
					t.Errorf("expected 1 thinking content type, got %d", res.ContentTypes["thinking"])
				}
				if res.ContentTypes["tool_use"] != 1 {
					t.Errorf("expected 1 tool_use content type, got %d", res.ContentTypes["tool_use"])
				}
				if res.ContentTypes["tool_result"] != 1 {
					t.Errorf("expected 1 tool_result content type, got %d", res.ContentTypes["tool_result"])
				}

				// toolNames のチェック
				if res.ToolNames["bash"] != 1 {
					t.Errorf("expected 1 bash tool call, got %d", res.ToolNames["bash"])
				}

				// usageKeys のチェック
				if res.UsageKeys["input_tokens"] != 1 {
					t.Errorf("expected 1 input_tokens usage, got %d", res.UsageKeys["input_tokens"])
				}
				if res.UsageKeys["cache_read_input_tokens"] != 1 {
					t.Errorf("expected 1 cache_read_input_tokens usage, got %d", res.UsageKeys["cache_read_input_tokens"])
				}

				// プライバシーチェック（長さバケット）
				// text: "hello world" (長さ11) => "1-100" バケット
				if res.PrivacyMetrics.TextLengthDist["1-100"] != 1 {
					t.Errorf("expected 1 text in '1-100' bucket, got %d", res.PrivacyMetrics.TextLengthDist["1-100"])
				}
				// thinking: "let me think" (長さ12) => "1-100" バケット
				if res.PrivacyMetrics.ThinkingLengthDist["1-100"] != 1 {
					t.Errorf("expected 1 thinking in '1-100' bucket, got %d", res.PrivacyMetrics.ThinkingLengthDist["1-100"])
				}

				// プライバシーチェック（コマンドハッシュ）
				for k, v := range res.PrivacyMetrics.CommandHashDist {
					if len(k) != 64 {
						t.Errorf("expected command hash key to be 64-char hex string (SHA256), got %s", k)
					}
					if v != 1 {
						t.Errorf("expected command hash count to be 1, got %d", v)
					}
				}

				// プライバシーチェック（ツール出力ハッシュ）
				for k, v := range res.PrivacyMetrics.ToolOutputDist {
					if len(k) != 64 {
						t.Errorf("expected tool output hash key to be 64-char hex string, got %s", k)
					}
					if v != 1 {
						t.Errorf("expected tool output hash count to be 1, got %d", v)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			uc := NewAnalyzeClaudeCorpusUseCase()
			uc.SetCustomProjectsDir(tmpDir)

			res, err := uc.Execute(context.Background(), dto.AnalyzeOptions{
				ProjectSource: "home",
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, res)
			}
		})
	}
}
