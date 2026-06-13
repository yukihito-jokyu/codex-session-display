# 要件定義書 — codex-session-display

## 1. システム概要

### 1.1 目的

Codex CLIのセッションログ（JSONL）を解析・可視化し、開発者がエージェントの行動分析とトークン消費分析を行えるようにする。

### 1.2 対象ユーザー

開発者自身。

### 1.3 プラットフォーム

Wails（Go + React）によるデスクトップアプリケーション。ローカルで起動して使用する。

### 1.4 前提条件

- Codex CLI（v0.131.0）のセッションファイルが `~/.codex/sessions` に格納されていること
- 対象ファイル形式はJSONL（1行1JSONレコード）
- ローカル環境でのみ動作し、外部サーバーへのデータ送信は行わない

### 1.5 対象外事項

以下は本システムのスコープ外とする。

- **リアルタイム監視**: セッションの実行中ライブ表示は行わない（既存のJSONLファイルの解析のみ）
- **JSONLの編集・操作**: 解析対象のJSONLファイルを変更・削除しない（読み取り専用）
- **複数セッションの同時表示**: 一度に1セッションのみ表示する
- **Codex以外のツール対応**: Claude Code、Aider等の他のAIコーディングツールのセッション形式には対応しない
- **認証・認可**: ローカル単一ユーザー前提のため、ログイン機能やアクセス制御は不要
- **多言語対応**: UI言語は日本語固定

---

## 2. 機能要件

### 2.1 セッション一覧・選択

| 要件               | 内容                                                             |
| ------------------ | ---------------------------------------------------------------- |
| 自動読込           | `~/.codex/sessions` ディレクトリ配下のセッションを自動で検出する |
| 一覧表示           | セッション一覧をUIに表示する                                     |
| セッション選択     | ユーザーが特定のセッションを選択して詳細を表示する               |
| 単一セッション表示 | 複数セッションの同時表示は行わない                               |

#### ディレクトリ構造によるグループ表示

セッションファイルは `~/.codex/sessions/YYYY/MM/DD/` のディレクトリ構造で格納されているため、一覧画面では年→月→日のツリー構造でグループ化して表示する。

- 各階層（年/月/日）は折りたたみ可能とする
- 各階層のヘッダーに含まれるセッション件数を表示する
- デフォルトでは最新の年/月/日を展開済みとする

#### セッション一覧の表示項目

一覧には `session_meta` レコードから以下の情報を表示する。ただし `session_meta` の解析はJSONLファイルの読込が必要であるため、未解析のセッションでは解析が必要な項目に「解析前」と表示する。ファイルパスから取得可能な情報（タイムスタンプ等）は常に表示する。

| 表示項目           | データソース             | 解析前の表示     | 備考            |
| ------------------ | ------------------------ | ---------------- | --------------- |
| セッションID       | ファイル名から抽出       | 表示可能         | UUID            |
| 作業ディレクトリ   | `payload.cwd`            | 「解析前」       |                 |
| CLIバージョン      | `payload.cli_version`    | 「解析前」       | 例: "0.131.0"   |
| 起動元             | `payload.originator`     | 「解析前」       | 例: "codex-tui" |
| モデルプロバイダー | `payload.model_provider` | 「解析前」       | 例: "openai"    |
| ブランチ           | `payload.git.branch`     | 「解析前」       |                 |
| タイムスタンプ     | ファイルパスから推測     | 表示可能（色薄） |                 |
| ファイルサイズ     | ファイルシステム         | 表示可能         |                 |

### 2.2 JSONL解析

| 要件           | 内容                                                                              |
| -------------- | --------------------------------------------------------------------------------- |
| 解析タイミング | セッションを開いた時にJSONLを解析する                                             |
| 解析結果の保存 | 解析結果をReact Flowのnode/edge形式のJSONとして `~/.codex-display` に保存する     |
| 再解析判定     | JSONLファイルの編集日時が解析結果の保存日時より新しい場合、再解析する             |
| ファイルサイズ | 50MB程度までのファイルを想定する                                                  |
| 対象バージョン | Codex CLI v0.131.0のJSONLフォーマットに対応する。ただし柔軟に変更可能な構成とする |

#### JSONLレコード型

JSONLは各レコードのトップレベル `type` フィールドで4種類に大別される。さらに `event_msg` と `response_item` は `payload.type` フィールドで詳細分類される。`session_meta` と `turn_context` は `payload.type` を持たないため、パーサーはトップレベル `type` でのみ判定する。

**第1段階: トップレベル type による大分類**

| トップレベル `type` | `payload.type` の有無   | 概要                                                     |
| ------------------- | ----------------------- | -------------------------------------------------------- |
| `session_meta`      | なし（1件のみ）         | セッションのメタ情報（ID、CWD、モデル、Git情報等）       |
| `turn_context`      | なし（原則ターンごとに1件※） | ターンのコンテキスト（モデル、ポリシー、ユーザー指示等） |
| `event_msg`         | あり                    | イベントメッセージ。詳細は第2段階へ                      |
| `response_item`     | あり                    | レスポンスアイテム。詳細は第2段階へ                      |

※ `turn_context` は原則として1ターンにつき1件出現するが、即時中断されたターン（`task_started` 直後に `turn_aborted` が発生したケース等）では0件となる場合がある。また、万が一1ターン内に複数出現した場合は、後から出現したレコード（最新）で上書きし、パーサーは警告を出力する。

**第2段階: payload.type による詳細分類**

