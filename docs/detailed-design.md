# 詳細設計書 — codex-session-display

## 1. プロジェクト概要

### 1.1 技術スタック

| レイヤー       | 技術                                          | バージョン |
| -------------- | --------------------------------------------- | ---------- |
| バックエンド   | Go                                            | 1.23+      |
| HTTPルーター   | 標準 `net/http` + `http.ServeMux`（Go 1.22+） | —          |
| フロントエンド | React                                         | 19.x       |
| 可視化         | @xyflow/react（React Flow v12）               | 12.x       |
| ビルドツール   | Vite                                          | 6.x        |
| 型チェック     | TypeScript                                    | 5.x        |
| スタイリング   | CSS Modules                                   | —          |
| ルーティング   | React Router                                  | 7.x        |

---

## 2. バックエンド詳細設計

### 2.1 データモデル

#### 2.1.1 セッション一覧の1件（SessionSummary）

| フィールド     | 型               | 説明                                       |
| -------------- | ---------------- | ------------------------------------------ |
| ID             | 文字列           | セッションの一意識別子（UUID）             |
| FilePath       | 文字列           | セッションファイルの相対パス               |
| Cwd            | 文字列またはnull | 作業ディレクトリ                           |
| CliVersion     | 文字列またはnull | Codex CLIのバージョン                      |
| Originator     | 文字列またはnull | 起動元（codex-tui など）                   |
| ModelProvider  | 文字列またはnull | モデルプロバイダー（openai など）          |
| Branch         | 文字列またはnull | Gitブランチ名                              |
| Source         | 文字列またはnull | ソース情報（cli など）                     |
| Timestamp      | 文字列またはnull | セッション開始時刻                         |
| FileSize       | 整数             | ファイルサイズ（バイト）                   |
| FileModifiedAt | 文字列またはnull | ファイル最終更新日時                       |
| Parsed         | 真偽値           | キャッシュが存在する（パース済み）かどうか |

#### 2.1.2 トークン内訳（TokenBreakdown）

| フィールド            | 型   | 説明               |
| --------------------- | ---- | ------------------ |
| TotalTokens           | 整数 | 合計トークン数     |
| InputTokens           | 整数 | 入力トークン数     |
| OutputTokens          | 整数 | 出力トークン数     |
| ReasoningOutputTokens | 整数 | 推論出力トークン数 |

#### 2.1.3 トークン詳細（TokenDetail）

TokenBreakdownに加えて、キャッシュされた入力トークン数（CachedInputTokens）を含む。トークンカウントの詳細表示で使用する。

#### 2.1.4 ターン統計（TurnStatistics）

| フィールド            | 型             | 説明                                                                                                                                                                                                                                      |
| --------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Index                 | 整数           | ターンのインデックス（0始まり）                                                                                                                                                                                                           |
| CollaborationModeKind | 文字列         | コラボレーションモードの種類                                                                                                                                                                                                              |
| DurationMs            | 整数           | ターンの所要時間（ミリ秒）                                                                                                                                                                                                                |
| TimeToFirstTokenMs    | 整数           | 初回トークン生成までの時間（ミリ秒）                                                                                                                                                                                                      |
| TokenCountCount       | 整数           | ターン内のtoken_countイベント数                                                                                                                                                                                                           |
| ConsumedTokens        | TokenBreakdown | ターンで消費されたトークンの内訳。当ターン内最後のtoken_countのtotal_token_usageから前ターン内最後のtoken_countのtotal_token_usageを減じた差分（各トークン種別ごとに計算）。最初のターンの場合は当ターンのtotal_token_usageをそのまま使用 |

#### 2.1.5 セッション全体の統計（Statistics）

| フィールド        | 型                   | 説明                                                                                                                                                                      |
| ----------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DurationMs        | 整数                 | 最初のターンのstarted_atから最後のターンのcompleted_atまでの差分（ミリ秒）。started_atとcompleted_atは秒単位のため、パーサーで×1000して計算する。ターン間のギャップを含む |
| TotalTokens       | 整数                 | 最後のtoken_countイベントのtotal_token_usage.total_tokensと等価                                                                                                           |
| ToolCallCount     | 整数                 | response_item(function_call)のレコード件数                                                                                                                                |
| TokenCountCount   | 整数                 | 全ターンのtoken_countイベントの合計件数                                                                                                                                   |
| ContextWindowSize | 整数                 | task_startedイベントのmodel_context_windowフィールドの値                                                                                                                  |
| TurnCount         | 整数                 | ターン数                                                                                                                                                                  |
| Turns             | TurnStatisticsの配列 | 各ターンの統計情報                                                                                                                                                        |

#### 2.1.6 トークンカウントエントリ（TokenCountEntry）

| フィールド      | 型          | 説明                          |
| --------------- | ----------- | ----------------------------- |
| Index           | 整数        | token_countの通しインデックス |
| TurnIndex       | 整数        | 属するターンのインデックス    |
| BoundToNodeID   | 文字列      | 紐付け先ノードのID            |
| LastTokenUsage  | TokenDetail | 直近のトークン使用量          |
| TotalTokenUsage | TokenDetail | 累計トークン使用量            |

### 2.2 JSONLパーサー

#### 2.2.1 JSONLレコードの型分類

JSONLファイルの1行は、トップレベルの `type` フィールドで以下のいずれかに分類される。

| トップレベル type | レコード種別         | 説明                                                                   |
| ----------------- | -------------------- | ---------------------------------------------------------------------- |
| `session_meta`    | セッションメタデータ | セッションの起動時情報（CLIバージョン、作業ディレクトリ、Git情報など） |
| `turn_context`    | ターンコンテキスト   | ターン開始時の設定（モデル、コラボレーションモード、指示内容など）     |
| `event_msg`       | イベントメッセージ   | ターン内の各種イベント。`payload.type` でさらに細分される              |
| `response_item`   | レスポンスアイテム   | APIレスポンスの構成要素。`payload.type` でさらに細分される             |

#### 2.2.2 event_msg のサブタイプ

| payload.type             | 内容                                                                       |
| ------------------------ | -------------------------------------------------------------------------- |
| `user_message`           | ユーザーからのメッセージ                                                   |
| `agent_message`          | エージェントからのメッセージ                                               |
| `agent_reasoning`        | エージェントの推論テキスト                                                 |
| `task_started`           | ターンの開始を示すイベント                                                 |
| `task_complete`          | ターンの完了を示すイベント。所要時間やトークン使用量を含む                 |
| `turn_aborted`           | ターンの中断を示すイベント。task_completeと同等にターン終了として扱う      |
| `token_count`            | トークン使用量のカウント                                                   |
| `item_completed`         | アイテム完了イベント                                                       |
| `thread_name_updated`    | セッションタイトルの更新。thread_idでセッションと紐付く                    |
| `exec_command_end`       | コマンド実行完了。call_id、command、exit_code、duration等を含む            |
| `error`                  | エラーイベント。message、codex_error_infoを含む                            |
| `web_search_end`         | Web検索完了。call_id、query、actionを含む                                  |
| `collab_agent_spawn_end` | コラボエージェント起動完了。新スレッドID、prompt、model等を含む            |
| `collab_close_end`       | コラボ終了。受信エージェント情報、完了内容を含む                           |
| `collab_waiting_end`     | コラボ待機完了。各エージェントのステータスを含む                           |
| `mcp_tool_call_end`      | MCPツール呼び出し完了。invocation（server, tool, arguments）、resultを含む |
| `view_image_tool_call`   | 画像表示ツール呼び出し。call_id、pathを含む                                |

#### 2.2.3 response_item のサブタイプ

| payload.type              | 内容                                                          |
| ------------------------- | ------------------------------------------------------------- |
| `message`                 | メッセージ。`role` で developer / user / assistant に分かれる |
| `reasoning`               | 推論のサマリー                                                |
| `function_call`           | ツール呼び出し。関数名と引数を含む                            |
| `function_call_output`    | ツール呼び出しの結果                                          |
| `custom_tool_call`        | カスタムツール呼び出し。name、inputを含む                     |
| `custom_tool_call_output` | カスタムツール呼び出しの結果。outputを含む                    |
| `web_search_call`         | Web検索呼び出し。action（type, query, queries）を含む         |

#### 2.2.4 session_meta のデータ構造

| フィールド       | 説明                                                                         |
| ---------------- | ---------------------------------------------------------------------------- |
| ID               | セッションID（UUID）                                                         |
| Cwd              | 作業ディレクトリ                                                             |
| CliVersion       | CLIバージョン                                                                |
| Originator       | 起動元                                                                       |
| Source           | ソース情報                                                                   |
| ThreadSource     | スレッドソース                                                               |
| ModelProvider    | モデルプロバイダー                                                           |
| BaseInstructions | システム指示（`{"text": "文字列"}` オブジェクト。パーサーで `.text` を抽出） |
| Git              | Git情報（コミットハッシュ、ブランチ名、リポジトリURL）                       |
| Timestamp        | ペイロード内タイムスタンプ（セッション開始時刻）                             |
| TopTimestamp     | トップレベルのタイムスタンプ（レコード書き込み時刻）                         |

#### 2.2.5 turn_context のデータ構造

| フィールド        | 説明                                                               |
| ----------------- | ------------------------------------------------------------------ |
| TurnID            | ターンID                                                           |
| Model             | 使用モデル名                                                       |
| ApprovalPolicy    | 承認ポリシー                                                       |
| CollaborationMode | コラボレーションモードの設定（モード種別、developer_instructions） |
| UserInstructions  | ユーザー指示                                                       |
| Effort            | 推論努力度                                                         |
| Personality       | パーソナリティ設定                                                 |

#### 2.2.6 event_msg のデータ構造

event_msg はサブタイプによって含まれるフィールドが異なる。共通フィールドは `Type`（イベント種別）のみ。主なフィールドを以下に示す。

