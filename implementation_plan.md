# サブエージェント メッセージ送信時のノード・タイムライン遷移対応計画

本計画は、親エージェントが子エージェント（サブエージェント）の作成後に、再度メッセージを送信した際（`send_input` ツール呼び出し）やエージェントを再開した際（`resume_agent` ツール呼び出し）に、詳細画面のフローキャンバスおよびタイムライン上にサブエージェントへの遷移リンク付きノード（`collabAgent` 型）を正しく生成・表示するためのものです。

---

## ユーザーレビューが必要な事項

> [!NOTE]
> - `send_input`（メッセージ送信）および `resume_agent`（エージェント再開）ツール呼び出し時も、起動時（`spawn_agent`）と同様に `collabAgent` 型ノードを割り当てることで、既存のフロントエンド機能「サブエージェントを表示」ボタンを流用し、子セッションへのジャンプが可能になります。
> - タイムライン上でも `collab` 種類のアイテムとして扱い、かつ引数の `target`/`id` から対象サブエージェントIDを `"Thread ID"` 詳細項目として設定することで、タイムラインからの遷移も可能にします。

---

## 提案する変更内容

### 1. バックエンド (Go)

#### [MODIFY] [get_session_detail.go](file:///Users/yukihito/Documents/github_projects/codex-session-display/internal/usecase/get_session_detail.go)
- **`batchTimelineUnit` 関数の修正**:
  - `send_input` を検知した際、タイムラインの `Kind` を `"collab"`、`Label` を `"サブエージェント メッセージ送信"` に設定し、`Details` に `Thread ID`（引数 `target`）と `Message`（引数 `message`）を含めるようにします。
  - `resume_agent` を検知した際、タイムラインの `Kind` を `"collab"`、`Label` を `"サブエージェント再開"` に設定し、`Details` に `Thread ID`（引数 `id`）を含めるようにします。
- **`Execute` 関数の修正 (フローキャンバス用ノード生成部分)**:
  - `displayUnitBatch` の `batch.CallRecords` ループ内において、`call.ResponseItem.Name` が `"send_input"` または `"resume_agent"` の場合も `nodeType = "collabAgent"` とし、`icon = "🤖"`、適切な `label` (`"Send Input to Subagent"` / `"Resume Subagent"`) を設定します。
  - メタデータ `meta` に `"new_thread_id"` として対象のサブエージェントID（`target` または `id`）を設定します。
- **`Execute` 関数の修正 (子セッションIDの収集部分)**:
  - キャッシュされていない場合や、`spawn_agent` ログとは別個に `send_input`/`resume_agent` が現れた場合に備えて、これらのツール引数からも対象サブエージェントIDを抽出し、`childSessionIDs` リストに追加する処理を追加します。

#### [MODIFY] [get_session_detail_test.go](file:///Users/yukihito/Documents/github_projects/codex-session-display/internal/usecase/get_session_detail_test.go)
- **ユニットテストの追加**:
  - `TestGetSessionDetailUseCase` に、`send_input` および `resume_agent` ツール呼び出しを含むレコードをモックしたテストケースを追加し、キャンバスノードが `collabAgent` として出力され、タイムラインアイテムが `collab` として出力され、子セッションIDが正しく集約されることを検証します。

---

## 検証計画

### 自動テスト
- 修正後のGoテストを実行します。
  ```bash
  go test -v ./internal/usecase/...
  ```
- リントチェックを実行します。
  ```bash
  task lint
  ```
- 既存のE2Eテストを実行します。
  ```bash
  task test:e2e
  ```

### 手動検証
- 開発用サーバーを実行し、実際の `send_input` 呼び出しのあるセッション詳細画面において、フローキャンバス上に「サブエージェントを表示」ボタンを持つノードが表示されること、およびタイムライン上から子エージェント詳細へ正しく遷移できることを確認します。
