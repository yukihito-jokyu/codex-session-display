# Implementation Plan

## Issue

- #138 `C8: キャッシュ管理とシステムロギングの実装`

## 目的

- セッション詳細キャッシュの再利用条件を仕様どおり「JSONL とキャッシュファイルの更新日時比較」に揃える
- キャッシュ保存失敗時は継続動作しつつ、ログで検知可能にする
- 本番ロガーのローテーション仕様を維持しつつ、ユニットテストで回帰を防ぐ

## 方針

1. `GetSessionDetailUseCase` のキャッシュ判定を `parsed_at` 比較からキャッシュファイルの `modtime` 比較へ変更する
2. `CacheRepository` / `CacheFSRepository` にキャッシュファイル更新日時取得の責務を追加する
3. キャッシュ無効化・再パース・保存失敗継続の振る舞いをユースケーステストで先に固定する
4. ログローテーションの既存テストを維持しつつ、不足ケースがあれば追加する
5. 実装完了後に `go test ./...` と `go test -tags production ./...` を実行する

## TDD サイクル

### Red

- キャッシュが JSONL より新しい場合だけ再利用されること
- キャッシュが古い場合は再パースされ、新しい結果が保存されること
- キャッシュ保存失敗時も詳細取得自体は成功すること

### Green

- 最小限のインターフェース変更と実装追加でテストを通す

### Refactor

- 時刻比較ロジックを読みやすく整理
- ダブルロギングを避けつつ、WARN/INFO のみ必要箇所へ残す

## 影響範囲

- `internal/usecase/get_session_detail.go`
- `internal/usecase/repository.go`
- `internal/repository/cache_fs.go`
- 関連テスト
- 必要なら設計ドキュメント同期