| フィールド            | 対象サブタイプ                                               | 説明                                                                                                        |
| --------------------- | ------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| Message / Text        | agent_message, agent_reasoning                               | メッセージ本文                                                                                              |
| TurnID                | task_started, task_complete, turn_aborted, item_completed    | ターンID                                                                                                    |
| StartedAt             | task_started                                                 | ターン開始時刻（秒タイムスタンプ）                                                                          |
| ModelContextWindow    | task_started                                                 | モデルのコンテキストウィンドウサイズ                                                                        |
| CollaborationModeKind | task_started                                                 | コラボレーションモードの種類                                                                                |
| CompletedAt           | task_complete, turn_aborted                                  | ターン完了時刻（秒タイムスタンプ）                                                                          |
| DurationMs            | task_complete, turn_aborted                                  | 所要時間（ミリ秒）                                                                                          |
| TimeToFirstTokenMs    | task_complete                                                | 初回トークンまでの時間                                                                                      |
| LastAgentMessage      | task_complete                                                | ターン完了時の最後のエージェントメッセージ。agent_messageと重複する場合がある                               |
| Reason                | turn_aborted                                                 | 中断理由                                                                                                    |
| Item                  | item_completed                                               | 完了したアイテムのID、テキスト、タイプ                                                                      |
| CompletedAtMs         | item_completed                                               | アイテム完了時刻（ミリ秒タイムスタンプ）。参照用                                                            |
| ThreadID              | item_completed                                               | スレッドID（セッションIDと同一の場合がある）。参照用                                                        |
| Info                  | token_count                                                  | `total_token_usage`（TokenDetail）, `last_token_usage`（TokenDetail）, `model_context_window`（整数）を含む |
| ThreadID              | thread_name_updated                                          | スレッドID（セッションIDと同一）。紐付けキーとして使用                                                      |
| ThreadName            | thread_name_updated                                          | セッションのタイトル。LLMが自動生成                                                                         |
| CallID                | exec_command_end, mcp_tool_call_end, view_image_tool_call    | 呼び出しID                                                                                                  |
| Command               | exec_command_end                                             | 実行コマンド（配列）                                                                                        |
| Cwd                   | exec_command_end                                             | 作業ディレクトリ                                                                                            |
| ExitCode              | exec_command_end                                             | 終了コード                                                                                                  |
| Duration              | exec_command_end                                             | 所要時間（secs, nanos）                                                                                     |
| AggregatedOutput      | exec_command_end                                             | コマンド実行結果（標準出力・エラー出力の統合）                                                              |
| ProcessID             | exec_command_end                                             | プロセスID                                                                                                  |
| Message               | error                                                        | エラーメッセージ（JSON文字列の場合がある）                                                                  |
| CodexErrorInfo        | error                                                        | エラー種別（"other" 等）                                                                                    |
| Query                 | web_search_end                                               | 検索クエリ                                                                                                  |
| Action                | web_search_end                                               | 検索アクション（type, query, queries）                                                                      |
| SenderThreadID        | collab_agent_spawn_end, collab_close_end, collab_waiting_end | 送信元スレッドID                                                                                            |
| NewThreadID           | collab_agent_spawn_end                                       | 新規エージェントのスレッドID                                                                                |
| NewAgentNickname      | collab_agent_spawn_end                                       | 新規エージェントのニックネーム                                                                              |
| NewAgentRole          | collab_agent_spawn_end                                       | 新規エージェントのロール                                                                                    |
| Prompt                | collab_agent_spawn_end                                       | エージェントへの指示                                                                                        |
| Model                 | collab_agent_spawn_end                                       | 使用モデル名                                                                                                |
| ReasoningEffort       | collab_agent_spawn_end                                       | 推論努力度                                                                                                  |
| Status                | collab_agent_spawn_end, collab_close_end                     | ステータス（pending_init、completed等）                                                                     |
| ReceiverThreadID      | collab_close_end                                             | 受信エージェントのスレッドID                                                                                |
| ReceiverAgentNickname | collab_close_end                                             | 受信エージェントのニックネーム                                                                              |
| Statuses              | collab_waiting_end                                           | 各エージェントのステータスマップ                                                                            |
| Invocation            | mcp_tool_call_end                                            | MCP呼び出し情報（server, tool, arguments）                                                                  |
| Result                | mcp_tool_call_end                                            | MCP呼び出し結果（Ok/Err）                                                                                   |
| Path                  | view_image_tool_call                                         | 画像ファイルパス                                                                                            |

以下のフィールドは実データに存在するが、本プロジェクトでは使用しない。

| フィールド           | 対象サブタイプ | 説明                       |
| -------------------- | -------------- | -------------------------- |
| RateLimits           | token_count    | レート制限情報（通常null） |
| Images / LocalImages | user_message   | 画像添付情報               |
| TextElements         | user_message   | テキスト要素               |
| Phase                | agent_message  | フェーズ情報（通常null）   |
| MemoryCitation       | agent_message  | メモリ引用（通常null）     |

#### 2.2.7 response_item のデータ構造

| フィールド       | 対象サブタイプ                                                                 | 説明                                                                                                                                              |
| ---------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Role             | message                                                                        | メッセージのロール（developer / user / assistant）                                                                                                |
| Content          | message                                                                        | メッセージの内容配列。各要素は `type` と `text` を持つ。`type` はロールに応じて決まる: developer / user → `input_text`、assistant → `output_text` |
| Summary          | reasoning                                                                      | 推論サマリーの配列。各要素は `type`（summary_text）と `text` を持つ                                                                               |
| Content          | reasoning                                                                      | 推論の本文（現在は常にnull）                                                                                                                      |
| EncryptedContent | reasoning                                                                      | 暗号化された推論内容（現在は常にnull）                                                                                                            |
| Name             | function_call, custom_tool_call                                                | 呼び出す関数名                                                                                                                                    |
| Arguments        | function_call                                                                  | 関数の引数（JSON文字列）                                                                                                                          |
| Input            | custom_tool_call                                                               | カスタムツールの入力（パッチ内容等）                                                                                                              |
| CallID           | function_call, function_call_output, custom_tool_call, custom_tool_call_output | 呼び出しと結果を紐付けるID                                                                                                                        |
| Output           | function_call_output, custom_tool_call_output                                  | 関数の実行結果テキスト                                                                                                                            |
| Action           | web_search_call                                                                | 検索アクション（type: "search", query, queries）                                                                                                  |

### 2.3 パース処理フロー

#### 2.3.1 全体フロー

```
入力: JSONLファイルパス
  │
  ├─ Step 1: ファイルチェック
  │   ├─ ファイル不存在 → エラー（SESSION_NOT_FOUND）
  │   ├─ ファイルサイズ > 50MB → エラー（FILE_TOO_LARGE）
  │   └─ ファイルサイズ == 0 → エラー（PARSE_ERROR）
  │
  ├─ Step 2: 行ごとパース
  │   各行をJSONとして読み込み、トップレベル type で分類し、
  │   各型に応じたペイロード構造にデシリアライズする。
  │   JSONパースに失敗した行はスキップし警告ログを出力する。
  │
  ├─ Step 3: ターン分割
  │   task_started → task_complete のペアでレコードをグループ化する。
  │   task_started の turn_id をキーにターンを識別する。
  │   turn_id を持たないレコードは出現位置から属するターンを判定する。
  │   session_meta はターン外レコードとして扱う。
  │
  ├─ Step 4: ターン内レコード分類
  │   各ターン内のレコードを以下のカテゴリに振り分ける：
  │   ├─ turn_context → ターンの設定情報
  │   ├─ response_item(message, role=developer) → 開発者メッセージ
  │   ├─ response_item(message, role=user) → ユーザーメッセージ
  │   ├─ response_item(message, role=assistant) → バッチ中間メッセージ
  │   ├─ event_msg(user_message) → ユーザーイベント
  │   ├─ event_msg(agent_message) → エージェントメッセージ
  │   ├─ event_msg(agent_reasoning) → 推論テキスト（後でペアリング）
  │   ├─ response_item(reasoning) → 推論サマリー（後でペアリング）
  │   ├─ response_item(function_call) → ツール呼び出し（後でバッチ検出）
  │   ├─ response_item(function_call_output) → ツール結果（後でバッチ検出）
  │   ├─ response_item(web_search_call) → Web検索呼び出し
  │   ├─ event_msg(web_search_end) → Web検索完了
  │   ├─ event_msg(item_completed) → アイテム完了イベント
  │   ├─ event_msg(token_count) → トークンカウント
  │   ├─ call_id を持つイベント（exec_command_end, mcp_tool_call_end,
  │   │   view_image_tool_call 等）→ 外部イベントブランチ（§2.6 Step 3 で処理）
  │   └─ その他（error, collab_* 等）→ generic ノードとして扱う
  │
  ├─ Step 5: reasoning ペアリング
  │   agent_reasoning と response_item(reasoning) を出現順で1:1ペアにする。
  │   ペアリングできないものは単独で保持する。
  │   ※ v0.121.0以降では両方存在する場合常に1:1が成立する。
  │   ※ 一部バージョンでは agent_reasoning が出力されず、
  │     response_item(reasoning) のみ（暗号化済み・summary空）となる場合がある。
  │
  ├─ Step 6: バッチ検出
  │   連続する function_call 群と function_call_output 群をバッチとして検出する。
  │   詳細は §2.4 を参照。
  │
  └─ Step 7: token_count 紐付け
      各 token_count の直前の非token_countレコードを特定し、紐付ける。
```

#### 2.3.2 パース結果のデータ構造（ParsedSession）

| フィールド  | 型                           | 説明                                                     |
| ----------- | ---------------------------- | -------------------------------------------------------- |
| SessionMeta | SessionMetaPayloadまたはnull | セッションのメタデータ                                   |
| ThreadName  | 文字列またはnull             | セッションのタイトル（thread_name_updatedのthread_name） |
| Turns       | Turnの配列                   | ターンごとのデータ                                       |
| RawRecords  | TypedRecordの配列            | 順序を保持した全レコード                                 |

#### 2.3.3 ターンのデータ構造（Turn）