| トップレベル `type` | `payload.type`         | テキストフィールド                                       | 概要                                                              |
| ------------------- | ---------------------- | -------------------------------------------------------- | ----------------------------------------------------------------- |
| `event_msg`         | `user_message`         | `payload.message`                                        | ユーザーの入力                                                    |
| `event_msg`         | `agent_message`        | `payload.message`                                        | エージェントの発言（Markdown形式）                                |
| `event_msg`         | `agent_reasoning`      | `payload.text`                                           | 推論プロセスのテキスト                                            |
| `event_msg`         | `task_started`         | —                                                        | タスク開始イベント                                                |
| `event_msg`         | `item_completed`       | —                                                        | アイテム完了イベント                                              |
| `event_msg`         | `task_complete`        | `payload.last_agent_message`                             | タスク完了イベント                                                |
| `event_msg`         | `token_count`          | —                                                        | トークン使用量                                                    |
| `response_item`     | `message`              | `payload.content[].text`（`content` はオブジェクト配列） | LLM APIメッセージ（`role` で developer/user/assistant に分岐）    |
| `response_item`     | `reasoning`            | `payload.summary[].text`                                 | 推論サマリー。本文を表示できない場合は `summary=[]`, `content=null`, `encrypted_content` のみとなる |
| `response_item`     | `function_call`        | `payload.arguments`（JSON文字列）                        | ツール呼び出し（`call_id` で `function_call_output` と1:1紐付け） |
| `response_item`     | `function_call_output` | `payload.output`（文字列）                               | ツール呼び出し結果（`call_id` で対応）                            |

**主要 payload フィールドの詳細**

`session_meta`（`payload.type` なし）:

| フィールド          | 型                | 内容                                      |
| ------------------- | ----------------- | ----------------------------------------- |
| `id`                | string (UUID)     | セッションID                              |
| `cwd`               | string            | 作業ディレクトリ                          |
| `cli_version`       | string            | CLIバージョン（例: "0.131.0"）            |
| `originator`        | string            | 起動元（例: "codex-tui"）                 |
| `source`            | string            | ソース（例: "cli"）                       |
| `thread_source`     | string            | スレッドソース（例: "user"）              |
| `model_provider`    | string            | モデルプロバイダー（例: "openai"）        |
| `base_instructions` | object            | `{ "text": "..." }`（~20K文字）           |
| `git`               | object            | `{ commit_hash, branch, repository_url }` |
| `timestamp`         | string (ISO 8601) | セッション開始時刻                        |

`turn_context`（`payload.type` なし）:

| フィールド           | 型            | 内容                                                                                                     |
| -------------------- | ------------- | -------------------------------------------------------------------------------------------------------- |
| `turn_id`            | string (UUID) | ターンID                                                                                                 |
| `model`              | string        | モデル名（例: "zai/glm-5.1"）                                                                            |
| `approval_policy`    | string        | 承認ポリシー（例: "on-request"）                                                                         |
| `collaboration_mode` | object        | `{ mode, settings }` — `settings.developer_instructions`（~8K文字）を含む                                |
| `user_instructions`  | string        | ユーザー指示（~4.7K文字）                                                                                |
| `sandbox_policy`     | object        | サンドボックス設定                                                                                       |
| `effort`             | string        | エフォート（例: "medium"）                                                                               |
| `personality`        | string        | ペルソナリティ（例: "pragmatic"）                                                                        |
| その他               | —             | `permission_profile`, `file_system_sandbox_policy`, `realtime_active`, `summary`, `truncation_policy` 等 |

`event_msg.type=task_started`:

| フィールド                | 型                   | 内容                                            |
| ------------------------- | -------------------- | ----------------------------------------------- |
| `turn_id`                 | string (UUID)        | ターンID                                        |
| `started_at`              | int (Unixエポック秒) | タスク開始時刻                                  |
| `model_context_window`    | int                  | コンテキストウィンドウサイズ（例: 258400）      |
| `collaboration_mode_kind` | string               | コラボレーションモード（例: "default", "plan"） |

`event_msg.type=task_complete`:

| フィールド               | 型                   | 内容                           |
| ------------------------ | -------------------- | ------------------------------ |
| `turn_id`                | string (UUID)        | ターンID                       |
| `completed_at`           | int (Unixエポック秒) | タスク完了時刻                 |
| `duration_ms`            | int                  | 所要時間（ミリ秒）             |
| `time_to_first_token_ms` | int                  | 初回トークン生成時間（ミリ秒） |
| `last_agent_message`     | string               | エージェントの最終メッセージ   |

`event_msg.type=item_completed`:

| フィールド        | 型               | 内容                                                                    |
| ----------------- | ---------------- | ----------------------------------------------------------------------- |
| `thread_id`       | string           | セッションID                                                            |
| `turn_id`         | string (UUID)    | ターンID                                                                |
| `item`            | object           | `{ id, text, type }` — Plan型の場合、`item.text` にプラン全文が含まれる |
| `completed_at_ms` | int (Unixミリ秒) | 完了時刻                                                                |

`event_msg.type=token_count`:

| フィールド    | 型                    | 内容                                                            |
| ------------- | --------------------- | --------------------------------------------------------------- |
| `info`        | object                | `{ last_token_usage, model_context_window, total_token_usage }` |
| `rate_limits` | null または存在しない | `info` と同階層（`info` の中ではない）                          |

`response_item.type=message`:

| フィールド | 型               | 内容                                                       |
| ---------- | ---------------- | ---------------------------------------------------------- |
| `role`     | string           | `"developer"` / `"user"` / `"assistant"`                   |
| `content`  | オブジェクト配列 | `[{ "type": "input_text"\|"output_text", "text": "..." }]` |

`response_item.type=function_call`:

| フィールド  | 型            | 内容                           |
| ----------- | ------------- | ------------------------------ |
| `name`      | string        | ツール名（例: "exec_command"） |
| `arguments` | string (JSON) | ツール引数のJSON文字列         |
| `call_id`   | string        | `"call_" + UUID` 形式のID      |

`response_item.type=function_call_output`:

| フィールド | 型     | 内容                              |
| ---------- | ------ | --------------------------------- |
| `call_id`  | string | 対応する `function_call` と同じID |
| `output`   | string | 実行結果文字列                    |

### 2.3 可視化（React Flow）

#### 2.3.1 ハーネス構造

縦方向のメインフロー（ハーネス）を中心に、コンテキスト情報が分岐するツリー構造で表示する。

全レコードの実データは `payload` キーの下にネストされている。本節のパス表記は `payload` を省略するが、実装時は全て `payload` 配下として扱うこと。

