# Issue #175 実装結果

## 変更概要

- `SessionDetailResponse.timeline` を追加し、通常ターンと疑似ターンのUser/AI発言をJSONL順で返すようにした。
- 同一ターン内の同一ロール・本文・タイムスタンプの重複発言を1件へ統合した。
- 表示単位へ紐付く `last_token_usage` の合算、件数、最新 `total_token_usage` を集約した。
- セッション詳細画面を左タイムライン、中央キャンバス・下部パネル、右分析パネルの3領域にした。
- キャッシュスキーマを3へ更新し、バージョン2以前を再解析対象にした。

## UI

- 通常ターンに所要時間とターン消費トークンを表示する。
- 疑似ターンに「ターン外イベント」を表示する。
- User/AI本文は折りたたまず常時表示する。
- 発言ごとに増分トークン、紐付け件数、累計、または「計測なし」を表示する。
- 中央領域が狭くなっても下部トークン詳細の分割幅を変更できるようにした。

## ドキュメント

- `docs/requirements.md`
- `docs/detailed-design.md`
- `docs/adr/0029-conversation-timeline-dto.md`

## 検証結果

- `go test ./...`: 131件成功
- `go test -tags production ./...`: 135件成功
- `golangci-lint run ./...`: 0 issues
- `frontend` の `npm run lint`: 成功
- `frontend` の `npm run build`: 成功
- セッション詳細Playwright E2E: 47件成功
- `task test:e2e`: 75件成功、失敗なし