| フィールド           | 型                           | 説明                                 |
| -------------------- | ---------------------------- | ------------------------------------ |
| Index                | 整数                         | ターンのインデックス                 |
| TurnID               | 文字列                       | ターンID                             |
| TaskStarted          | EventMsgPayloadまたはnull    | task_startedイベント                 |
| TaskComplete         | EventMsgPayloadまたはnull    | task_complete / turn_abortedイベント |
| Aborted              | 真偽値                       | turn_abortedで終了した場合true       |
| TurnContext          | TurnContextPayloadまたはnull | ターンコンテキスト                   |
| Records              | TypedRecordの配列            | ターン内の全レコード（時系列順）     |
| Batches              | Batchの配列                  | 検出されたバッチ                     |
| DeveloperMessages    | TypedRecordの配列            | 開発者メッセージ                     |
| UserMessages         | TypedRecordの配列            | ユーザーメッセージ                   |
| UserEventMsg         | TypedRecordまたはnull        | event_msg(user_message)              |
| AgentReasonings      | TypedRecordの配列            | 推論テキストとサマリーのペア         |
| AgentMessages        | TypedRecordの配列            | エージェントメッセージ               |
| ItemCompleted        | TypedRecordの配列            | アイテム完了イベント                 |
| TokenCounts          | TokenCountWithBindingの配列  | 紐付け済みtoken_count                |
| WebSearchRecords     | TypedRecordの配列            | Web検索呼び出し・完了レコード        |
| ExternalEventRecords | TypedRecordの配列            | call_idを持つ外部イベントレコード    |
| GenericRecords       | TypedRecordの配列            | その他の未分類レコード               |

#### 2.3.4 型付きレコード（TypedRecord）

パースされた1レコードのラッパー。元ファイルの行番号、レコード種別、イベント/レスポンスのサブタイプ、ペイロードの生データとデシリアライズ済みデータを保持する。

### 2.4 バッチ検出アルゴリズム

#### 2.4.1 バッチとは

1ターン内で、エージェントが複数のツールを連続して呼び出し、その結果をまとめて受け取るパターンを「バッチ」と呼ぶ。バッチは以下の要素で構成される。

| 要素           | 説明                                                                                                                                                                               |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CallRecords    | 連続する function_call または custom_tool_call の配列（混在しない）                                                                                                                |
| OutputRecords  | 対応する function_call_output または custom_tool_call_output の配列（null を含む場合がある）                                                                                       |
| MiddleMessage  | バッチ中間のエージェントメッセージ（存在する場合）                                                                                                                                 |
| ContextRecords | バッチ中間の実行コンテキストイベント（exec_command_end、token_count等）。パターンA/Bの中間に挟まる。call_idを持つレコードは§2.6 Step 3で外部イベントブランチノードとして処理される |
| IsPatternB     | パターンB（中間メッセージが response_item のみ）かどうか                                                                                                                           |

#### 2.4.2 バッチのパターン

**function_call バッチ:**

**パターンA**: function_call群 → [ContextRecords] → agent_message → response_item(message, role=assistant) → [ContextRecords] → function_call_output群

**パターンB**: function_call群 → [ContextRecords] → response_item(message, role=assistant) → [ContextRecords] → function_call_output群（agent_messageなし）

※ ContextRecords として exec_command_end、token_count、mcp_tool_call_end、view_image_tool_call 等が挟まる場合がある。これらは実行コンテキストイベントであり、MiddleMessage とは別枠で ContextRecords に格納する。ノード生成時（§2.6）では、call_id を持つ ContextRecords は外部イベントブランチノード（Step 3）として処理され、ハーネスには配置されない。token_count は token_count 紐付け（Step 4）で処理される。

**custom_tool_call バッチ:**

custom_tool_call群 → custom_tool_call_output群（中間レコードなし、バッチサイズは常に1）

#### 2.4.3 検出処理

```
入力: ターン内のレコード配列（時系列順）
出力: Batch の配列

=== function_call バッチ検出 ===

1. レコード配列を先頭から走査する
2. 連続する function_call を検出する（callBatch）
3. callBatch の直後から次の function_call_output までの間にあるレコードを確認:
   a. agent_message が含まれる → パターンA。MiddleMessage として保持。
      同一位置の response_item(message, role=assistant) があればスキップ
   b. response_item(message, role=assistant) のみで agent_message なし → パターンB
   c. 上記以外 → MiddleMessage なし
   ※ a〜c いずれの場合も、agent_message/assistant message 以外のレコード
     （exec_command_end、token_count 等）は ContextRecords として保持
4. 連続する function_call_output を検出する（outputBatch）
5. call_id で call と output の対応を検証・復旧:
   a. callBatch[i] の call_id == outputBatch[i] の call_id が全件一致
      → そのまま Batch 生成へ
   b. 不一致あり → 警告ログを出力し、call_id マップで復旧:
      - outputMap[call_id] = outputRecord を構築
      - call の順序に従い output を再配置:
        OutputRecords[i] = outputMap[callBatch[i].call_id]（存在しない場合は null）
      - 対応する call がない output → 破棄
6. Batch を生成する
7. 走査位置を outputBatch の末尾に進める

=== custom_tool_call バッチ検出 ===

8. レコード配列を先頭から走査する
9. 連続する custom_tool_call を検出する（callBatch）
10. 連続する custom_tool_call_output を検出する（outputBatch）
11. call_id で call と output の対応を検証・復旧（手順5と同一ロジック）
12. Batch を生成する（MiddleMessage なし、ContextRecords なし）
13. 走査位置を outputBatch の末尾に進める

両者の検出結果をマージして Batch の配列として返す
```

### 2.5 React Flow形式への変換

#### 2.5.1 FlowGraph のデータ構造

| フィールド | 型             | 説明             |
| ---------- | -------------- | ---------------- |
| Nodes      | FlowNodeの配列 | グラフのノード群 |
| Edges      | FlowEdgeの配列 | ノード間の接続   |

#### 2.5.2 FlowNode のデータ構造

| フィールド | 型       | 説明                                 |
| ---------- | -------- | ------------------------------------ |
| ID         | 文字列   | ノードの一意ID                       |
| Type       | 文字列   | カスタムノードタイプ名（§2.5.4参照） |
| Position   | {X, Y}   | キャンバス上の座標                   |
| Data       | NodeData | ノードの表示データ                   |

#### 2.5.3 NodeData のデータ構造

| フィールド | 型                         | 説明                                                                                           |
| ---------- | -------------------------- | ---------------------------------------------------------------------------------------------- |
| Category   | 文字列                     | ノードのカテゴリ（メタ系、Turn系、イベント系、思考系、Action系、メッセージ系、コンテキスト系） |
| Label      | 文字列                     | ノードヘッダーのタイトル                                                                       |
| Icon       | 文字列                     | ノードアイコン文字                                                                             |
| Summary    | 文字列                     | ノードbodyに表示する短いテキスト                                                               |
| FullText   | 文字列または未設定         | クリック展開時の全文                                                                           |
| Meta       | キー・バリューのマップ     | 詳細パネル表示用のメタデータ                                                                   |
| BatchIndex | 整数                       | バッチ内のインデックス                                                                         |
| BatchSize  | 整数                       | バッチサイズ                                                                                   |
| TokenBadge | TokenBadgeDataまたは未設定 | トークンバッジ                                                                                 |
| Collapsed  | 真偽値                     | 折りたたみ状態                                                                                 |
| TextLength | 整数                       | テキスト文字数                                                                                 |
| TurnIndex  | 整数                       | 属するターンのインデックス                                                                     |

#### 2.5.3.1 TokenBadgeData のデータ構造

| フィールド      | 型   | 説明                                                                                        |
| --------------- | ---- | ------------------------------------------------------------------------------------------- |
| TotalTokens     | 整数 | 紐付く最新の token_count の総トークン数                                                     |
| TokenCountIndex | 整数 | 紐付く最初の token_count の token_counts 配列インデックス                                   |
| BoundCount      | 整数 | 紐付く token_count の件数。1の場合は通常表示、2以上の場合は件数をバッジに併記（例: 「×2」） |

#### 2.5.4 カスタムノードタイプ一覧

| ノードタイプ       | 対象レコード                                                                                                            | カテゴリ       |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------- | -------------- |
| `sessionMeta`      | session_meta                                                                                                            | メタ系         |
| `taskEvent`        | task_started, task_complete, turn_aborted                                                                               | イベント系     |
| `turnContext`      | turn_context                                                                                                            | Turn系         |
| `userMessage`      | event_msg(user_message)                                                                                                 | Turn系         |
| `agentMessage`     | event_msg(agent_message), response_item(message, role=assistant)                                                        | Turn系         |
| `reasoning`        | agent_reasoning + reasoning（統合）                                                                                     | 思考系         |
| `action`           | function_call, function_call_output, custom_tool_call, custom_tool_call_output                                          | Action系       |
| `webSearchAction`  | response_item(web_search_call), event_msg(web_search_end)                                                               | Action系       |
| `developerMessage` | response_item(message, role=developer)                                                                                  | メッセージ系   |
| `userApiMessage`   | response_item(message, role=user)                                                                                       | メッセージ系   |
| `contextDoc`       | base_instructions, developer_instructions, user_instructions                                                            | コンテキスト系 |
| `itemCompleted`    | event_msg(item_completed)                                                                                               | イベント系     |
| `externalEvent`    | exec_command_end, mcp_tool_call_end, view_image_tool_call, collab_agent_spawn_end, collab_close_end, collab_waiting_end | Action系       |
| `generic`          | 未知タイプ                                                                                                              | 未知タイプ     |

#### 2.5.5 FlowEdge のデータ構造

| フィールド | 型     | 説明               |
| ---------- | ------ | ------------------ |
| ID         | 文字列 | エッジの一意ID     |
| Source     | 文字列 | 接続元ノードID     |
| Target     | 文字列 | 接続先ノードID     |
| Type       | 文字列 | "default" / "step" |
| Animated   | 真偽値 | アニメーション有無 |

### 2.6 ノード生成フロー

パース結果からFlowGraphを生成する処理フローを以下に示す。

