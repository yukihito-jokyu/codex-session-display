# Walkthrough

- `internal/usecase/get_session_detail.go` のキャッシュ再利用判定を、`parsed_at` 比較よりも「JSONL とキャッシュファイルの最終更新日時比較」を優先する実装へ変更
- `internal/usecase/repository.go` と `internal/repository/cache_fs.go` に、キャッシュ詳細ファイルの更新日時取得メソッドを追加
- `internal/usecase/get_session_detail_test.go` に、更新日時ベースのキャッシュヒット、古いキャッシュの再パース、保存失敗時の継続動作を検証するテストを追加
- `internal/repository/cache_fs_test.go` に、キャッシュ詳細ファイルの更新日時取得テストを追加
- 検証コマンド:
  - `go test ./internal/usecase -run TestGetSessionDetailUseCase_Execute`
  - `go test ./...`
  - `go test -tags production ./...`
  - `golangci-lint run` は `context loading failed: no go files to analyze` で実行環境側エラー
