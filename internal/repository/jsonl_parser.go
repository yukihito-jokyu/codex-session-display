package repository

import (
	"bufio"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/utils/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrFileTooLarge = errors.New("file too large")
	ErrParseFailed  = errors.New("parse failed")
)

const maxFileSize = 50 * 1024 * 1024

type JSONLParser struct{}

func NewJSONLParser() *JSONLParser {
	return &JSONLParser{}
}

func (p *JSONLParser) ParseSessionFile(ctx context.Context, filePath string) ([]*model.TypedRecord, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if info.Size() == 0 {
		return nil, ErrParseFailed
	}
	if info.Size() > maxFileSize {
		return nil, ErrFileTooLarge
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var records []*model.TypedRecord
	scanner := bufio.NewScanner(file)
	// 1行がデフォルトの64KBを超える場合に備えてバッファサイズの上限を10MBに拡張
	const maxLineSize = 10 * 1024 * 1024 // 10MB
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	// トラッキング用カウンタ
	var totalLines int
	var failedLines int
	var hasSessionMeta bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue // 空行は無視し、分母にも含めない
		}

		totalLines++
		lineNumber := totalLines // トラッキングされているパース試行された行

		var env model.RecordEnvelope
		if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
			logger.Warn("failed to parse envelope", "line", lineNumber, "error", err)
			failedLines++
			continue
		}

		record := &model.TypedRecord{
			LineNumber: lineNumber,
			Type:       env.Type,
			Raw:        trimmed,
			Envelope:   env,
		}

		var payloadErr error
		switch env.Type {
		case "session_meta":
			var meta model.SessionMetaPayload
			if err := json.Unmarshal(env.Payload, &meta); err != nil {
				payloadErr = err
			} else {
				record.SessionMeta = &meta
				hasSessionMeta = true
			}
		case "turn_context":
			var turnCtx model.TurnContextPayload
			if err := json.Unmarshal(env.Payload, &turnCtx); err != nil {
				payloadErr = err
			} else {
				record.TurnContext = &turnCtx
			}
		case "event_msg":
			var event model.EventMsgPayload
			if err := json.Unmarshal(env.Payload, &event); err != nil {
				payloadErr = err
			} else {
				record.EventMsg = &event
				record.SubType = event.Type
			}
		case "response_item":
			var resp model.ResponseItemPayload
			if err := json.Unmarshal(env.Payload, &resp); err != nil {
				payloadErr = err
			} else {
				record.ResponseItem = &resp
				record.SubType = resp.Type
			}
		default:
			// 未知のレコードタイプはEnvelopeの状態で保持
		}

		if payloadErr != nil {
			logger.Warn("failed to parse payload", "line", lineNumber, "type", env.Type, "error", payloadErr)
			failedLines++
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	// 実際にパース試行した行数がない場合は空ファイルと同様にエラー
	if totalLines == 0 {
		return nil, ErrParseFailed
	}

	// パース失敗が50%を超える場合はエラー
	if float64(failedLines)/float64(totalLines) > 0.5 {
		return nil, ErrParseFailed
	}

	if !hasSessionMeta {
		logger.Warn("session_meta is missing, estimating session metadata from file name", "file", filePath)
	}

	return records, nil
}
