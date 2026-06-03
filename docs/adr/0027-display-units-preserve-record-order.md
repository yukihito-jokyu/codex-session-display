# ADR 0027: 表示単位によるレコード出現順の維持

## ステータス

承認済み

## コンテキスト

ターン内のノードを Reasoning、Tool Batch、Agent Message などの種別別ループで生成すると、JSONL 上では別レコードを挟むノードが UI 上で隣接して接続される。

特に表示不能な Reasoning は、実データ上で Tool Batch や Agent Message を挟んでいても UI 上では連続して見え、集約対象の判定と表示結果が一致しなかった。

## 決定

ターン内の動的ノードは、既存の Reasoning ペアリングや Tool Batch 検出結果から `DisplayUnit` を構築し、代表レコードの行番号順でハーネスへ生成する。

`DisplayUnit` は、1つの UI 表示として扱うレコード群を表す。

- Reasoning ペアまたは表示不能 Reasoning グループ
- Tool Batch
- Web Search
- Item Completed
- Agent Message
- Generic Event

表示不能 Reasoning は、隣接する `DisplayUnit` が同じ表示不能 Reasoning の場合のみ集約する。`token_count` など表示ノードを生成しないレコードは `DisplayUnit` を作らないため、連続性を分断しない。

## 理由

1. JSONL の時系列と UI の接続順を一致させられる
2. 集約判定をユーザーが見る表示単位に合わせられる
3. 既存のペアリング・バッチ検出ロジックを再利用できる
4. ノード種別ごとのレイアウト責務を維持できる

## 結果

- 動的ノードは代表レコードの出現順で生成される
- Agent Message や Tool Batch を挟む Reasoning は UI 上でも分離される
- UI 上で隣接する表示不能 Reasoning は1ノードへ集約される
- FlowGraph 生成方式の変更前に作られたキャッシュは、キャッシュスキーマバージョンにより再解析される

## 関連決定

- [[ADR-0002: React Flow形式への変換をバックエンドで行う]]
- [[ADR-0004: キャッシュ形式としてReact Flow形式を採用]]
- [[ADR-0006: reasoningの二重表現の統合]]
