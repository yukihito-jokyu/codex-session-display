package repository

import (
	"codex-session-display/internal/domain/model"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONLParser(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, tmpDir string) (string, func())
		wantErr    error
		wantErrMsg string
		verify     func(t *testing.T, records []*model.TypedRecord)
	}{
		{
			name: "empty file",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				emptyFilePath := filepath.Join(tmpDir, "empty.jsonl")
				if err := os.WriteFile(emptyFilePath, []byte(""), 0o644); err != nil {
					t.Fatal(err)
				}
				return emptyFilePath, nil
			},
			wantErr: ErrParseFailed,
		},
		{
			name: "file too large",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				largeFilePath := filepath.Join(tmpDir, "large.jsonl")
				f, err := os.Create(largeFilePath)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := f.Truncate(50*1024*1024 + 1); err != nil {
					t.Fatal(err)
				}
				return largeFilePath, nil
			},
			wantErr: ErrFileTooLarge,
		},
		{
			name: "normal logs",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				normalLogs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-uuid-1234","cwd":"/path/to/project","cli_version":"0.1.0","base_instructions":{"text":"You are a helpful assistant"}}}`,
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":"turn-uuid-5678","model":"gpt-4o","user_instructions":"Fix the bug"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-uuid-5678","started_at":1717084810,"model_context_window":128000}}`,
					`{"type":"response_item","timestamp":1717084815,"payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Done"}]}}`,
				}
				normalFilePath := filepath.Join(tmpDir, "normal.jsonl")
				if err := os.WriteFile(normalFilePath, []byte(strings.Join(normalLogs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return normalFilePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 4 {
					t.Fatalf("expected 4 records, got %d", len(records))
				}
				// 1. session_meta
				r0 := records[0]
				if r0.LineNumber != 1 || r0.Type != "session_meta" || r0.SessionMeta == nil {
					t.Errorf("r0 metadata mismatch: %+v", r0)
				} else {
					if r0.SessionMeta.ID != "session-uuid-1234" || r0.SessionMeta.Cwd != "/path/to/project" || r0.SessionMeta.BaseInstructions.Text != "You are a helpful assistant" {
						t.Errorf("r0 payload mismatch: %+v", r0.SessionMeta)
					}
				}
				// 2. turn_context
				r1 := records[1]
				if r1.LineNumber != 2 || r1.Type != "turn_context" || r1.TurnContext == nil {
					t.Errorf("r1 metadata mismatch: %+v", r1)
				} else {
					if r1.TurnContext.TurnID != "turn-uuid-5678" || r1.TurnContext.Model != "gpt-4o" || r1.TurnContext.UserInstructions != "Fix the bug" {
						t.Errorf("r1 payload mismatch: %+v", r1.TurnContext)
					}
				}
				// 3. event_msg
				r2 := records[2]
				if r2.LineNumber != 3 || r2.Type != "event_msg" || r2.SubType != "task_started" || r2.EventMsg == nil {
					t.Errorf("r2 metadata mismatch: %+v", r2)
				} else {
					if r2.EventMsg.TurnID != "turn-uuid-5678" || r2.EventMsg.ModelContextWindow != 128000 {
						t.Errorf("r2 payload mismatch: %+v", r2.EventMsg)
					}
				}
				// 4. response_item
				r3 := records[3]
				if r3.LineNumber != 4 || r3.Type != "response_item" || r3.SubType != "message" || r3.ResponseItem == nil {
					t.Errorf("r3 metadata mismatch: %+v", r3)
				} else {
					if r3.ResponseItem.Role != "assistant" || len(r3.ResponseItem.Content) != 1 || r3.ResponseItem.Content[0].Text != "Done" {
						t.Errorf("r3 payload mismatch: %+v", r3.ResponseItem)
					}
				}
			},
		},
		{
			name: "tolerant parsing",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				tolerantLogs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-uuid-1234","cwd":"/path/to/project","cli_version":"0.1.0","base_instructions":{"text":"You are a helpful assistant"}}}`,
					`{invalid json line here}`,
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":"turn-uuid-5678","model":"gpt-4o","user_instructions":"Fix the bug"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-uuid-5678","started_at":1717084810,"model_context_window":invalid_payload_here}}`,
					`{"type":"response_item","timestamp":1717084815,"payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Done"}]}}`,
				}
				tolerantFilePath := filepath.Join(tmpDir, "tolerant.jsonl")
				if err := os.WriteFile(tolerantFilePath, []byte(strings.Join(tolerantLogs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return tolerantFilePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 3 {
					t.Fatalf("expected 3 records, got %d", len(records))
				}
				// 1. session_meta (元の行番号: 1)
				r0 := records[0]
				if r0.LineNumber != 1 || r0.Type != "session_meta" {
					t.Errorf("r0 mismatch: %+v", r0)
				}
				// 2. turn_context (元の行番号: 3)
				r1 := records[1]
				if r1.LineNumber != 3 || r1.Type != "turn_context" {
					t.Errorf("r1 mismatch: %+v", r1)
				}
				// 3. response_item (元の行番号: 5)
				r2 := records[2]
				if r2.LineNumber != 5 || r2.Type != "response_item" {
					t.Errorf("r2 mismatch: %+v", r2)
				}
			},
		},
		{
			name: "too many errors",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				tooManyErrorsLogs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-uuid-1234","cwd":"/path/to/project","cli_version":"0.1.0","base_instructions":{"text":"You are a helpful assistant"}}}`,
					`{invalid json line 1}`,
					`{invalid json line 2}`,
					`{invalid json line 3}`,
				}
				tooManyErrorsFilePath := filepath.Join(tmpDir, "too_many_errors.jsonl")
				if err := os.WriteFile(tooManyErrorsFilePath, []byte(strings.Join(tooManyErrorsLogs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return tooManyErrorsFilePath, nil
			},
			wantErr: ErrParseFailed,
		},
		{
			name: "missing session meta",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				missingMetaLogs := []string{
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":"turn-uuid-5678","model":"gpt-4o","user_instructions":"Fix the bug"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-uuid-5678","started_at":1717084810,"model_context_window":128000}}`,
				}
				missingMetaFilePath := filepath.Join(tmpDir, "missing_meta.jsonl")
				if err := os.WriteFile(missingMetaFilePath, []byte(strings.Join(missingMetaLogs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return missingMetaFilePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 2 {
					t.Fatalf("expected 2 records, got %d", len(records))
				}
				r0 := records[0]
				if r0.Type != "turn_context" {
					t.Errorf("expected r0 to be turn_context, got %s", r0.Type)
				}
			},
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				return "non-existent-file.jsonl", nil
			},
			wantErrMsg: "failed to stat file",
		},
		{
			name: "empty lines only",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				filePath := filepath.Join(tmpDir, "empty_lines.jsonl")
				if err := os.WriteFile(filePath, []byte("\n\n  \n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return filePath, nil
			},
			wantErr: ErrParseFailed,
		},
		{
			name: "payload unmarshal errors",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				invalidPayloadLogs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":123}}`,
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":123}}`,
					`{"type":"response_item","timestamp":1717084815,"payload":{"type":123}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":123}}`,
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-uuid-1234"}}`,
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":"turn-uuid-5678"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-uuid-5678"}}`,
					`{"type":"response_item","timestamp":1717084815,"payload":{"type":"message","role":"assistant"}}`,
				}
				filePath := filepath.Join(tmpDir, "invalid_payloads.jsonl")
				if err := os.WriteFile(filePath, []byte(strings.Join(invalidPayloadLogs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return filePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 4 {
					t.Fatalf("expected 4 records, got %d", len(records))
				}
			},
		},
		{
			name: "unknown type",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-uuid-1234"}}`,
					`{"type":"unknown_type","timestamp":1717084805,"payload":{"foo":"bar"}}`,
				}
				filePath := filepath.Join(tmpDir, "unknown.jsonl")
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				return filePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 2 {
					t.Fatalf("expected 2 records, got %d", len(records))
				}
				if records[1].Type != "unknown_type" {
					t.Errorf("expected type 'unknown_type', got '%s'", records[1].Type)
				}
			},
		},
		{
			name: "long line",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				longMessage := strings.Repeat("a", 100*1024)
				longLog := `{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","message":"` + longMessage + `"}}`
				filePath := filepath.Join(tmpDir, "long_line.jsonl")
				if err := os.WriteFile(filePath, []byte(longLog), 0o644); err != nil {
					t.Fatal(err)
				}
				return filePath, nil
			},
			verify: func(t *testing.T, records []*model.TypedRecord) {
				if len(records) != 1 {
					t.Fatalf("expected 1 record, got %d", len(records))
				}
				if records[0].Type != "event_msg" {
					t.Errorf("expected type 'event_msg', got '%s'", records[0].Type)
				}
			},
		},
		{
			name: "open permission error",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				noReadFilePath := filepath.Join(tmpDir, "noread.jsonl")
				if err := os.WriteFile(noReadFilePath, []byte("{}"), 0o000); err != nil {
					t.Fatal(err)
				}
				cleanup := func() {
					_ = os.Chmod(noReadFilePath, 0o644)
				}
				return noReadFilePath, cleanup
			},
			wantErrMsg: "permission denied",
		},
		{
			name: "scanner error",
			setup: func(t *testing.T, tmpDir string) (string, func()) {
				return tmpDir, nil
			},
			wantErrMsg: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath, cleanup := tt.setup(t, tmpDir)
			if cleanup != nil {
				defer cleanup()
			}

			parser := NewJSONLParser()
			records, err := parser.ParseSessionFile(context.Background(), filePath)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
				}
			} else if tt.wantErrMsg != "" {
				if err == nil || (!strings.Contains(err.Error(), tt.wantErrMsg) && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantErrMsg))) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErrMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}

			if tt.verify != nil {
				tt.verify(t, records)
			}
		})
	}
}