実際のJSONLでは以下の特徴がある:

- `task_started` / `task_complete` はターン（turn）の境界を示す。1ターン = 1つの `task_started` 〜 `task_complete` のペアで、それぞれ同一の `turn_id` を持つ
- `task_started` はセッション開始直後（`session_meta` の直後）と、各 `task_complete` の直後に出現する
- `task_started` 〜 `task_complete` の枠外（セッション開始直後、ターン間、セッション終了後）に出現するレコードや、対応する `task_started` が欠落した「孤立した `task_complete` / `turn_aborted`」などのレコードは、**「疑似ターン（Pseudo-Turn）」**としてグループ化し、ハーネス（メインフロー）上に時系列順で縦方向に配置してエッジで接続する。
- `turn_context` の前後に出現する `response_item(message)` の順序・件数はターンによって変動する（例: developer → user の場合と user → developer の場合がある）
- 複数の `function_call` がバッチで連続出力され、対応する `function_call_output` もバッチで返る
- `function_call` バッチと `function_call_output` バッチの間に `agent_message` + `response_item(message, role=assistant)` が挟まる
- `response_item(type=message)` が `turn_context` の前後にも出現する
- `item_completed` は常に出現するとは限らない（出現頻度が極めて低い場合がある）。また出現位置も固定ではなく、reasoning と agent_message の間に出現することもある
- `token_count` は `turn_id` を持たず、出現位置から属するターンを判定する。`token_count` は直前のイベント（通常は `function_call` または `agent_message`）に紐付けて表示する（§2.3.8参照）


**基本パターン（実際の出現順序）:**

```
[session_meta] ← セッション開始（cli_version, model_provider等）
     │
[task_started] ← Turn 1 開始: started_at, model_context_window, collaboration_mode_kind
     │
[response_item(message, role=developer)] ← ハーネス外（分岐）
     │
[response_item(message, role=user)] ← ハーネス外（分岐）
     │
[turn_context] ── [base_instructions（~20K文字）]
     │            ── [developer_instructions（~8K文字）]
     │            ── [user_instructions（~4.7K文字）]
     │
[response_item(message, role=user)] ← ハーネス外（分岐）
     │
[user_message（event_msg）] ← ハーネス外（分岐）
     │
[agent_reasoning] ← ハーネス上、詳細はクリック展開
     │
[function_call ×N] ─┐ ← token_count は最後のfunction_callに紐付くバッジとして表示
     │              │
[agent_message（event_msg）] ← バッチ中間に挟まる
[response_item(message, role=assistant)] ← 同上
     │              │
[function_call_output ×N] ─┘
     │  ←───────────┘
     │     （最後のoutputからハーネスへ戻る矢印）
[item_completed] ← item詳細、completed_at_ms
     │
     ...（バッチ繰り返し）
     │
[task_complete] ← Turn 1 終了: completed_at, duration_ms, time_to_first_token_ms
     │
     ...（次ターン: task_started → task_complete から繰り返し）
```

**バッチ実行パターンの詳細:**

`function_call` バッチと `function_call_output` バッチの間に、エージェントの中間メッセージが挟まる。バッチサイズは1〜8以上を確認済み。中間メッセージには以下の2パターンが存在する:

| パターン                                    | 構成                           | 割合 |
| ------------------------------------------- | ------------------------------ | ---- |
| A: agent_message + response_item(assistant) | 両方出現                       | ~94% |
| B: response_item(assistant) のみ            | 空テキスト、agent_message なし | ~6%  |

パーサーは両パターンを処理可能にすること。

```

パターンA（最頻出）: バッチサイズ1の例
[function_call] ──────────────┐
[agent_message（event_msg）] ← ハーネス上
[response_item(message, asst.)] ← ハーネス上
[function_call_output] ───────┘

```

```

パターンB: バッチサイズ1の例
[function_call] ──────────────┐
[response_item(message, asst.)] ← 空テキスト、ハーネス上
[function_call_output] ───────┘

```

```

パターンA: バッチサイズ3の例
[function_call] ──────────────────┐
[function_call] ──────────────────┤ 分岐ノード
[function_call] ──────────────────┘
[agent_message（event_msg）] ← ハーネス上
[response_item(message, asst.)] ← ハーネス上
[function_call_output] ───────────┐
[function_call_output] ───────────┤ 分岐ノード
[function_call_output] ───────────┘

```

- ハーネス上には時系列のメイン処理フローを配置する
- 分岐ノードはハーネスから横方向に展開する
- `function_call` バッチと `function_call_output` バッチは別々に検出する（call群 → 中間メッセージ → output群の順序で混在しない）
- `function_call` / `function_call_output` はバッチ単位でグループ化して表示する
- `function_call_output` の最後からハーネス方向へ戻る矢印を描画し、処理の流れを明確にする
- 未知のレコード型（`custom_tool_call` 等）は generic node としてハーネス上に表示する

#### 2.3.2 ハーネス上のノード

- `session_meta`
- `task_started`（`event_msg.type=task_started`）— ターン開始
- `turn_context`
- `agent_reasoning`（`event_msg.type=agent_reasoning`）
- `agent_message`（`event_msg.type=agent_message`）— `function_call` バッチと `function_call_output` バッチの間にも出現
- `response_item(message, role=assistant)` — `agent_message` と同位置に出現
- `item_completed`（`event_msg.type=item_completed`）
- `task_complete`（`event_msg.type=task_complete`）— ターン終了

#### 2.3.2.1 ターン外ノードおよび孤立イベントの表示

ハーネス（メインフロー）上に配置されるターン外ノード（疑似ターン内のノード）は、以下のルールに従って表示・スタイリングする。

- **通常のターン外レコード（グローバル設定、システムイベント等）**:
  - 対象ノード: `sessionMeta` 以外のターン外で出現する設定やシステムイベント等
  - 表示内容: ラベルの先頭に `[System]` プレフィックスを付与する（例: `[System] thread_name_updated`）。
  - スタイル: 通常のノードと区別するため、ボーダーを**破線（dashed border）**にし、背景色をトーンダウンしたグレー系にする。
  - データ: ターン外であることを示すため、`turnIndex` は `null` または `-1` とする。

