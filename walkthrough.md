# Walkthrough

- `internal/usecase/get_session_detail_test.go` に、`token_count` の直前レコードがノード未生成でも、さらに前のマップ済みノードへフォールバックして解決されることを検証するテストを追加
- `BoundToNodeID` が `node-3` になることと、同ノードにトークンバッジが集約されることを確認
- 検証コマンド:
  - `go test ./internal/usecase -run TestGetSessionDetailUseCase_Execute`
  - `go test -tags production ./internal/usecase -run TestGetSessionDetailUseCase_Execute`
  - `go test ./...`
  - `go test -tags production ./...`
