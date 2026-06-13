# ADR 0029: 会話タイムラインの表示単位をバックエンドで確定する

## ステータス

承認済み

## コンテキスト

セッション詳細画面で、`event_msg` と `response_item` に重複して記録されるUser/AI発言を
JSONL順に表示し、各発言へ `token_count` の増分と累計を対応付ける必要がある。
フロントエンドで生レコードを再解釈すると、ターン分割、重複統合、トークン紐付けが
React Flow生成処理と二重実装になる。

## 決定

- `SessionDetailResponse` に `timeline` を追加する。
- バックエンドは既存のターン分割と `BoundToRecord` を利用し、
  `ConversationTimelineTurn` と `ConversationTimelineItem` を生成する。
- 同一ターン内でロール・本文・タイムスタンプが一致する発言を1表示単位へ統合する。
- 統合対象へ紐付く全 `last_token_usage` を合算し、件数と最後の
  `total_token_usage` を表示用DTOへ格納する。
- フロントエンドはDTOを再解釈せず、JSONL順の配列をそのまま描画する。
- `ConversationTimelineItem` を会話以外にも使える表示DTOへ拡張し、`kind`, `label`,
  `record_count`, `collapsible`, `details` を追加する。
- reasoningペア、tool batch、Web、MCP、instructions、system、参照情報を代表行番号順で
  会話と同じ時間軸へ配置する。
- 会話は常時展開し、会話以外の展開状態はフロントエンド内だけで管理する。
- DTOをキャッシュ対象に含め、キャッシュスキーマをバージョン4へ更新する。

## 理由

レコードの意味解釈をGoユースケースへ集約することで、キャンバスとタイムラインの
ターン境界・トークン紐付けを一致させられる。キャッシュヒット時も再計算なしで
同じ表示を復元できる。

## 結果

- フロントエンドは表示責務に限定される。
- スキーマ3以前のキャッシュは初回表示時に再解析される。
- 今後タイムライン種別を追加する場合はDTOとキャッシュスキーマの互換性評価が必要になる。

## 関連決定

- ADR 0004: キャッシュ形式をReact Flow形式にする
- ADR 0027: 表示単位は元レコード順を保持する
- ADR 0028: ノードバッジにノード消費トークン数を表示