- **孤立したターン終了イベント（task_started なしの task_complete / turn_aborted）**:
  - 対象ノード: 対応する `task_started` が欠落した状態で出現した `task_complete` または `turn_aborted` レコード
  - 表示内容: ラベルを `"Orphan Complete / 孤立した完了"` または `"Orphan Aborted / 孤立した中断"` とし、ノードのアイコンには `⚠️`（または `✕`）を表示する。
  - スタイル: ボーダーを赤色またはオレンジ色の警告色にする。
  - 詳細表示: ノードクリックで下部パネル（BottomPanel）を開いた際、「対応する task_started が存在しない孤立した完了イベントです」という警告メッセージを表示する。

#### 2.3.3 ハーネス外（分岐）のノード


- `response_item(message, role=developer)` — `task_started` と `turn_context` の前後に出現
- `response_item(message, role=user)` — `turn_context` の前後に出現
- `user_message`（`event_msg.type=user_message`）
- `base_instructions`（`session_meta` → `payload.base_instructions.text`）
- `developer_instructions`（`turn_context` → `payload.collaboration_mode.settings.developer_instructions`）
- `user_instructions`（`turn_context` → `payload.user_instructions`）
- `reasoning`（`response_item.type=reasoning`）
- `function_call`（`response_item.type=function_call`）
- `function_call_output`（`response_item.type=function_call_output`）

#### 2.3.4 ノードの種類

| カテゴリ       | 対象タイプ                                                                                                              | 備考                                                        |
| -------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Turn系         | `session_meta`, `turn_context`, `user_message`, `agent_message`                                                         | セッション・ターンの区切り                                  |
| メッセージ系   | `response_item(message, role=developer)`, `response_item(message, role=user)`, `response_item(message, role=assistant)` | LLM APIのメッセージ。roleにより配置先が異なる（§2.3.7参照） |
| Action系       | `function_call`, `function_call_output`, `custom_tool_call`, `custom_tool_call_output`                                  | ツール呼び出しと結果。バッチで連続発生する                  |
| 思考系         | `reasoning`, `agent_reasoning`                                                                                          | 同じ推論の2つの表現（§2.3.5参照）                           |
| イベント系     | `task_started`, `item_completed`, `task_complete`                                                                       | ターン進捗。`item_completed` はPlan完了情報を含む           |
| コンテキスト系 | `base_instructions`, `developer_instructions`, `user_instructions`                                                      | ハーネスから分岐。長文のため折りたたみ表示                  |
| 未知タイプ     | その他の `payload.type`                                                                                                 | generic node としてハーネス上に表示                         |

#### 2.3.5 reasoning の二重表現の扱い

`reasoning` と `agent_reasoning` は同じ推論プロセスの2つの表現である。

| ソース                           | 型                | 内容                                                             | 用途                         |
| -------------------------------- | ----------------- | ---------------------------------------------------------------- | ---------------------------- |
| `event_msg.type=agent_reasoning` | `text` フィールド | 推論のテキスト全文                                               | **ハーネス上のノード表示用** |
| `response_item.type=reasoning`   | `summary` 配列    | 推論のサマリー。本文を表示できない場合は暗号化内容のみを持つ     | **クリック時の詳細展開用**   |

両方を統合して1つのノードとして扱い、ハーネス上に配置する。ノードには `agent_reasoning` のテキストを表示し、クリック展開時には `reasoning` のサマリーも併せて表示する。

`agent_reasoning` がなく、`response_item(type=reasoning)` の `summary` も空で本文を表示できない場合は、ノード本文を `（暗号化済み・表示不可）×N` とする。表示不能な reasoning が UI 上で連続する場合は1ノードに集約する。`token_count` など表示ノードを生成しないレコードは連続性を分断せず、Agent Message や Tool Batch など表示ノードを生成するレコードがある場合は別グループとして扱う。`agent_reasoning` または非空の `summary` を持つ reasoning は集約しない。

ターン内の動的ノードは、JSONL レコードの出現順を維持してハーネス上へ配置する。

#### 2.3.6 ノードの詳細表示

| ノード種別                                               | 表示方法                                                          |
| -------------------------------------------------------- | ----------------------------------------------------------------- |
| ツール呼び出し (`function_call`, `function_call_output`) | ノードとして表示し、クリックで詳細を展開                          |
| 推論 (`reasoning` / `agent_reasoning`)                   | ハーネス上の1ノードに統合表示し、クリックで詳細展開（§2.3.5参照） |
| `base_instructions` (~20K文字)                           | 折りたたみ可能。デフォルトは非表示                                |
| `developer_instructions` (~8K文字)                       | 折りたたみ可能。デフォルトは非表示                                |
| `user_instructions` (~4.7K文字)                          | 折りたたみ可能。デフォルトは非表示                                |
| 未知タイプ                                               | generic node として表示。クリックで生データを展開                 |

#### 2.3.7 response_item(type=message) の role 扱い

`response_item.type=message` はセッション中に60件程度出現し、3種類の `role` が存在する。出現位置と件数は以下の通り。

| role        | 件数                         | 出現位置                                                   | ハーネス上の扱い                                    |
| ----------- | ---------------------------- | ---------------------------------------------------------- | --------------------------------------------------- |
| `developer` | 2                            | `task_started` と `turn_context` の前後                    | **ハーネス外（分岐）** のコンテキストノード         |
| `user`      | セッション・ターンにより変動 | `task_started` と `turn_context` の前後                    | **ハーネス外（分岐）** のノード                     |
| `assistant` | 55                           | `function_call` バッチと `function_call_output` バッチの間 | **ハーネス上** のノード（`agent_message` と同位置） |

`event_msg` の `user_message` / `agent_message` と `response_item` の `message(role=user/assistant)` は内容が異なる場合がある:

