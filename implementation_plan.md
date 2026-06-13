# Issue #176 実装計画

## 目的

会話タイムラインへ、AIの推論、ツール・コマンド、Web、MCP、instructions、
システムイベントをJSONLの出現順で追加する。

会話は常時展開を維持し、会話以外は初期状態で折りたたむ。展開時は、AIが参照・入力・
実行した内容と主要な結果を追跡できる表示にする。

## 公開インターフェース

既存の `SessionDetailResponse.timeline` を維持し、
`ConversationTimelineItem` を会話以外も表せる表示DTOへ拡張する。

- `kind`: `conversation`, `reasoning`, `tool`, `web`, `mcp`,
  `instructions`, `system`, `reference`
- `label`: 折りたたみ時に表示する種類名
- `role`, `body`: 会話本文。既存契約を維持する
- `timestamp`: 表示単位の代表レコードのタイムスタンプ
- `record_count`: 表示単位へ統合したレコード数
- `collapsible`: 会話以外の展開可能な項目か
- `details`: 展開時に表示するラベル・値の配列
- 既存の増分トークン、紐付け件数、最新累計

フロントエンドはDTOを再分類せず、`ConversationTimeline` 内で項目ごとの展開状態だけを
管理する。表示状態は永続化しない。

DTO変更に伴い、セッション詳細キャッシュスキーマを4へ更新する。

## 表示単位と順序

- 既存のターン分割、Reasoningペア、Tool Batch、トークン紐付けを再利用する。
- 各表示単位は代表レコードの行番号を持ち、代表行番号順でターン内へ配置する。
- 会話の重複統合は既存仕様を維持する。
- Reasoningは`agent_reasoning`と`response_item(reasoning)`を1項目へ統合する。
- Tool Batchは呼び出し、引数、対応する結果を1項目へ統合する。
- Webは検索クエリ、検索種別、閲覧先を表示する。
- MCPはサーバー名、ツール名、引数、結果を表示する。
- instructionsはbase、developer、userを区別して表示する。
- task開始・完了・中断等はsystem項目として表示する。
- ファイル参照、画像参照、コマンド完了等はreferenceまたはtool項目として表示する。
- 未知レコードもsystem項目として残し、順序から脱落させない。
- `token_count` 自体は独立項目にせず、従来どおり直前の表示単位へ集約する。

## TDDサイクル

1. トレーサー弾: 推論を同じ時間軸へ追加
   - RED: 会話の間にあるReasoningが代表レコード順で返り、トークン計測を持つことを検証する。
   - GREEN: 汎用タイムライン項目DTOとReasoning変換を最小実装する。
   - UI RED/GREEN: Reasoningが初期状態で折りたたまれ、展開すると本文と要約を確認できるようにする。
2. Tool Batchとコマンド
   - RED: 複数呼び出し、引数、対応出力、コマンド完了情報を1表示単位から確認できることを検証する。
   - GREEN: 既存Batchとcall_id対応をタイムライン詳細へ変換する。
3. WebとMCP
   - RED: 検索クエリ、閲覧先、MCPサーバー・ツール・引数・結果を順序どおり返すことを検証する。
   - GREEN: Web/MCP表示単位を追加する。
4. instructionsとsystemイベント
   - RED: base/developer/user instructionsとターン開始・完了・中断が時間軸へ含まれることを検証する。
   - GREEN: セッション先頭の疑似ターンと通常ターンへ各項目を追加する。
5. 順序、件数、トークン集約
   - RED: 異種項目を統合した後も代表行番号順で、件数・増分・累計が正しいことを検証する。
   - GREEN: 共通の整列とトークン集約を完成させる。
6. キャッシュ互換性
   - RED: スキーマ3のキャッシュが無効化されることを検証する。
   - GREEN: キャッシュスキーマを4へ更新する。
7. リファクタリング
   - 分類、詳細生成、順序決定、トークン集約を小さな公開面の内部モジュールへ整理する。
   - 会話表示、キャンバス、BottomPanel、RightPanelの回帰を確認する。

## ドキュメント

- `docs/requirements.md`: 非会話イベントの種類、初期折りたたみ、展開内容を追加する。
- `docs/detailed-design.md`: DTO、表示単位の統合規則、順序、UI状態、テストケースを更新する。
- `docs/adr/0029-conversation-timeline-dto.md`: 汎用タイムラインDTOへの拡張とキャッシュ影響を追記する。
- 新規ADR: 表示単位の分類や詳細構造が既存ADRの範囲を超える場合のみ作成する。

## 検証

- 各RED/GREENで対象Goテストまたは対象Playwrightテストを実行する。
- `go test ./...`
- `go test -tags production ./...`
- `golangci-lint run`
- `frontend` で `npm run lint`
- `task test:e2e:detail -- session-detail.spec.ts --grep "<対象テスト>"`
- 最終確認で `task test:e2e`

## 優先する振る舞い

1. 会話を含む全表示単位のJSONL順
2. ツール・コマンド・Web・MCPの入力と結果の追跡可能性
3. 初期折りたたみと展開後の情報量
4. 表示単位ごとの増分・累計トークンと件数
