# Issue #175 実装計画

## 目的

セッション詳細画面の左側に、JSONLの出現順でユーザー発言とAI発言を追える会話タイムラインを追加する。

通常ターンと疑似ターンを区別し、各発言には紐付く `last_token_usage` の合算値と件数を強調表示し、最新の `total_token_usage` をセッション累計として補助表示する。

## 公開インターフェース

`SessionDetailResponse` に `timeline` を追加し、バックエンドで表示単位を確定する。

- `ConversationTimelineTurn`
  - ターンインデックス、ターンID
  - 通常ターンか疑似ターンか
  - 所要時間
  - ターン消費トークン
  - JSONL順の `ConversationTimelineItem` 配列
- `ConversationTimelineItem`
  - ユーザーまたはAIのロール
  - 本文
  - 元レコードのタイムスタンプ
  - 紐付く `last_token_usage` の合算
  - 紐付く `token_count` 件数
  - 最新の `total_token_usage`

フロントエンドはDTOを再解釈せず、`ConversationTimeline` コンポーネントへ渡して描画する。

## 振る舞いの定義

- `event_msg(user_message/agent_message)` と `response_item(message)` からユーザー・AI発言を抽出する。
- 同一ターン内で、同じロール・本文・タイムスタンプを持つ発言は1表示単位へ統合する。
- 統合後も代表レコードの行番号順で表示する。
- 統合対象レコードのいずれかに紐付く `token_count` を、その表示単位の計測値として集約する。
- `last_token_usage` があるレコードだけを合算し、件数は値の欠落を含む紐付けレコード数とする。
- 紐付く記録がない表示単位は「計測なし」とする。
- `total_token_usage` は、表示単位に紐付く最後の値をセッション累計として表示する。
- 通常ターンは所要時間と既存 `TurnStatistics.consumed_tokens` 相当の値を表示する。
- 疑似ターンは「ターン外イベント」と表示する。
- 会話項目を持つターンだけをタイムラインDTOへ含める。

## TDDサイクル

1. トレーサー弾: 通常ターンの会話
   - RED: 実JSONLに近い入力から、ユーザー発言とAI発言がJSONL順の `timeline` として返るテストを追加する。
   - GREEN: 最小限のDTOと会話抽出を実装する。
   - UI RED/GREEN: 左ペインに通常ターンと常時展開された本文を表示するE2Eを追加して実装する。
2. 重複統合
   - RED: 同一ロール・本文・タイムスタンプの `event_msg` と `response_item` が1件になるテストを追加する。
   - GREEN: レコード群を1表示単位へ統合する。
3. 表示単位のトークン計測
   - RED: 統合対象に紐付く複数 `token_count` の増分合算、件数、最新累計を検証する。
   - GREEN: 表示単位ごとのトークン集約を実装する。
   - RED/GREEN: 「計測なし」と増分・累計表示をE2Eで検証する。
4. 疑似ターン
   - RED: ターン外の会話が疑似ターンとして順序を保って返るテストを追加する。
   - GREEN: 疑似ターンのDTO生成とUI表示を実装する。
5. キャッシュ互換性
   - RED: 旧スキーマのキャッシュが無効化されることを検証する。
   - GREEN: キャッシュスキーマを更新する。
6. リファクタリング
   - 会話抽出・重複統合・トークン集約を小さな公開面の内部関数へ整理する。
   - 既存キャンバス、BottomPanel、RightPanelの操作を回帰確認する。

## 画面構成

- `SessionDetailMainContent` を横3領域にする。
  - 左: `ConversationTimeline`
  - 中央: 既存キャンバスとBottomPanel
  - 右: 既存RightPanel
- タイムラインは固定幅のスクロール領域とし、Issue #177の幅変更・保存機能は先取りしない。
- ユーザー発言とAI発言は本文を省略・折りたたみせず表示する。

## ドキュメント

- `docs/requirements.md`: 会話タイムラインの機能要件と画面構成を追加する。
- `docs/detailed-design.md`: DTO、生成規則、状態・コンポーネント、テストケースを追加する。
- `docs/adr/0029-conversation-timeline-dto.md`: 表示単位をバックエンドで確定し、キャッシュ対象DTOへ含める決定を記録する。
- キャッシュスキーマをバージョン3へ更新する。

## 検証

- 各RED/GREENで対象Goテストまたは対象Playwrightテストを実行する。
- `go test ./...`
- `go test -tags production ./...`
- `golangci-lint run`
- `frontend` で `npm run lint`
- `task test:e2e:detail -- session-detail.spec.ts --grep "<対象テスト>"`
- 最終確認で `task test:e2e`

## 優先する検証

1. JSONL順と重複統合
2. 表示単位ごとのトークン増分・件数・累計
3. 通常ターンと疑似ターンの区別
4. 既存のキャンバス、BottomPanel、RightPanel操作の維持