- `message(role=user)` には AGENTS.md のようなコンテキスト情報が含まれる一方、`event_msg.user_message` はユーザーの実際の入力のみを含む
- `message(role=assistant)` はバッチ中間に挟まる場合、空テキストであることが多い

ハーネス上では `event_msg` の表現をメインとし、`response_item` 側は補助情報として扱う。

#### 2.3.8 token_count の直前イベント紐付け

`token_count`（`event_msg.type=token_count`）はハーネス上の独立ノードとはせず、**直前のイベントに紐付くバッジ**として表示する。

- `token_count` は `turn_id` フィールドを持たない。出現位置から属するターンを判定する（`task_started` 〜 `task_complete` の間に出現する `token_count` はそのターンに属する）
- 直前のイベントノード（通常は `function_call` または `agent_message`）の右側にトークンバッジを表示する
- バッジには、当該ノードに紐付く全 `info.last_token_usage.total_tokens` の合計をノード消費トークン数として表示する
- 同一ノードに複数件が紐付く場合は `×N` による件数表示を維持する
- `last_token_usage` が欠落したレコードは合計へ加算しないが、件数には含める
- バッジ用APIは `tokenBadge.consumedTokens` とし、セッション累計を表す `statistics.total_tokens` と区別する
- バッジをクリックすると下部パネルに内訳（input/output/cached/reasoning）を展開する

### 2.4 トークン使用量

トークン使用量は2つの表示方式を提供する。いずれもReact Flowキャンバス外の右パネルに表示する。

データソースは `event_msg.type=token_count` の `info` フィールド。

#### トークンデータの内訳

`token_count` イベントの `info` フィールドに含まれる項目:

| フィールド                  | 型                | 内容                                 |
| --------------------------- | ----------------- | ------------------------------------ |
| `info.last_token_usage`     | オブジェクト      | 直近（1回分）のトークン使用量        |
| `info.total_token_usage`    | オブジェクト      | セッション累計トークン使用量         |
| `info.model_context_window` | int               | モデルのコンテキストウィンドウサイズ |
| `info.rate_limits`          | null または不存在 | フィールド自体が存在しない場合がある |

**`last_token_usage` / `total_token_usage` の内訳:**

| サブフィールド            | 内容                         |
| ------------------------- | ---------------------------- |
| `input_tokens`            | 入力トークン数               |
| `cached_input_tokens`     | キャッシュ済み入力トークン数 |
| `output_tokens`           | 出力トークン数               |
| `reasoning_output_tokens` | 推論出力トークン数           |
| `total_tokens`            | 合計トークン数               |

#### 2.4.1 ターンごとのトークン使用量

各ターンのトークン消費量をサマリー表示する。

- 各ターンの消費量 = そのターンに属する最後の `token_count` の `total_token_usage` − 前ターンの最後の `token_count` の `total_token_usage`（Turn 1の場合は 0 を減算）
- ターンごとに以下を表示:
  - 消費トークン合計（`total_tokens`）
  - `input_tokens` / `output_tokens` / `reasoning_output_tokens` の内訳
  - ターン内の `token_count` 出現回数
  - ターンの collaboration_mode（plan / default 等）

#### 2.4.2 token_countごとのトークン使用量

各 `token_count` イベントの `last_token_usage` を行列表で表示する。

- 各行に `last_token_usage` の内訳を表示: `total_tokens`, `input_tokens`, `output_tokens`, `cached_input_tokens`, `reasoning_output_tokens`
- 行頭に連番とターン番号を表示
- ターン境界をセパレータで区切る（Turn 1 / Turn 2 ...）
- フッターにセッション累計（最後の `token_count` の `total_token_usage`）を表示
- リストはスクロール可能とする
- `bound_to_node_id` が現在のセッションに存在するノードを指す行、および
  `Last Token Consumption per Index` の Total / Input / Output / Reasoning /
  Cached の各データ点は操作可能とする
- 操作時は対応ノードを選択してトークン詳細パネルを開き、800ms のアニメーション・
  最大1.2倍で対象ノードを中央へズームする。同じ項目の連続操作でも毎回実行する
- `bound_to_node_id` がない項目、または参照先ノードが現在のセッションに存在しない
  項目は操作可能な表示にせず、操作時のエラー通知も行わない

### 2.5 統計情報・グラフ表示

セッション全体の以下の統計情報を表示する。

| 項目                         | 内容                               | データソース                                                                                  |
| ---------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------- |
| 所要時間                     | セッションの開始から終了までの時間 | `task_started.started_at`（Unixエポック秒） 〜 `task_complete.completed_at`（Unixエポック秒） |
| 総トークン数                 | 累計トークン使用量                 | 最後の `token_count.info.total_token_usage.total_tokens`                                      |
| ツール呼び出し回数           | `function_call` の発生回数         | `response_item.type=function_call` の件数                                                     |
| token_count回数              | `token_count` イベントの総数       | `event_msg.type=token_count` の件数                                                           |
| コンテキストウィンドウサイズ | モデルのコンテキスト上限           | `task_started.model_context_window`                                                           |
| ターン数                     | ターンの数                         | `task_started` の件数                                                                         |
| ターン所要時間               | 各ターンの実行時間                 | `task_complete.duration_ms`                                                                   |

#### 統計グラフ表示

セッション詳細画面（右パネルなど）において、セッション全体の推移を把握するための以下のインタラクティブなグラフ（チャート）を表示する。

- **ターンごとのトークン消費量グラフ**:
  - 横軸: ターン番号（Turn 1, Turn 2...）
  - 縦軸: 当該ターンでの消費トークン数（`input_tokens`, `output_tokens`, `reasoning_output_tokens` の内訳を積み上げ棒グラフ等で表現）
- **ターンごとのツール呼び出し回数グラフ**:
  - 横軸: ターン番号
  - 縦軸: ツール呼び出し回数（棒グラフまたは折れ線グラフで表現）

### 2.6 エクスポート（保留）

※本機能は実装優先度および難易度のトレードオフにより、一旦保留（スコープ外）とする（ADR 0016参照）。

| 要件     | 内容                                         |
| -------- | -------------------------------------------- |
| 画像出力 | (保留) 可視化結果を画像としてエクスポート可能にする |

