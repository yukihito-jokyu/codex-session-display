# Issue #176 実装結果

## 実装内容

- `SessionDetailResponse.timeline` を汎用タイムラインDTOへ拡張した。
- 会話、推論、ツールバッチ、コマンド、Web、MCP、instructions、system、画像参照を代表レコード行順で配置した。
- ツール呼び出しは引数と対応結果、コマンドは終了コードと出力、MCPはサーバー・引数・結果を詳細へ格納した。
- 未分類response、item完了、スレッド名更新、コラボ系外部イベントをsystem項目として保持した。
- 各表示単位へ統合レコード件数、増分トークン、紐付け件数、最新累計を付与した。
- 会話は常時展開し、会話以外は初期状態で折りたたむUIへ変更した。
- キャッシュスキーマを3から4へ更新し、Wails TypeScript bindingsを再生成した。

## TDD

以下を1テストずつRED/GREENで追加した。

1. 会話間の推論とトークン集約
2. 推論の折りたたみUI
3. Tool Batchの引数・対応結果
4. Web検索・閲覧とMCP呼び出し
5. base/developer/user instructionsとsystemイベント
6. コマンド出力と画像参照
7. その他systemイベントのフォールバック
8. スキーマ3キャッシュの無効化

## ドキュメント

- `docs/requirements.md`
- `docs/detailed-design.md`
- `docs/adr/0029-conversation-timeline-dto.md`

## 検証結果

- `go test ./... -count=1`: 137件成功
- `go test -tags production ./... -count=1`: 141件成功
- `golangci-lint run`: 0 issues
- `npm run lint`: 成功
- `npm run build`: 成功
- `task test:e2e:detail -- session-detail.spec.ts`: 48件成功
- `task test:e2e`: 76件成功、flaky 0件