```
Build(session)
  │
  ├─ 1. sessionMeta ノード生成 → ハーネススタックに push
  │
  ├─ 2. 各 Turn を処理:
  │     ├─ 2a. task_started ノード生成 → ハーネススタックに push
  │     │
  │     ├─ 2b. DeveloperMessages + UserMessages を分岐ノードとして生成
  │     │   ハーネス最上位ノードから分岐エッジを張る
  │     │   分岐ノードはハーネス右側に配置
  │     │
  │     ├─ 2c. turnContext ノード生成 → ハーネススタックに push
  │     │   コンテキスト分岐ノードを生成:
  │     │     ├─ base_instructions ノード（折りたたみ）
  │     │     ├─ developer_instructions ノード（折りたたみ）
  │     │     └─ user_instructions ノード（折りたたみ）
  │     │   turnContext から各コンテキストノードへ分岐エッジ
  │     │
  │     ├─ 2d. user_message ノード生成 → ハーネススタックに push
  │     │
  │     ├─ 2e. reasoning ノード生成 → ハーネススタックに push
  │     │   ペアリング済みの場合:
  │     │     agent_reasoning のテキストを summary に設定
  │     │     reasoning の summary を fullText の一部に設定
  │     │   スタンドアロンAR（ARのみ）の場合:
  │     │     ARテキストを summary および fullText に設定
  │     │   スタンドアロンRI（RIのみ・暗号化済み）の場合:
  │     │     「（暗号化済み・表示不可）」を summary および fullText に設定
  │     │
  │     ├─ 2f. 各 Batch を処理（fork-join パターン）:
  │     │   ├─ call[i] ノードを分岐ノードとして生成
  │     │   │   ハーネス最上位ノードから step エッジで接続
  │     │   │   X = BranchOffsetX + i * (NodeWidth + BranchNodeGapX)
  │     │   │   Y = ハーネス最上位ノード.Y（全call同一Y）
  │     │   ├─ output[i] ノードを call[i] の直下に生成
  │     │   │   call[i] から default エッジで接続
  │     │   │   X = call[i].X
  │     │   │   Y = call[i].Y + NodeHeight + BatchNodeGap
  │     │   ├─ MiddleMessage が存在する場合:
  │     │   │   MiddleMessage ノードをハーネススタックに push（join先）
  │     │   │   各 output[i] から default エッジで MiddleMessage に接続
  │     │   ├─ MiddleMessage が存在しない場合:
  │     │   │   join先は次のハーネスノード（2g' の agentMessage 等）
  │     │   │   各 output[i] から default エッジで join先に接続
  │     │   └─ ※ OutputRecords[i] が null の場合、当該 output ノードはスキップ
  │     │
  │     ├─ 2f'. 各 web_search_call / web_search_end を処理:
  │     │   ├─ webSearchAction ノード（call側）を生成 → ハーネススタックに push
  │     │   ├─ webSearchAction ノード（end側）を生成 → ハーネススタックに push
  │     │   └─ call側 → end側 へエッジ
  │     │
  │     ├─ 2g. turn.ItemCompleted から item_completed ノード生成（存在する場合）
  │     │
  │     ├─ 2g'. 非バッチの agent_message / response_item(message, assistant) を
  │     │       agentMessage ノードとしてハーネススタックに push
  │     │       バッチ検出（Step 6）でバッチに含まれなかったレコードが対象。
  │     │       agent_message と response_item(message, assistant) が
  │     │       連続して出現する場合は同一内容のことが多いため、1つのノードに統合し、
  │     │       マッピングテーブルには両方のレコードを同じノードIDに紐付ける
  │     │
  │     ├─ 2h. genericRecords から generic ノード生成 → ハーネススタックに push
  │     │   ※ call_id を持つ外部イベント（exec_command_end, mcp_tool_call_end,
  │     │     view_image_tool_call 等）は除外（Step 3 で処理）
  │     │
  │     └─ 2i. task_complete ノード生成 → ハーネススタックに push
  │
  ├─ 3. 外部イベントブランチノード生成（タイプ: externalEvent）:
  │     ├─ call_id を持つ外部イベントレコード（exec_command_end, mcp_tool_call_end,
  │     │   view_image_tool_call, collab_agent_spawn_end, collab_close_end,
  │     │   collab_waiting_end 等）からブランチノードを生成
  │     ├─ call_id で対応する action ノード（function_call_output 等）を特定し、
  │     │   step エッジで接続する
  │     └─ ブランチノードは対応する action ノードの右側に配置
  │
  ├─ 4. 各 token_count について、紐付け先ノードに TokenBadgeData を設定
  │     ├─ a. レコード→ノードIDのマッピングテーブルを構築:
  │     │      ノード生成時に TypedRecord → FlowNode.ID の対応を記録
  │     ├─ b. token_count.BoundToRecord をマッピングテーブルで解決し
  │     │      BoundToNodeID（文字列）を設定
  │     └─ c. 紐付け先ノードごとに TokenBadgeData を集約:
  │            同一ノードに複数の token_count が紐付く場合:
  │            ├─ TotalTokens: 最後の token_count の totalTokens
  │            ├─ TokenCountIndex: 最初の token_count のインデックス
  │            └─ BoundCount: 紐付く token_count の件数
  │
  └─ 5. 全ノード・エッジを FlowGraph として返す
```

**ハーネススタック**とは、メインフロー（縦方向の直列接続）を構成するノードの管理用スタックである。スタックの最上位ノードが次のノードの接続元となる。ノードをハーネススタックに push すると、直前のスタック最上位ノードから当該ノードへエッジが張られ、スタックの最上位が更新される。

#### 2.6.1 エッジタイプ使い分け基準

| エッジタイプ | 接続パターン                       | 説明                                                                                |
| ------------ | ---------------------------------- | ----------------------------------------------------------------------------------- |
| default      | ハーネス上の直列接続・バッチ内接続 | メインフロー、call→output接続、output→join接続、WebSearch接続                       |
| step         | ハーネスからの水平分岐             | バッチ分岐（ハーネス→call）、コンテキスト分岐、メッセージ分岐、外部イベントブランチ |

### 2.7 レイアウト計算

#### 2.7.1 設計方針

- **Y座標**: 上から下へのフロー。ノード高さ + gap で累積的に増加
- **X座標**: ハーネス（メインフロー）は X=0。分岐ノードは X > 0
- レイアウトはバックエンドで計算し、固定座標をNode.Positionに設定する
- React Flow側ではズーム・パンのみ（自動レイアウトなし）

#### 2.7.2 レイアウト定数

| 定数名          | 値  | 説明                            |
| --------------- | --- | ------------------------------- |
| HarnessX        | 0   | ハーネスのX座標                 |
| BranchOffsetX   | 400 | 分岐ノードのXオフセット         |
| ContextOffsetX  | 400 | コンテキストノードのXオフセット |
| NodeWidth       | 320 | ノード幅                        |
| NodeHeight      | 80  | ノードの基本高さ                |
| NodeHeightLarge | 120 | テキスト量が多いノードの高さ    |
| NodeGap         | 40  | ノード間の縦gap                 |
| BatchNodeGap    | 8   | バッチ内ノードの縦gap           |
| BatchMiddleGap  | 24  | バッチ中間メッセージのgap       |
| BranchNodeGapX  | 20  | バッチ分岐ノード間の水平gap     |

NodeHeightの使い分け基準:

- turnContext, reasoning → NodeHeightLarge=120
- それ以外 → NodeHeight=80

#### 2.7.3 座標計算アルゴリズム

1. ハーネス上のノードを上から順に配置。Y座標は「直前ノードのY + ノード高さ + gap」で累積
2. 分岐ノードのY座標: 対応するハーネスノードのY座標に揃える
3. 分岐ノードのX座標: BranchOffsetX
4. コンテキストノードのX座標: ContextOffsetX。複数のコンテキストノードは縦に並べる
5. バッチ分岐ノードの座標計算（fork-join パターン）:
   a. call[i].X = BranchOffsetX + i \* (NodeWidth + BranchNodeGapX)
   b. call[i].Y = 分岐元ハーネスノード.Y（全call同一Y）
   c. output[i].X = call[i].X
   d. output[i].Y = call[i].Y + NodeHeight + BatchNodeGap
   e. join先（MiddleMessage または次のハーネスノード）.Y = call[i].Y + NodeHeight + BatchNodeGap + NodeHeight + NodeGap
6. コンテキスト分岐ノードの座標計算:
   a. contextDoc[i].X = ContextOffsetX（全ノード同一X）
   b. contextDoc[0].Y = turnContext.Y（1番目はturnContextに揃える）
   c. contextDoc[i].Y = turnContext.Y + i \* (NodeHeight + NodeGap)（2番目以降は累積）

### 2.8 セッションスキャン処理

`~/.codex/sessions` 配下をスキャンし、セッション一覧を生成する。

#### 2.8.1 ファイル名フォーマット

ファイルパスは以下の形式に従う:

```
~/.codex/sessions/{YYYY}/{MM}/{DD}/rollout-{YYYY}-{MM}-{DD}T{HH}-{MM}-{SS}-{UUID}.jsonl
```

| 要素           | パターン                                                               | 例                                     |
| -------------- | ---------------------------------------------------------------------- | -------------------------------------- |
| ディレクトリ   | `YYYY/MM/DD`                                                           | `2026/05/23`                           |
| プレフィックス | `rollout-`（固定）                                                     | —                                      |
| タイムスタンプ | `YYYY-MM-DDTHH-MM-SS`（`\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}`）         | `2026-05-23T22-44-55`                  |
| セッションID   | UUID（`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`） | `019e5514-ed44-78b2-bf88-233d6e4273bf` |

※ ディレクトリの日付（YYYY/MM/DD）は常にファイル名のタイムスタンプ日付と一致する

#### 2.8.2 セッションID・タイムスタンプ抽出

UUIDは常に末尾36文字（固定長）のため、文字列操作で抽出する:

```
1. ファイル名から "rollout-" プレフィックスと ".jsonl" サフィックスを除去
2. 末尾36文字をセッションIDとして抽出
3. 残りの文字列（末尾のハイフンを除く）をタイムスタンプとして抽出
4. タイムスタンプの "-" を ":" に変換（時刻部分のみ: T以降）
5. ローカル時刻としてパース後、UTCに変換（例: 22:44:55 JST → 13:44:55 UTC）
6. ISO 8601形式で出力（例: 2026-05-23T13:44:55Z）
```

※ ファイルパスのタイムスタンプはローカル時刻。session_meta.Timestamp（UTC）との一貫性を保つため、UTCに変換する

#### 2.8.3 処理フロー

```
処理フロー:
  1. セッションディレクトリ (YYYY/MM/DD/) を再帰走査
  2. .jsonl ファイルを検出
  3. ファイルパスからセッションIDを抽出（§2.8.2）
  4. ファイルパスからタイムスタンプを推測（§2.8.2）
  5. ファイルサイズを取得
  6. ファイル更新日時を取得
  7. キャッシュを確認:
     a. キャッシュが存在 → session_meta をキャッシュから読込 → Parsed=true
     b. キャッシュなし → Parsed=false
  8. 日付降順でソートして返す
```

### 2.9 キャッシュ管理

#### 2.9.1 キャッシュの場所

`~/.codex-display/{sessionID}.json`。ファイルの中身はセッション詳細API（`GET /api/sessions/:id`）のレスポンスと同一形式。

#### 2.9.2 再パース要否の判定

再パースが必要な条件は以下の通り:

1. キャッシュファイルが存在しない
2. JSONLファイルの最終更新日時 > キャッシュファイルの最終更新日時
3. キャッシュファイルの読み込み（JSONデコード）に失敗（破損ファイル）

上記以外の場合はキャッシュを使用する。詳細は §7.2 を参照。

#### 2.9.3 キャッシュ書き込み失敗時の挙動

エラーログを出力し、キャッシュなしで動作を継続する。次回アクセス時に再パースが走る。

### 2.10 HTTPハンドラ

#### 2.10.1 セッション一覧ハンドラ（GET /api/sessions）

1. セッションスキャンを実行
2. セッション一覧をJSONで返す
3. エラー時は適切なエラーレスポンスを返す

#### 2.10.2 セッション詳細ハンドラ（GET /api/sessions/:id）

1. URLパスからセッションIDを抽出
2. キャッシュの再パース要否をチェック
3. キャッシュが有効 → キャッシュから読み込んで返す
4. キャッシュなし/期限切れ:
   a. スキャナからファイルパスを特定
   b. パーサーでJSONLをパース
   c. FlowGraphを生成
   d. 統計情報を生成
   e. token_countエントリを構築
   f. レイアウト計算を実行
   g. キャッシュに保存
   h. レスポンスを返す
5. エラー処理:
   - セッション未検出 → 404 SESSION_NOT_FOUND
   - ファイルサイズ超過 → 413 FILE_TOO_LARGE
   - 非対応形式 → 422 UNSUPPORTED_FORMAT
   - パース失敗 → 422 PARSE_ERROR
   - ファイル読み込み失敗 → 500 FILE_READ_ERROR
   - 内部エラー → 500 INTERNAL_ERROR

#### 2.10.3 統計画像ハンドラ（GET /api/sessions/:id/stats-image）

1. URLパスからセッションIDを抽出
2. キャッシュまたはパースから統計情報（Statistics）とセッションメタデータを取得
3. Goネイティブの2DグラフィックスライブラリでPNG画像を生成:
   a. 上部ヘッダー: セッションID（短縮形式）、ブランチ名、作業ディレクトリ、タイムスタンプ
   b. 下部: 6つのstat-card（2列×3行グリッド）
   - 所要時間、総トークン数、ツール呼び出し数、トークンカウント数、コンテキストウィンドウサイズ、ターン数
     c. 画像サイズ: 1920×1080（固定）
4. PNG画像バイナリをレスポンスとして返す（Content-Type: image/png）
5. エラー処理:
   - セッション未検出 → 404 SESSION_NOT_FOUND
   - ファイルサイズ超過 → 413 FILE_TOO_LARGE
   - 非対応形式 → 422 UNSUPPORTED_FORMAT
   - パース失敗 → 422 PARSE_ERROR
   - ファイル読み込み失敗 → 500 FILE_READ_ERROR
   - 画像生成失敗 → 500 INTERNAL_ERROR

---

## 3. フロントエンド詳細設計

### 3.1 コンポーネントツリー

```
<App>
  ├── ルーター
  │     ├── ルート "/"
  │     │     └── <SessionListPage>
  │     │           ├── <Toolbar>
  │     │           │     ├── セッション件数表示
  │     │           │     ├── ソースパス表示
  │     │           │     └── <SearchBox>
  │     │           └── <DateTree>
  │     │                 └── <YearGroup>[] → <MonthGroup>[] → <DayGroup>[]
  │     │                       └── <SessionRow>[]
  │     │
  │     └── ルート "/sessions/:id"
  │           └── <SessionDetailPage>
  │                 ├── <Header>
  │                 │     ├── 戻るボタン（← 一覧に戻る）
  │                 │     ├── セッションID表示
  │                 │     ├── ブランチ表示
  │                 │     ├── CWD表示
  │                 │     ├── タイムスタンプ表示
  │                 │     └── <ExportButton>
  │                 ├── <FlowCanvas>            ← React Flow
  │                 │     ├── <SessionMetaNode>
  │                 │     ├── <TurnEventNode>
  │                 │     ├── <TurnContextNode>
  │                 │     │     └── <ContextDocNode>[]     ← コンテキスト分岐
  │                 │     ├── <UserMessageNode>
  │                 │     ├── <AgentMessageNode>
  │                 │     ├── <DeveloperMessageNode>
  │                 │     ├── <UserApiMessageNode>
  │                 │     ├── <ReasoningNode>
  │                 │     ├── <ActionNode>
  │                 │     ├── <WebSearchActionNode>
  │                 │     ├── <ItemCompletedNode>
  │                 │     ├── <ExternalEventNode>
  │                 │     ├── <GenericNode>
  │                 │     └── <TokenBadge>
  │                 ├── <RightPanel>
  │                 │     ├── 統計パネル
  │                 │     ├── ターン別トークンサマリー
  │                 │     └── トークンカウント表
  │                 └── <BottomPanel>           ← ノード詳細
  │                       └── ノード詳細コンテンツ
```

### 3.2 状態管理

#### 3.2.1 セッション一覧画面の状態

| 状態          | 型                   | 説明                          |
| ------------- | -------------------- | ----------------------------- |
| sessions      | SessionSummaryの配列 | APIから取得したセッション一覧 |
| loading       | 真偽値               | ローディング状態              |
| error         | 文字列またはnull     | エラーメッセージ              |
| searchQuery   | 文字列               | 検索クエリ                    |
| expandedPaths | 文字列のセット       | 展開中の年/月/日パス          |

#### 3.2.2 セッション詳細画面の状態

| 状態               | 型                        | 説明                                | デフォルト |
| ------------------ | ------------------------- | ----------------------------------- | ---------- |
| sessionData        | SessionDetailまたはnull   | APIから取得したセッション詳細       | null       |
| loading            | 真偽値                    | ローディング状態                    | false      |
| error              | 文字列またはnull          | エラーメッセージ                    | null       |
| selectedNode       | FlowNodeまたはnull        | 選択中のノード（BottomPanel表示用） | null       |
| selectedTokenBadge | TokenCountEntryまたはnull | 選択中のトークンバッジ              | null       |
| rightPanelOpen     | 真偽値                    | RightPanelの開閉状態                | true       |
| exporting          | 真偽値                    | エクスポート中                      | false      |
| notification       | Notificationまたはnull    | トースト通知                        | null       |

Notification型: `{ message: string, type: 'success' \| 'error' \| 'info' }`

BottomPanelの表示状態は導出判定とする: `bottomPanelOpen = selectedNode !== null || selectedTokenBadge !== null`

#### 3.2.3 React Flowの状態管理方針

APIレスポンスの nodes / edges をそのまま React Flow の初期値として渡す。ノードのドラッグ移動は無効とする。ノードの選択状態は `onNodeClick` コールバックで管理し、親コンポーネントに通知する。ビューポート（ズームレベル・パン位置）の状態はReact Flowの内部管理に委ね、独自stateは定義しない。

### 3.3 APIクライアント

APIクライアントは同一オリジンで通信する。以下の3つの関数を提供する。

| 関数               | HTTPメソッド | パス                          | 戻り値                | タイムアウト |
| ------------------ | ------------ | ----------------------------- | --------------------- | ------------ |
| fetchSessionList   | GET          | /api/sessions                 | SessionListResponse   | 30秒         |
| fetchSessionDetail | GET          | /api/sessions/:id             | SessionDetailResponse | 60秒         |
| fetchStatsImage    | GET          | /api/sessions/:id/stats-image | 画像Blob              | 30秒         |

#### 3.3.1 ApiError型

HTTPステータスが正常でない場合、またはタイムアウト・ネットワークエラー時にApiErrorをスローする。

| フィールド | 型     | 説明                                                        |
| ---------- | ------ | ----------------------------------------------------------- |
| message    | 文字列 | ユーザー向けエラーメッセージ                                |
| code       | 文字列 | エラーコード（バックエンドのcode、またはTIMEOUT/NETWORK）   |
| status     | 整数   | HTTPステータスコード。タイムアウト・ネットワークエラー時は0 |

#### 3.3.2 タイムアウト処理

各API関数は `AbortController` + `setTimeout` でタイムアウトを実装する。タイムアウト発生時は `AbortError` を補足し、`code: 'TIMEOUT'`, `status: 0` のApiErrorに変換してスローする。

#### 3.3.3 ネットワークエラー処理

`fetch` 自体が失敗した場合（`TypeError`）は `code: 'NETWORK_ERROR'`, `status: 0` のApiErrorに変換してスローする。

#### 3.3.4 fetchStatsImageの処理フロー

`fetchStatsImage` はレスポンスのContent-Typeが成功時とエラー時で異なるため、以下の順序で処理する:

1. `response.ok` をチェック
2. `false` の場合: `response.json()` でエラーレスポンスをパースし、ApiErrorをスロー
3. `true` の場合: `response.blob()` で画像Blobを取得して返す

### 3.4 フロントエンドの型定義

#### 3.4.0 命名規則

| 対象                | 規則       | 例                                         |
| ------------------- | ---------- | ------------------------------------------ |
| API ドメイン型      | snake_case | `file_path`, `duration_ms`, `total_tokens` |
| React Flow 可視化型 | camelCase  | `batchIndex`, `textLength`, `totalTokens`  |

- **変換レイヤーなし**: API レスポンスの JSON をそのままフロントエンドの型として使用する。snake_case と camelCase の混在を許容する
- **Go 側の JSON タグで制御**: バックエンドの Go 構造体は JSON タグで snake_case（API ドメイン型）または camelCase（React Flow 可視化型）を出力する

#### 3.4.1 セッション関連の型

| 型名                  | フィールド概要                                                                                                                                     |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| SessionSummary        | セッション一覧の1件（id, file_path, cwd, cli_version, originator, model_provider, branch, source, timestamp, file_size, file_modified_at, parsed） |
| SessionListResponse   | sessionsフィールドにSessionSummaryの配列を持つ                                                                                                     |
| TokenBreakdown        | total_tokens, input_tokens, output_tokens, reasoning_output_tokens                                                                                 |
| TokenDetail           | TokenBreakdown + cached_input_tokens                                                                                                               |
| TurnStatistics        | index, collaboration_mode_kind, duration_ms, time_to_first_token_ms, token_count_count, consumed_tokens（TokenBreakdown）                          |
| Statistics            | duration_ms, total_tokens, tool_call_count, token_count_count, context_window_size, turn_count, turns（TurnStatisticsの配列）                      |
| TokenCountEntry       | index, turn_index, bound_to_node_id, last_token_usage（TokenDetail）, total_token_usage（TokenDetail）                                             |
| SessionDetailResponse | id, parsed_at, nodes（FlowNodeの配列）, edges（FlowEdgeの配列）, statistics（Statistics）, token_counts（TokenCountEntryの配列）                   |

#### 3.4.2 React Flow関連の型

| 型名           | フィールド概要                                                                                                           |
| -------------- | ------------------------------------------------------------------------------------------------------------------------ |
| NodeData       | category, label, icon, summary, fullText, meta, batchIndex, batchSize, tokenBadge, collapsed, textLength, turnIndex      |
| TokenBadgeData | totalTokens（紐付く最新の総トークン数）, tokenCountIndex（最初のtoken_counts配列インデックス）, boundCount（紐付く件数） |
| FlowNode       | React FlowのNode型。dataにNodeDataを持つ                                                                                 |
| FlowEdge       | React FlowのEdge型                                                                                                       |

### 3.5 React Flowカスタムノード設計

#### 3.5.1 ノード共通構造

各カスタムノードは以下の共通構造を持つ。

```
┌─────────────────────────────┐
│ [icon] LABEL         [badge]│ ← node-header
├─────────────────────────────┤
│ summary text                │ ← node-body
│ ...                         │
└─────────────────────────────┘
```

- **node-header**: アイコン + ラベル + トークンバッジ（存在する場合）。boundCount ≥ 2 の場合は件数を併記（例: 「12.3K ×2」）
- **node-body**: summary テキスト。クリックで BottomPanel に詳細を表示
- **ホバー**: ボーダー色変化（青）+ ドロップシャドウ
- **選択**: 2pxのブルーボーダー

#### 3.5.2 各カスタムノードの仕様

**SessionMetaNode**（タイプ: sessionMeta）

```
┌───────────────────────────┐
│ [S] Session Meta          │
├───────────────────────────┤
│ CLI v0.131.0 · openai     │
│ codex-tui · CW: 258,400   │
└───────────────────────────┘
```

- summary: `"CLI v{cli_version} · {model_provider} · {originator}"` — nullの項目は "—" で表示
- meta: cli_version, model_provider, originator, cwd, git_branch, git_commit

**TurnEventNode**（タイプ: taskEvent）

```
┌───────────────────────────┐
│ [▶] Turn N Started        │
├───────────────────────────┤
│ mode: plan                │
│ started_at: 2026-05-23... │
└───────────────────────────┘
```

- task_started: label = "Turn {N} Started", icon = "▶"
- task_complete: label = "Turn {N} Complete", icon = "■"
- turn_aborted: label = "Turn {N} 中断", icon = "✕"
- summary（開始時）: `"mode: {collaboration_mode_kind}\nstarted_at: {time}"` — {time}はローカル時刻 `YYYY-MM-DD HH:MM:SS` 形式
- summary（完了時）: `"所要時間: {duration}\n初回トークン: {ttft}"`
- summary（中断時）: `"所要時間: {duration}\n理由: {reason}"`

**TurnContextNode**（タイプ: turnContext）

```
┌───────────────────────────┐
│ [T] Turn Context          │
├───────────────────────────┤
│ model: glm-5.1            │
│ effort: medium            │
│ approval: on-request      │
└───────────────────────────┘
```

ハーネス上のノード + 右側にコンテキスト分岐ノードを展開。分岐先は base_instructions, developer_instructions, user_instructions。

**ActionNode**（タイプ: action）

```
┌───────────────────────────┐
│ [⚡] exec_command  [badge]│  ← function_call の場合
├───────────────────────────┤
│ cat plan_skill.md          │
└───────────────────────────┘

┌───────────────────────────┐
│ [📋] Output: exec_command │  ← function_call_output の場合
├───────────────────────────┤
│ exit code: 0               │
└───────────────────────────┘
```

- function_call: label = 関数名, summary = 引数のJSONをパースし、最初の値を再シリアライズして先頭1行を表示。値が文字列ならそのまま、配列・オブジェクトならJSON文字列の先頭を切り詰め
- function_call_output: label = "Output: {関数名}", summary = 出力の先頭1行
- バッチ情報表示: BatchSize ≥ 2 の場合のみ「{BatchIndex+1}/{BatchSize}」をノードに表示。BatchSize = 1 の場合は非表示
- デフォルト値: バッチに含まれないノード（非Action系ノード）は BatchSize=0, BatchIndex=0 とする。BatchSize=0 は非表示として扱う

**ReasoningNode**（タイプ: reasoning）

```
┌───────────────────────────┐
│ [💭] Agent Reasoning      │
├───────────────────────────┤
│ The user just sent "1"... │
│ ▸ クリックしてサマリー展開│
└───────────────────────────┘
```

- summary: agent_reasoning.text の先頭2行
- fullText: agent_reasoning.text の全文 + reasoning.summary の全文
- クリック → BottomPanel に推論テキスト全文 + サマリーを表示
- スタンドアロンAR（RIなし）: summary/fullText にARテキストのみ設定
- スタンドアロンRI（ARなし・暗号化済み）: summary/fullText に「（暗号化済み・表示不可）」を設定

**ContextDocNode**（タイプ: contextDoc）— 分岐ノード

```
┌───────────────────────────┐
│ [📄] base_instructions    │
├───────────────────────────┤
│ ▸ クリックして展開 (~20K) │
└───────────────────────────┘
```

デフォルトは折りたたみ状態。クリックでテキスト全文を展開表示。textLength から文字数を表示。

**WebSearchActionNode**（タイプ: webSearchAction）

```
┌───────────────────────────┐
│ [🔍] Web Search           │  ← call側
├───────────────────────────┤
│ query: "codex session..."  │
└───────────────────────────┘

┌───────────────────────────┐
│ [🔍] Web Search Result    │  ← end側
├───────────────────────────┤
│ action: search             │
└───────────────────────────┘
```

- call側: label = "Web Search", summary = `query: {action.query の先頭1行}`
- end側: label = "Web Search Result", summary = `action: {action.type}`
- meta（call側）: action.type, action.query, action.queries
- meta（end側）: query, action.type, action.queries

**ExternalEventNode**（タイプ: externalEvent）— ブランチノード

action ノード（function_call_output 等）の右側に step エッジで分岐配置。call_id で対応する action ノードと紐付く。

```
┌───────────────────────────┐
│ [⌨] exec_command    [✓]  │  ← exec_command_end
├───────────────────────────┤
│ find docs -type f         │
│ exit: 0 · 0.3s           │
└───────────────────────────┘

┌───────────────────────────┐
│ [🔌] figma_whoami    [✓] │  ← mcp_tool_call_end
├───────────────────────────┤
│ server: codex_apps        │
│ result: Ok                │
└───────────────────────────┘

┌───────────────────────────┐
│ [🖼] View Image           │  ← view_image_tool_call
├───────────────────────────┤
│ /Users/.../my.jpeg        │
└───────────────────────────┘
```

サブタイプごとの仕様:

| サブタイプ             | label            | icon | summary                                                    | meta                                                                        | ステータス表示                         |
| ---------------------- | ---------------- | ---- | ---------------------------------------------------------- | --------------------------------------------------------------------------- | -------------------------------------- |
| exec_command_end       | "exec_command"   | ⌨    | `{command の先頭行}` + `\nexit: {exit_code} · {duration}s` | command, exit_code, duration, cwd, process_id, aggregated_output            | exit_code=0 → ✓（緑）, 0以外 → ✕（赤） |
| mcp_tool_call_end      | "{tool}"         | 🔌   | `server: {server}` + `\nresult: {Ok/Err}`                  | invocation.server, invocation.tool, invocation.arguments, result            | result.Ok → ✓, result.Err → ✕          |
| view_image_tool_call   | "View Image"     | 🖼   | `{path のファイル名部分}`                                  | path                                                                        | なし                                   |
| collab_agent_spawn_end | "Spawn Agent"    | 👥   | `{new_agent_nickname} ({new_agent_role})`                  | new_agent_nickname, new_agent_role, prompt, model, reasoning_effort, status | status=completed → ✓                   |
| collab_close_end       | "Collab Close"   | 👥   | `{receiver_agent_nickname} ({receiver_agent_role})`        | receiver_agent_nickname, receiver_agent_role, status                        | status=completed → ✓                   |
| collab_waiting_end     | "Collab Waiting" | 👥   | `agents: {statuses のエージェント数}`                      | statuses                                                                    | なし                                   |