### 2.7 セッション比較（nice to have）

複数セッション間の比較機能を将来実装する可能性がある。必須要件ではない。

### 2.8 ユースケース

#### UC-1: セッション一覧の閲覧

1. アプリケーションを起動する（Wailsデスクトップアプリ）
2. `~/.codex/sessions` のセッション一覧が自動表示される
3. セッションID、ブランチ、タイムスタンプ等を確認する

#### UC-2: セッションのハーネス表示

1. セッション一覧から特定のセッションを選択する
2. JSONLが解析され、React Flowのハーネス表示が表示される
3. ハーネス上のノードをクリックして詳細（ツール呼び出し内容、推論テキスト等）を確認する
4. 分岐ノード（コンテキスト情報等）を展開して内容を確認する
5. ノードに紐付いたtoken_countバッジをクリックしてトークン内訳を確認する

#### UC-3: トークン消費の分析

1. セッション詳細画面の右パネルで統計情報（所要時間、総トークン数、ツール呼び出し回数等）を確認する
2. ターンごとのトークン使用量サマリーで各ターンの消費量を確認する
3. token_countごとの行列表で各API呼び出しラウンドのトークン消費量を確認する

#### UC-4: 解析結果のエクスポート（保留）

※本ユースケースは一旦保留（スコープ外）とする。

### 2.9 画面構成

#### 画面一覧

| 画面               | 内容                                                       |
| ------------------ | ---------------------------------------------------------- |
| セッション一覧画面 | `~/.codex/sessions` のセッション一覧を表示                 |
| セッション詳細画面 | 選択したセッションの会話タイムライン、ハーネス表示、トークングラフ、統計情報 |

#### 画面遷移

```

[起動] → [セッション一覧画面] → セッション選択 → [セッション詳細画面]
↑ │
└── 一覧に戻る ←─────┘

```

#### セッション詳細画面のパネル構成

```

┌──────────────────────────────────────────────────────────────────────┐
│ ヘッダー（セッションID、ブランチ、タイムスタンプ等）                 │
├──────────────────┬──────────────────────────────┬────────────────────┤
│ 会話タイムライン │ React Flowキャンバス          │ 統計・トークン分析 │
│ - ターン区切り   │ （ハーネス表示）              │                    │
│ - User/AI本文    ├──────────────────────────────┤                    │
│ - 増分/累計      │ ノード/token_count詳細パネル  │                    │
└──────────────────┴──────────────────────────────┴────────────────────┘

```

- 会話タイムライン: 左側。`event_msg(user_message/agent_message)` と `response_item(message)` をJSONL順で常時展開表示する
  - 通常ターンは所要時間とターン消費トークンを表示し、疑似ターンは「ターン外イベント」と表示する
  - 同一ターン内の同一ロール・本文・タイムスタンプの発言は1件へ統合する
  - 発言に紐付く全 `last_token_usage` を合算し、紐付け件数と最新 `total_token_usage` を表示する。紐付けがない場合は「計測なし」と表示する
- React Flowキャンバス: メイン領域。ズーム・パン操作可能。token_countは直前のイベントノードにバッジとして表示
- 統計情報パネル: 右側上部。固定表示
- ターンごとのトークン使用量: 右側中部。各ターンの消費トークン内訳カード
- token_countごとの使用量: 右側下部。行列表で各token_countのlast_token_usageを表示
- ノード詳細パネル: 下部。ノード/token_countバッジクリック時に展開

---

## 3. 非機能要件

### 3.1 基本要件

| 項目               | 要件                                               |
| ------------------ | -------------------------------------------------- |
| データ配置         | ローカル完結。外部サーバーへのデータ送信を行わない |
| 解析結果保存先     | `~/.codex-display`                                 |
| ファイルサイズ上限 | 50MB程度                                           |
| 機密情報           | マスキング機能は不要                               |
| Codexバージョン    | v0.131.0固定。柔軟に変更可能な構成とする           |

### 3.2 性能目標

| 項目                           | 目標値    |
| ------------------------------ | --------- |
| 50MBファイルの解析所要時間     | 10秒以内  |
| キャッシュ済みセッションの表示 | 1秒以内   |
| React Flowキャンバスの初期描画 | 2秒以内   |
| ノードクリック時の詳細展開     | 500ms以内 |

### 3.3 エラーケース

| ケース                              | 挙動                                                                                             |
| ----------------------------------- | ------------------------------------------------------------------------------------------------ |
| JSONLファイルが破損している         | エラーメッセージを表示（"セッションファイルの解析に失敗しました"）。パースできた部分まで表示する |
| ファイルサイズが50MBを超過          | エラーメッセージを表示（"ファイルサイズが上限を超えています"）。解析を中止する                   |
| セッションが空（0バイト）           | エラーメッセージを表示（"セッションファイルが空です"）                                           |
| `~/.codex/sessions` が存在しない    | 起動時にディレクトリを作成するか、エラーメッセージを表示して終了する                             |
| 不正なJSONレコードが含まれる        | 該当レコードをスキップし、警告を表示する。解析は継続する                                         |
| 未知のレコード型が含まれる          | generic node として表示する。解析は継続する                                                      |
| `~/.codex-display` への書き込み失敗 | エラーログを出力し、キャッシュなしで動作を継続する                                               |

#### 3.4 テーマ・カラーパレット要件

##### 3.4.1 テーマ切替のサポート
- アプリケーションは、ダークモード（Dark Mode）とライトモード（Light Mode）の両方をサポートする。
- ユーザーはUI上の切り替えボタン等を用いて、動的にテーマを切り替えることができる。
- 切り替えられたテーマ設定はローカル（localStorage等）に保存され、次回の起動時にも引き継がれる。
- デフォルトテーマは、開発者が最も慣れ親しんでいるダークモードとする。

##### 3.4.2 ノードカテゴリのカラーパレット
各ノードはその役割（セマンティック）に応じた配色を行い、視覚的な識別性を向上させる。ダークモードとライトモードのそれぞれの配色は以下の通りとする。

