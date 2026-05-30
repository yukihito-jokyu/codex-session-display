package dto

import "fmt"

// AppError は IPC 呼び出しでエラーが発生した際にフロントエンドへ返す共通エラー型です。
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error は error インターフェースを実装します。
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