- ステータス表示: ノードヘッダーの右側に ✓/✕ アイコンを表示（exit_code や result に基づく）
- fullText: exec*command_end の aggregated_output 全文、mcp_tool_call_end の result 全文、collab*\* の status（完了内容）全文

### 3.6 画面詳細設計

#### 3.6.1 セッション一覧画面

**コンポーネント階層:**

```
SessionListPage
  ├── ツールバー
  │     ├── セッション件数表示
  │     ├── ソースパス表示
  │     └── SearchBox
  │
  └── DateTree
        └── YearGroup[]   (折りたたみ可能)
              └── MonthGroup[]  (折りたたみ可能)
                    └── DayGroup[]   (折りたたみ可能)
                          └── SessionRow[]  (クリックで遷移)
```

**イベントハンドリング:**

| イベント | 対象コンポーネント | 処理                                                 |
| -------- | ------------------ | ---------------------------------------------------- |
| マウント | SessionListPage    | カスタムフックでAPI呼び出し、最新年月日を展開        |
| クリック | YearGroup Header   | 展開/折りたたみ切替。arrow の向きを変更              |
| クリック | MonthGroup Header  | 展開/折りたたみ切替                                  |
| クリック | DayGroup Header    | 展開/折りたたみ切替                                  |
| クリック | SessionRow         | `/sessions/:id` へ遷移                               |
| 入力変更 | SearchBox          | セッションID、ブランチ、ディレクトリでフィルタリング |

**SearchBox フィルタリング仕様:**

- 検索対象: セッションID、cwd、branch
- 部分一致（大文字小文字区別なし）
- フィルタリングはフロントエンド側で実行（全セッションを既に取得済み）
- 空文字列の場合は全セッション表示
- 入力変更時は 200ms の debounce を適用し、再レンダリングを抑制する

**展開状態の初期値:**

- 最新の年: 展開
- 最新の月: 展開
- 最新の日: 展開
- それ以外: 折りたたみ

**SessionRow の表示ルール:**

- `parsed === true` の場合: 全項目を表示
- `parsed === false` の場合:
  - セッションID: ファイル名から抽出して表示（色薄）
  - 作業ディレクトリ〜プロバイダー: 「解析前」
  - タイムスタンプ: ファイルパスから推測して表示（色薄）
  - ファイルサイズ: 常に表示

#### 3.6.2 セッション詳細画面

**レイアウト構造:**

```
┌─ App Header (固定) ────────────────────────────────────┐
│ [← 一覧に戻る] [SessionID] [Branch] [CWD] [Time] [Export]│
├────────────────────────────────────┬───────────────────┤
│                                    │ 統計情報          │
│  React Flow Canvas                 │ ──────────────── │
│  (ズーム・パン可能)                │ ターン別Token     │
│                                    │ ──────────────── │
│                                    │ Token Count表     │
├────────────────────────────────────┴───────────────────┤
│ Bottom Panel (ノード詳細・クリック時に表示)              │
└────────────────────────────────────────────────────────┘
```

**React Flow Canvas 仕様:**

| 項目         | 設定値                                            |
| ------------ | ------------------------------------------------- |
| ズーム       | 初期: 自動フィット（fitView）。範囲: 0.1 〜 2.0   |
| パン         | ドラッグでパン。Scroll wheel でズーム             |
| ノード移動   | 無効                                              |
| コントロール | ズームイン/アウト/フィット/リセットボタン（左下） |

**RightPanel 仕様:**

| セクション             | コンポーネント     | 内容                                             |
| ---------------------- | ------------------ | ------------------------------------------------ |
| 統計情報               | StatisticsPanel    | 6つのstat-card（2×3グリッド）                    |
| ターン所要時間         | StatisticsPanel 内 | バーチャートで各ターンの所要時間を表示           |
| ターン別トークン使用量 | TurnTokenSummary   | ターンごとのカード（消費トークン内訳）           |
| token_countごとの推移  | TokenCountTable    | 行列表。スクロール可能。ターン境界セパレータ付き |

**BottomPanel 仕様:**

- デフォルト: 非表示（max-height: 0）
- ノードクリック時: スライドインで表示（max-height: 250px）
- 表示内容: クリックしたノードの meta および fullText
- token_count バッジクリック時: 紐付く全 token_count のトークン内訳（input/output/cached/reasoning）を表示。boundCount ≥ 2 の場合は各 token_count を時系列で表示

**エクスポートボタン:**

- クリック → 確認ダイアログなしでエクスポート開始
- バックエンドの `GET /api/sessions/:id/stats-image` を呼び出し、PNG画像をダウンロード
- ファイル名: `stats-{sessionID短縮}.png`

---

## 4. API詳細設計

### 4.1 GET /api/sessions

**概要:** セッション一覧を取得する

**リクエスト:** パラメータなし

**レスポンス 200:**

```json
{
  "sessions": [
    {
      "id": "019e5514-ed44-78b2-bf88-233d6e4273bf",
      "file_path": "2026/05/23/rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl",
      "cwd": "/Users/dev/projects/backend-dev-val",
      "cli_version": "0.131.0",
      "originator": "codex-tui",
      "model_provider": "openai",
      "branch": "feature/5",
      "source": "cli",
      "timestamp": "2026-05-23T13:44:55.385Z",
      "file_size": 378000,
      "file_modified_at": "2026-05-23T14:30:00.000Z",
      "parsed": true
    }
  ]
}
```

**エラーレスポンス:**

| ステータス | code           | 条件                     |
| ---------- | -------------- | ------------------------ |
| 500        | INTERNAL_ERROR | ディレクトリ読み込み失敗 |

### 4.2 GET /api/sessions/:id

**概要:** セッション詳細（React Flow形式）を取得する

**リクエスト:**

| 項目        | 値                        |
| ----------- | ------------------------- |
| Path        | /api/sessions/:id         |
| Path Params | id — セッションID（UUID） |

**レスポンス 200:**

```json
{
  "id": "019e5514-ed44-78b2-bf88-233d6e4273bf",
  "parsed_at": "2026-05-23T15:00:00.000Z",
  "nodes": [ ... ],
  "edges": [ ... ],
  "statistics": { ... },
  "token_counts": [ ... ]
}
```

**エラーレスポンス:**

| ステータス | code               | 条件                                                     |
| ---------- | ------------------ | -------------------------------------------------------- |
| 404        | SESSION_NOT_FOUND  | 指定IDのセッションファイルが存在しない                   |
| 413        | FILE_TOO_LARGE     | ファイルサイズが50MBを超過                               |
| 422        | UNSUPPORTED_FORMAT | 非対応のセッションファイル形式（Codex CLI v0.121.0未満） |
| 422        | PARSE_ERROR        | JSONLの解析に失敗                                        |
| 500        | FILE_READ_ERROR    | ファイル読み込み失敗                                     |
| 500        | INTERNAL_ERROR     | 内部エラー                                               |

### 4.3 GET /api/sessions/:id/stats-image

**概要:** セッションの統計情報を画像として取得する

**リクエスト:**

| 項目        | 値                            |
| ----------- | ----------------------------- |
| Path        | /api/sessions/:id/stats-image |
| Path Params | id — セッションID（UUID）     |

**レスポンス 200:**

- Content-Type: image/png
- Body: PNG画像バイナリ（1920×1080）
  - 上部: セッションヘッダー（セッションID短縮形式、ブランチ名、作業ディレクトリ、タイムスタンプ）
  - 下部: 6つのstat-card（2列×3行グリッド）
    - 所要時間、総トークン数、ツール呼び出し数、トークンカウント数、コンテキストウィンドウサイズ、ターン数

**エラーレスポンス:**

| ステータス | code               | 条件                                                     |
| ---------- | ------------------ | -------------------------------------------------------- |
| 404        | SESSION_NOT_FOUND  | セッションが存在しない                                   |
| 413        | FILE_TOO_LARGE     | ファイルサイズが50MBを超過                               |
| 422        | UNSUPPORTED_FORMAT | 非対応のセッションファイル形式（Codex CLI v0.121.0未満） |
| 422        | PARSE_ERROR        | セッションファイルの解析に失敗                           |
| 500        | FILE_READ_ERROR    | セッションファイルの読み込みに失敗                       |
| 500        | INTERNAL_ERROR     | 画像生成失敗                                             |

### 4.4 エラーレスポンス共通形式

```json
{
  "error": "セッションファイルの読み込みに失敗しました",
  "code": "FILE_READ_ERROR"
}
```

| フィールド | 型     | 説明                                   |
| ---------- | ------ | -------------------------------------- |
| error      | 文字列 | ユーザー向けエラーメッセージ（日本語） |
| code       | 文字列 | エラーコード（英語・大文字）           |

---

## 5. JSONLパーサー処理詳細

### 5.1 パース全体フロー

