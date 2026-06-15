# Issue #187 実装結果

## 実装内容

- `TokenCountEntry` にtoken_count単位の `model_context_window` を追加した。
- token_count固有値を優先し、無効時は所属ターンの `task_started` 値へ
  フォールバックし、どちらも無効なら0へ正規化した。
- `Last Token Consumption per Index` に右軸の `Context Usage (%)` 系列を追加した。
- 使用率は入力トークンをコンテキスト上限で除算し、キャッシュ入力を差し引かず、
  100%超過値もクランプしない。
- 算出不能な点を非描画にし、ツールチップでは `N/A` と表示した。
- Context Usage点を既存のタイムライン、ノード、TOKEN COUNT LOG、
  BottomPanelの共通選択経路へ接続した。
- キャッシュスキーマを6へ更新し、要件定義、詳細設計、ADR 0031、
  Wails生成型を同期した。

## TDD

1. token_count固有のコンテキスト上限をGoテストでRED/GREEN
2. ターン上限フォールバックと無効値の0正規化をGoテストでRED/GREEN
3. 右軸系列、算出不能点、`N/A` ツールチップをPlaywrightでRED/GREEN
4. キャッシュ入力を含む150%表示と共通選択をPlaywrightで回帰テスト

## 検証結果

- `go test ./...`: 139件成功
- `go test -tags production ./...`: 143件成功
- `golangci-lint run`: 0件
- `npm run lint`: 成功
- `npm run build`: 成功
- Context Usage対象E2E: 2件成功
- `task test:e2e`: 96件成功
- `wails generate module`: 成功
- go-review: 指摘事項なし
- frontend-review: 指摘事項なし