| カテゴリ | 対象ノード | ダークモード (背景 / 枠・文字) | ライトモード (背景 / 枠・文字) | 役割・意味合い |
| :--- | :--- | :--- | :--- | :--- |
| **Meta & Context** | `sessionMeta`, `turnContext`, `contextDoc` | 深い緑系 (`#1c2d1c` / `#3fb950`) | 薄い緑系 (`#e8f5e9` / `#2e7d32`) | 設定・システム前提情報（静的コンテキスト） |
| **Inputs** | `userMessage`, `developerMessage`, `userApiMessage` | 深い青系 (`#0d223a` / `#58a6ff`) | 薄い青系 (`#e3f2fd` / `#1565c0`) | ユーザー・開発者からの入力 |
| **Outputs** | `agentMessage` (Assistant) | 深い青紫系 (`#131c38` / `#79c0ff`) | 薄い紫系 (`#f5f3ff` / `#6d28d9`) | エージェントの発話・最終結果 |
| **Thinking** | `reasoning` | 深い紫系 (`#241935` / `#bc8cff`) | 薄い紫系 (`#faf5ff` / `#7e22ce`) | エージェントの内部推論プロセス（思考） |
| **Actions** | `action` (ツール呼出/結果), `webSearchAction`, `externalEvent` | 深い黄・橙系 (`#2e1f0a` / `#e3b341`) | 薄い黄・橙系 (`#fff8e1` / `#b78103`) | ツール実行・外部アクション |
| **System Events** | `taskEvent` (正常開始/完了), `itemCompleted` | 深い灰系 (`#1c2128` / `#8b949e`) | 薄い灰系 (`#f6f8fa` / `#57606a`) | 進捗・システム管理イベント |
| **Warnings (例外)** | `taskEvent` (孤立した完了やエラー等) | 深い赤系 (`#3c181c` / `#f85149`) | 薄い赤系 (`#ffebee` / `#c62828`) | 孤立完了やエラー、異常系イベント |
| **Default / Unknown** | `generic` | 暗いニュートラル (`#161b22` / `#8b949e`) | 明るいニュートラル (`#f6f8fa` / `#57606a`) | 未知のレコード型 |

---

## 4. アーキテクチャ概要

### 4.1 構成

| レイヤー       | 技術                | 役割                                               |
| -------------- | ------------------- | -------------------------------------------------- |
| アプリフレーム | Wails               | Go + Reactのデスクトップアプリフレームワーク       |
| フロントエンド | React + React Flow  | JSONL解析結果の可視化                              |
| バックエンド   | Go                  | JSONLの読込・解析・React Flow形式への変換・IPC提供 |
| データソース   | `~/.codex/sessions` | CodexセッションJSONLファイル                       |
| キャッシュ     | `~/.codex-display`  | 解析結果JSON（React Flow形式）                     |

### 4.2 Goバックエンドの役割

- `~/.codex/sessions` ディレクトリのスキャン
- JSONLファイルの読込・パース
- JSONLレコードからReact Flowのnode/edgeへの変換
- 解析結果のJSON保存・読込（`~/.codex-display`）
- 再解析判定（JSONLと解析結果JSONの日時比較）
- ReactフロントエンドへのIPC提供（Wailsバインディング経由）

### 4.3 データフロー

```

~/.codex/sessions/_.jsonl
│
▼
Go Backend（解析）
│
▼
~/.codex-display/_.json（React Flow形式）
│
▼
React Frontend（React Flow表示）

```

### 4.4 IPCメソッド仕様

GoバックエンドはWailsのバインディング生成を通じてReactフロントエンドにIPCメソッドを提供する。`wails generate module` によりTypeScript型付き関数が自動生成される。

#### ListSessions()

セッション一覧を取得する。ディレクトリ構造（年/月/日）でグループ化して返す。

**戻り値:** `SessionSummary` の配列

- `parsed`: 解析済みかどうか。`false` の場合、`cwd` / `cli_version` / `originator` / `model_provider` / `branch` は `null`
- フロントエンドは `parsed === false` の項目に「解析前」と表示する

#### GetSessionDetail(id)

指定セッションの解析結果（React Flow形式）を取得する。キャッシュが存在しない、JSONLが新しい、またはキャッシュスキーマバージョンが現行と異なる場合は再解析する。

**引数:** `id` — セッションID（UUID）

**戻り値:** `SessionDetailResponse`（nodes, edges, statistics, token_counts, timeline）

#### ExportStatsImage(id)（保留）

※本メソッドは一旦保留（実装対象外）とする。

#### エラー形式

全メソッドで共通の `AppError` 構造体:

```json
{
  "code": "FILE_READ_ERROR",
  "message": "セッションファイルの読み込みに失敗しました"
}
```

| code                 | 内容                                                     |
| -------------------- | -------------------------------------------------------- |
| `SESSION_NOT_FOUND`  | 指定IDのセッションが存在しない                           |
| `FILE_TOO_LARGE`     | ファイルサイズが50MBを超過                               |
| `UNSUPPORTED_FORMAT` | 非対応のセッションファイル形式（Codex CLI v0.121.0未満） |
| `PARSE_ERROR`        | JSONLの解析に失敗（破損・形式不正）                      |
| `FILE_READ_ERROR`    | ファイルの読み込みに失敗                                 |
| `IMAGE_GEN_ERROR`    | (保留) 画像の生成に失敗                                         |
| `INTERNAL_ERROR`     | 内部エラー                                               |

---

## 5. テスト要件