```
入力: ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl
  │
  ├─ Step 1: ファイルチェック
  │   ├─ ファイル存在確認 → 404
  │   ├─ ファイルサイズ > 50MB → 413
  │   └─ ファイルサイズ == 0 → 422 ("セッションファイルが空です")
  │
  ├─ Step 2: 行ごとパース
  │   各行をJSONとして読み込み、トップレベル type で RecordType を判定:
  │   ├─ "session_meta" → SessionMetaPayload にデシリアライズ
  │   ├─ "turn_context" → TurnContextPayload にデシリアライズ
  │   ├─ "event_msg" → EventMsgPayload にデシリアライズ
  │   │   └─ payload.Type で EventMsgType を判定
  │   ├─ "response_item" → ResponseItemPayload にデシリアライズ
  │   │   └─ payload.Type で ResponseItemType を判定
  │   └─ その他 → 未知タイプ（rawのまま保持）
  │   JSONパース失敗 → 警告ログ + スキップ
  │
  ├─ Step 2.5: 非対応形式チェック
  │   session_meta の cli_version を確認:
  │   ├─ v0.121.0 未満 → エラー（UNSUPPORTED_FORMAT）
  │   └─ v0.121.0 以上 → 続行
  │
  ├─ Step 3: ターン分割 + thread_name収集
  │   ├─ currentTurn = null
  │   ├─ session_meta → ターン外レコード
  │   ├─ thread_name_updated → ParsedSession.ThreadName に設定（最新値で上書き）
  │   ├─ task_started → 新しい Turn を開始
  │   │   ├─ Turn{Index: turnIndex++, TurnID: payload.turn_id}
  │   │   └─ currentTurn = 新しいTurn
  │   ├─ task_complete → currentTurn.TaskComplete に設定
  │   │   └─ currentTurn = null（ターン終了）
  │   ├─ turn_aborted → currentTurn.TaskComplete に設定
  │   │   ├─ Aborted = true（中断フラグ）
  │   │   └─ currentTurn = null（ターン終了）
  │   └─ その他 → currentTurn.Records に追加
  │       └─ currentTurn == null → ターン外レコードとして扱う
  │
  ├─ Step 4: ターン内レコード分類
  │   for turn in turns:
  │     for record in turn.Records:
  │       ├─ turn_context → turn.TurnContext
  │       ├─ event_msg(user_message) → turn.UserEventMsg
  │       ├─ event_msg(agent_message) → turn.AgentMessages
  │       ├─ event_msg(agent_reasoning) → 遅延マージ（Step 5で処理）
  │       ├─ response_item(reasoning) → 遅延マージ（Step 5で処理）
  │       ├─ response_item(message):
  │       │   ├─ role=developer → turn.DeveloperMessages
  │       │   ├─ role=user → turn.UserMessages
  │       │   └─ role=assistant → バッチ中間として扱う
  │       ├─ response_item(function_call) → functionCalls に一時保持
  │       ├─ response_item(function_call_output) → functionOutputs に一時保持
  │       ├─ response_item(web_search_call) → webSearchCalls に一時保持
  │       ├─ event_msg(web_search_end) → webSearchEnds に一時保持
  │       ├─ event_msg(item_completed) → turn.ItemCompleted に追加
  │       ├─ event_msg(token_count) → tokenCounts に追加
    │       └─ その他（error, collab_*, exec_command_end 等）
  │          ├─ call_id を持つレコード → externalEventRecords に一時保持
  │          └─ call_id を持たないレコード → genericRecords に一時保持
  │
  ├─ Step 5: reasoning ペアリング
  │   for turn in turns:
  │     ├─ agent_reasoning レコードのリストを取得
  │     ├─ response_item(reasoning) レコードのリストを取得
  │     ├─ 出現順序で 1:1 ペアリング:
  │     │   reasoningPairs[i] = {agentReasoning, reasoningSummary}
  │     └─ ペアリングできない場合は単独で保持:
  │         ├─ ARのみ（agent_reasoning が余る）:
  │         │   standaloneReasoning として保持
  │         │   summary/fullText に ARテキストを設定
  │         └─ RIのみ（response_item(reasoning) が余る、またはAR不在）:
  │             standaloneReasoning として保持
  │             summaryが空の場合はプレースホルダーを設定
  │
  ├─ Step 6: バッチ検出
  │   for turn in turns:
  │     detectBatches(turn.Records) → turn.Batches
  │
  └─ Step 7: token_count 紐付け
      for turn in turns:
        for each tokenCount in turn.TokenCounts:
          ├─ 直前のレコードを特定:
          │   turn.Records を逆順に走査し、
          │   最初の非-token_count レコードを取得
          ├─ BoundToRecord = 直前レコード（TypedRecordへの参照）
          │   ※ 連続token_countの場合、複数のtoken_countが
          │   同一レコードに紐付けられることを許容する
          │   ※ 同一ノードに複数紐付く場合、FlowGraph生成（§2.6 Step 4c）
          │     で TokenBadgeData に集約する
          └─ TurnIndex = turn.Index

      ※ BoundToRecord はパース時点では TypedRecord への参照として保持し、
      FlowGraph 生成フェーズ（§2.6 Step 3）でレコード→ノードIDの
      マッピングテーブルを用いて BoundToNodeID（文字列）に解決する

      ※ turn.Records に非token_countレコードが存在しない場合
      （ターン内レコードが空の状態でtoken_countが出現）、
      BoundToRecord を null とし、BoundToNodeID を空文字列とする。
      フロントエンドでは当該token_countにバッジを表示せず、
      token_count表には表示する
```

### 5.2 対応対象外のセッションファイル

Codex CLI v0.121.0 未満で生成されたセッションファイル（`task_started`/`task_complete` を含まない形式）は対応範囲外とする。

- **セッション一覧**: 対応対象外のファイルも一覧に表示する
- **セッション詳細**: パース時にエラーを返す（§6.1 `UNSUPPORTED_FORMAT` を参照）
- **判定方法**: `session_meta` の `cli_version` が v0.121.0 未満かどうかで判定する

### 5.3 破損ファイルの扱い

- 1行のJSONパースに失敗 → 該当行をスキップ、警告ログを出力
- `session_meta` が存在しない → 警告ログのみ、解析は継続（IDはファイル名から推測）
- `task_started` なしで `task_complete` が出現 → 孤立 `task_complete` として扱い、ターン外レコードに分類
- 不正なJSONレコードが50%を超える → `PARSE_ERROR` としてエラーを返す

---

## 6. エラー処理設計

### 6.1 バックエンドエラー処理

APIエラーは以下の構造で返す。

| エラー               | ステータス | code               | メッセージ                                                                 |
| -------------------- | ---------- | ------------------ | -------------------------------------------------------------------------- |
| セッション未検出     | 404        | SESSION_NOT_FOUND  | 指定されたセッションが見つかりません                                       |
| ファイルサイズ超過   | 413        | FILE_TOO_LARGE     | ファイルサイズが上限（50MB）を超えています                                 |
| 非対応形式           | 422        | UNSUPPORTED_FORMAT | このセッションファイルの形式には対応していません（Codex CLI v0.121.0未満） |
| パース失敗           | 422        | PARSE_ERROR        | セッションファイルの解析に失敗しました                                     |
| ファイル読み込み失敗 | 500        | FILE_READ_ERROR    | セッションファイルの読み込みに失敗しました                                 |
| 内部エラー           | 500        | INTERNAL_ERROR     | 内部エラーが発生しました                                                   |

### 6.2 フロントエンドエラー処理

#### 6.2.1 セッション一覧画面

| エラーコード           | UI表示メッセージ                 |
| ---------------------- | -------------------------------- |
| NETWORK_ERROR          | サーバーに接続できません         |
| TIMEOUT                | リクエストがタイムアウトしました |
| INTERNAL_ERROR         | 内部エラーが発生しました         |
| 一覧が空（エラーなし） | セッションが見つかりません       |

- 表示方法: 画面中央にエラーメッセージ + 再試行ボタン
- 一覧が空の場合は空状態メッセージのみ（再試行ボタンなし）

#### 6.2.2 セッション詳細画面

| エラーコード       | UI表示メッセージ                                 |
| ------------------ | ------------------------------------------------ |
| SESSION_NOT_FOUND  | 指定されたセッションが見つかりません             |
| FILE_TOO_LARGE     | ファイルサイズが大きすぎます（上限50MB）         |
| UNSUPPORTED_FORMAT | このセッションファイルの形式には対応していません |
| PARSE_ERROR        | セッションファイルの解析に失敗しました           |
| FILE_READ_ERROR    | 内部エラーが発生しました                         |
| INTERNAL_ERROR     | 内部エラーが発生しました                         |
| NETWORK_ERROR      | サーバーに接続できません                         |
| TIMEOUT            | リクエストがタイムアウトしました                 |

- 表示方法: 画面中央にエラーメッセージ + 再試行ボタン
- ApiError.code でエラーコードを判定し、対応するメッセージを表示する

#### 6.2.3 エクスポートエラー

エクスポートボタン（stats-image取得）のエラーはトースト通知で対応する。画面全体のエラー表示は行わない。

| 状況 | 通知type | メッセージ                               |
| ---- | -------- | ---------------------------------------- |
| 成功 | success  | 画像をエクスポートしました               |
| 失敗 | error    | ApiErrorのエラーコードに応じたメッセージ |

#### 6.2.4 ローディング中

スピナーまたはスケルトンローディングを表示する

---

## 7. キャッシュ設計

### 7.1 キャッシュファイル形式

キャッシュ先: `~/.codex-display/{sessionID}.json`

ファイルの中身は `GET /api/sessions/:id` のレスポンスと同一形式。

### 7.2 再パース判定ロジック

1. キャッシュファイルが存在しない → 再パースが必要
2. JSONLファイルの最終更新日時 > キャッシュファイルの最終更新日時 → 再パースが必要
3. キャッシュファイルの読み込み（JSONデコード）に失敗 → キャッシュを破棄して再パースが必要
4. 上記以外 → キャッシュを使用

### 7.3 キャッシュ書き込み失敗時の挙動

エラーログを出力し、キャッシュなしで動作を継続する。次回アクセス時に再パースが走る。

---

## 8. ビルド・実行設計

### 8.1 開発時構成

- フロントエンド: Vite dev server（ポート 5173）
- バックエンド: Go server（ポート 8080）
- Viteのプロキシ設定で `/api/*` を Go server に転送

### 8.2 プロダクション構成

- フロントエンド: ビルド成果物を静的ファイルとして生成
- バックエンド: 生成された静的ファイルを配信
- 単一のGoバイナリでフロントエンド + API を提供
- 起動コマンド実行 → `http://localhost:8080` でアクセス可能

### 8.3 Wails移行への考慮

- バックエンドのハンドラ関数は HTTPに直接依存しないインターフェースで設計する
- Wails移行時は HTTPレイヤーを WailsのIPCに差し替える
- フロントエンドのAPIクライアントを抽象化し、fetch と Wails呼び出しを切り替え可能にする
