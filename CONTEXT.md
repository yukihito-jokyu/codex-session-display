# CONTEXT — codex-session-display

## ドメイン概要

Codex CLIのセッションログ（JSONL）を解析・可視化し、開発者がエージェントの行動分析とトークン消費分析を行えるようにするデスクトップアプリケーション。

---

## コア概念

### セッション（Session）

Codex CLIの1回の起動から終了までの期間（プロセスライフタイム）。

1個のJSONLファイル（`~/.codex/sessions/YYYY/MM/DD/rollout-{timestamp}-{uuid}.jsonl`）に対応する。

**関連用語**: [[ターン]]

### ターン（Turn）

ユーザーがAI agentに指示をしてAI agentから回答を受け取るまでの単位。

JSONLレコード上では `task_started` イベントから `task_complete`（または `turn_aborted`）イベントまでの区間に対応する。同一の `turn_id` で識別される。

1つのセッションは複数のターンで構成される。

**関連用語**: [[セッション]], [[ハーネス]]

### ハーネス（Harness）

**本質的な定義**: Codex CLI自体のこと。

**実装上の定義**: React Flowキャンバス上のメインフロー（縦方向の直列接続）を構成するノードの並び。

ハーネス上には時系列のメイン処理フロー（`session_meta` → `task_started` → ... → `task_complete`）を配置する。

**関連用語**: [[セッション]], [[ターン]], [[ハーネス外（分岐）]]

### ハーネス外（分岐）（Off-harness Branch）

ハーネスから横方向に展開されるノード群。

外部コンテキストの入力やユーザーの入力、システムプロンプトや外部へのアクションなどが含まれる。

メインフローを邪魔しないよう、`step` エッジでハーネスから分岐して表示される。

**関連用語**: [[ハーネス]], [[バッチ]]

---

## データ構造

### バッチ（Batch）

エージェントが複数のツールを連続して呼び出し、その結果をまとめて受け取るパターン。

以下の構造を持つ：
```
function_call 群 → 中間メッセージ → function_call_output 群
```

バッチサイズは1〜8以上を確認済み。同一バッチ内の `function_call` / `function_call_output` は `call_id` で1:1に対応する。

**関連用語**: [[中間メッセージ]], [[function_call]]

### 中間メッセージ（Middle Message）

`function_call` バッチと `function_call_output` バッチの間に出現するメッセージ。

エージェントがツール実行中に出力するメッセージであり、以下のパターンが存在する：

- **パターンA**: `agent_message` + `response_item(message, role=assistant)` の両方（~94%）
- **パターンB**: `response_item(message, role=assistant)` のみ（~6%）

**関連用語**: [[バッチ]]

### function_call / function_call_output

ツール呼び出しとその結果を表すJSONLレコード型。

`call_id` フィールドで1:1に紐付けられる。バッチ内では複数の `function_call` が連続し、対応する `function_call_output` もまとめて返る。

**関連用語**: [[バッチ]], [[中間メッセージ]]

---

## レコード型カテゴリ

JSONLファイルの各レコードは以下のカテゴリに分類される：

| カテゴリ | 説明 | 対象タイプ例 |
|----------|------|--------------|
| **Turn系** | ターンの区切りや構造を定義するレコード | `session_meta`, `turn_context`, `user_message`, `agent_message` |
| **メッセージ系** | LLM APIに渡されるメッセージ | `response_item(type=message)` |
| **Action系** | ツール呼び出しとその結果 | `function_call`, `function_call_output`, `custom_tool_call` |
| **思考系** | エージェントの推論プロセス | `agent_reasoning`, `response_item(type=reasoning)` |
| **イベント系** | ターン内の進捗イベント | `task_started`, `item_completed`, `task_complete` |
| **コンテキスト系** | エージェントの振る舞いを制御する指示・設定 | `base_instructions`, `developer_instructions`, `user_instructions` |
| **未知タイプ** | 将来の追加や拡張に備えるプレースホルダー | 上記以外の全て |

---

## 重要な挙動

### token_count の紐付け

`token_count` イベントは `turn_id` フィールドを持たない。

直前のイベントノード（通常は `function_call` または `agent_message`）に**バッジ**として紐付けられ、独立ノードとしては扱われない。

### reasoning の二重表現

同じ推論プロセスが2つのレコード型で表現される：

- `event_msg(type=agent_reasoning)`: 推論テキスト全文
- `response_item(type=reasoning)`: 推論サマリー

本アプリではこれらを**統合して1つのノード**として扱う。ハーネス上には `agent_reasoning` のテキストを表示し、クリック展開時には `reasoning` のサマリーも併せて表示する。

### tolerant パース

JSONLパース時に、1行のJSONパースに失敗した場合、**該当行をスキップし警告ログを出力**して解析を継続する。

不正レコードの割合が50%を超えた場合のみエラーを返す。