### 5.1 テストアプローチ
本システムでは、デスクトップアプリ（Wails）としての実行環境依存を分離し、堅牢性と開発生産性を両立させるために以下のテスト戦略を採用する。詳細は [ADR 0020](file:///Users/yukihito/Documents/github_projects/codex-session-display/docs/adr/0020-testing-strategy.md) を参照。

1. **Goバックエンドのユニットテスト**: ビジネスロジックの完全な検証と高いカバレッジの維持。
2. **Reactフロントエンドのユニットテスト**: 共通のロジック（Hooks、ユーティリティ）の検証。
3. **フロントエンド単体E2Eテスト**: ブラウザ（Playwright）による、Wailsバインディングをモック化した画面・インタラクション検証。

### 5.2 Goバックエンドのテスト要件
- **対象範囲**: `internal/` 配下のすべてのロジック（`domain`, `usecase`, `repository`）。
- **目標値**: ユニットテストによるテストカバレッジ **80%以上** を維持する。
- **検証項目**:
  - `session_fs`: 正常なJSONLのパースおよび、破損したJSONL・対応バージョンの判定・フォールバック処理が仕様通りに行われること。
  - `get_session_detail`: パースデータから React Flow 用のノード・エッジが正しく組み立てられ、正しいレイアウト座標が計算されること。

### 5.3 フロントエンドのユニットテスト要件
- **対象範囲**: 共通Hooks（Debounce処理、展開状態管理など）および共通ユーティリティ関数（日付フォーマット、バイトサイズ変換など）。
- **除外対象**: `FlowCanvas`（React Flow依存部分）や各画面（ListPage, DetailPage）の描画ロジックは、JSDOM上でのモック作成コストが高いため、ユニットテストの対象外としE2Eテストで検証する。

### 5.4 E2Eテスト（Playwright）要件
- **実行方法**: バックエンド（Go）は起動せず、フロントエンドをVite開発サーバーで起動し、GoからバインドされるAPI（`window.go`）をPlaywright上でモック化して検証を行う。
- **検証シナリオ**:
  - **一覧画面**: 年/月/日ツリーの展開・折りたたみ動作、検索入力時のDebounceフィルタリング動作。
  - **詳細画面**: 戻るボタンによる画面遷移、ノードクリックによるBottomPanelのスライドイン表示、token_countバッジクリックによる左右分割表示とドラッグによる幅調整。

### 5.5 テストデータの管理要件
- **整合性の維持**: バックエンドのユニットテストで使用する静的JSONLテストファイル（`internal/repository/testdata/`）と、フロントエンドのE2Eテストで使用するモックJSONデータ（`frontend/tests/mocks/`）は、同一のセッションデータを表すものとして一貫性を保って管理しなければならない。

---

## 6. ロギング要件

### 6.1 ログレベルと運用基準
バックエンドの動作状況や異常検知を適切に行うため、以下の4つのログレベルを定義する。詳細は [ADR 0021](file:///Users/yukihito/Documents/github_projects/codex-session-display/docs/adr/0021-logging-strategy.md) を参照。

- **DEBUG**: 開発時のみ有効化される詳細なデバッグ情報（パース進捗、内部計算など）。
- **INFO**: アプリの起動・終了、セッション読み込み/パース成功、キャッシュの読み込み成功などの正常イベント。
- **WARN**: 動作に問題はないが、入力ファイルや連携データの不整合等がある場合の警告（パース時の破損行スキップ、`call_id` の不整合に伴うインデックス紐付けフォールバック、`turn_context` 重複上書き、`session_meta` 欠損時の推測処理など）。
- **ERROR**: 主要機能の失敗や致命的なエラー（パースの強制中断、ファイルアクセス失敗、キャッシュ書き込み/読み込み失敗など）。

### 6.2 ログの出力先とフォーマット
動作環境に応じた出力先および形式を採用する。

- **開発環境**: 標準エラー出力 (stderr) に人間が可読なテキスト形式で出力する。
- **本番環境（パッケージ版）**: 標準エラー出力 (stderr) およびローカルファイル（`~/.codex-display/logs/app.log`）にJSON形式で構造化出力する。

### 6.3 警告ログにおける情報保護とパフォーマンス
パースエラー時など、ログに詳細情報を含める際は以下の要件を満たすこと。
- パースに失敗した不正なJSON行の情報をログに残す場合、ログの肥大化およびトークンやパスコード等の機密情報の流出を防ぐため、該当行の**先頭100文字までのプレビュー出力**に制限する。

---

## 7. エラー監視・クラッシュレポート要件

### 7.1 外部監視サービスの不使用とローカル完結要件
「§1.4 前提条件」および「§3.1 基本要件」で定義されたローカル完結・外部送信禁止ポリシーに基づき、SentryやFirebase Crashlyticsなどの自動外部送信型エラー監視・クラッシュレポートサービスは一切導入しない。詳細は [ADR 0022](file:///Users/yukihito/Documents/github_projects/codex-session-display/docs/adr/0022-error-monitoring-and-crash-reporting-strategy.md) を参照。

### 7.2 エラー確認手段とフロントエンド導線支援
外部自動送信を行わない代替手段として、以下のローカル確認手段を提供する。
- **ログとUIエラー表示**: 動作エラー時はバックエンドによるログ記録（`~/.codex-display/logs/app.log`）を行うとともに、フロントエンドUIに分かりやすいエラーメッセージを表示する。
- **ログフォルダへのアクセス支援**: フロントエンドのエラー画面等において、ユーザーがログを簡単に確認して手動提供できるようにするため、「ログ保存先パスのコピー」ボタンおよび「ログフォルダを開く（OSのFinderやエクスプローラーを起動する）」ボタンを配置する。

### 7.3 パニック時の扱い
バックエンドの実行時エラーである `panic` など、アプリケーション自体が強制終了するような致命的クラッシュについては、独自のリカバリー処理を伴うログファイル出力などは行わず、Goランタイムによる標準エラー出力（stderr）への書き出しに留め、実装をシンプルに保つ。

### 7.4 アップデート検知の非搭載と完全オフラインポリシー
- 本アプリケーションは、自動・手動問わずインターネットを介したアップデートチェック機能（GitHub API等への問い合わせを含む）や自動アップデート機能は一切搭載しない。
- アプリケーションが動作中に発生させるすべてのネットワークリクエストはローカル環境（`localhost` 等）のみに制限され、外部への通信は完全に排除される。
- アップデートは、新しい実行バイナリ/パッケージを手動で再取得し、再インストールする運用とする。
