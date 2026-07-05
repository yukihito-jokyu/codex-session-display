package usecase

import (
	"bufio"
	"codex-session-display/internal/domain/dto"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var errClaudeConfigDirNotSet = errors.New("CLAUDE_CONFIG_DIR is not set")

type AnalyzeClaudeCorpusUseCase struct {
	customProjectsDir string
}

func NewAnalyzeClaudeCorpusUseCase() *AnalyzeClaudeCorpusUseCase {
	return &AnalyzeClaudeCorpusUseCase{}
}

func (uc *AnalyzeClaudeCorpusUseCase) SetCustomProjectsDir(dir string) {
	uc.customProjectsDir = dir
}

// Duplicate structs removed (already defined in get_session_detail.go)

func (uc *AnalyzeClaudeCorpusUseCase) Execute(ctx context.Context, options dto.AnalyzeOptions) (*dto.AnalyzeResult, error) {
	projectsDir := uc.customProjectsDir
	if projectsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		if options.ProjectSource == "config" {
			if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
				projectsDir = filepath.Join(configDir, "projects")
			} else {
				return nil, errClaudeConfigDirNotSet
			}
		} else {
			projectsDir = filepath.Join(home, ".claude", "projects")
		}
	}

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return &dto.AnalyzeResult{
			FieldPaths:   make(map[string]dto.TypeCount),
			ContentTypes: make(map[string]int64),
			ToolNames:    make(map[string]int64),
			UsageKeys:    make(map[string]int64),
			PrivacyMetrics: dto.PrivacyMetrics{
				TextLengthDist:     make(map[string]int64),
				ThinkingLengthDist: make(map[string]int64),
				CommandHashDist:    make(map[string]int64),
				ToolOutputDist:     make(map[string]int64),
			},
		}, nil
	}

	result := &dto.AnalyzeResult{
		FieldPaths:   make(map[string]dto.TypeCount),
		ContentTypes: make(map[string]int64),
		ToolNames:    make(map[string]int64),
		UsageKeys:    make(map[string]int64),
		PrivacyMetrics: dto.PrivacyMetrics{
			TextLengthDist:     make(map[string]int64),
			ThinkingLengthDist: make(map[string]int64),
			CommandHashDist:    make(map[string]int64),
			ToolOutputDist:     make(map[string]int64),
		},
	}

	// パースエラー追跡用のマップ (fileID -> errorCount)
	parseErrorsMap := make(map[string]int64)

	err := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}

		result.TotalFiles++

		file, err := os.Open(path)
		if err != nil {
			return nil // スキップ
		}
		defer file.Close()

		// 匿名ファイルIDの生成
		fileID := getSHA256Hash(path)[:12]

		scanner := bufio.NewScanner(file)
		const maxLineSize = 10 * 1024 * 1024
		scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			result.TotalLines++

			// 1. ジェネリックな JSON 解析 (fieldPaths 抽出用)
			var genericJSON interface{}
			if err := json.Unmarshal([]byte(line), &genericJSON); err != nil {
				parseErrorsMap[fileID]++
				continue
			}
			traverseJSON(genericJSON, "", result.FieldPaths)

			// 2. 構造化データの抽出 (contentTypes, toolNames, usageKeys, プライバシー指標)
			var rec claudeDetailRecord
			if err := json.Unmarshal([]byte(line), &rec); err == nil {
				for i := range rec.Message.Content.Items {
					item := &rec.Message.Content.Items[i]
					if item.Type != "" {
						result.ContentTypes[item.Type]++
					}

					switch item.Type {
					case "text":
						bucket := getLengthBucket(len(item.Text))
						result.PrivacyMetrics.TextLengthDist[bucket]++
					case "thinking":
						bucket := getLengthBucket(len(item.Thinking))
						result.PrivacyMetrics.ThinkingLengthDist[bucket]++
					case "tool_use":
						if item.Name != "" {
							result.ToolNames[item.Name]++
						}
						// コマンドのハッシュ集計
						var inputMap map[string]interface{}
						if err := json.Unmarshal(item.Input, &inputMap); err == nil {
							if cmdVal, ok := inputMap["command"]; ok {
								if cmdStr, ok := cmdVal.(string); ok {
									hash := getSHA256Hash(cmdStr)
									result.PrivacyMetrics.CommandHashDist[hash]++
								}
							} else {
								hash := getSHA256Hash(string(item.Input))
								result.PrivacyMetrics.CommandHashDist[hash]++
							}
						} else {
							hash := getSHA256Hash(string(item.Input))
							result.PrivacyMetrics.CommandHashDist[hash]++
						}
					case "tool_result":
						// ツール出力のハッシュ集計
						var contentStr string
						if err := json.Unmarshal(item.Content, &contentStr); err == nil {
							hash := getSHA256Hash(contentStr)
							result.PrivacyMetrics.ToolOutputDist[hash]++
						} else {
							hash := getSHA256Hash(string(item.Content))
							result.PrivacyMetrics.ToolOutputDist[hash]++
						}
					}
				}

				// テキストが単一文字列の場合のケア
				if rec.Message.Content.Text != "" {
					result.ContentTypes["text"]++
					bucket := getLengthBucket(len(rec.Message.Content.Text))
					result.PrivacyMetrics.TextLengthDist[bucket]++
				}

				if rec.Message.Usage != nil {
					usage := rec.Message.Usage
					if usage.InputTokens > 0 {
						result.UsageKeys["input_tokens"]++
					}
					if usage.OutputTokens > 0 {
						result.UsageKeys["output_tokens"]++
					}
					if usage.CacheReadInputTokens > 0 {
						result.UsageKeys["cache_read_input_tokens"]++
					}
					if usage.CacheCreationTokens > 0 {
						result.UsageKeys["cache_creation_input_tokens"]++
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk projects dir: %w", err)
	}

	// パースエラーの詰め込み
	for fid, count := range parseErrorsMap {
		result.ParseErrors = append(result.ParseErrors, dto.ParseErrorInfo{
			FileID: fid,
			Count:  count,
		})
	}

	return result, nil
}

func traverseJSON(val interface{}, path string, result map[string]dto.TypeCount) {
	var typeName string
	switch val.(type) {
	case nil:
		typeName = "null"
	case bool:
		typeName = "boolean"
	case float64:
		typeName = "number"
	case string:
		typeName = "string"
	case []interface{}:
		typeName = "array"
	case map[string]interface{}:
		typeName = "object"
	default:
		typeName = "unknown"
	}

	if path != "" {
		if _, ok := result[path]; !ok {
			result[path] = make(dto.TypeCount)
		}
		result[path][typeName]++
	}

	switch v := val.(type) {
	case map[string]interface{}:
		for k, child := range v {
			var childPath string
			if path == "" {
				childPath = k
			} else {
				childPath = path + "." + k
			}
			traverseJSON(child, childPath, result)
		}
	case []interface{}:
		for _, child := range v {
			var childPath string
			// 配列要素へのパス表現は [] を付与する
			if path == "" {
				childPath = "[]"
			} else {
				childPath = path + "[]"
			}
			traverseJSON(child, childPath, result)
		}
	}
}

func getLengthBucket(length int) string {
	if length == 0 {
		return "0"
	} else if length <= 100 {
		return "1-100"
	} else if length <= 500 {
		return "101-500"
	} else if length <= 2000 {
		return "501-2000"
	} else {
		return "2001+"
	}
}

func getSHA256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}
