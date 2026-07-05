# ADR 0032: セッション一覧に provider 境界を導入する

## ステータス

承認済み

## コンテキスト

セッション一覧はこれまで Codex の `~/.codex/sessions` のみを対象にしていた。
Claude Code は `projects/<encoded-cwd>/*.jsonl` に transcript を保存し、Codex の
`session_meta` や詳細解析用レコードとは形式が異なる。一方で、一覧画面では
プロジェクト、日付、セッション単位の軽量な情報だけを表示できればよい。

## 決定

- `SessionSummary` と一覧APIに provider 境界を導入し、値は `codex` / `claude` とする。
- `ListSessions(query, year, month)` は Codex 用の後方互換APIとして維持する。
- 新しい一覧画面は `ListSessionsByProvider(provider, query, year, month)` を使う。
- Claude Code は `$CLAUDE_CONFIG_DIR/projects` を優先し、未設定時は `~/.claude/projects` を探索する。
- Claude Code 一覧では `sessionId`, `cwd`, 最初の `timestamp`, `message_count`,
  `tool_call_count`, `total_cost_usd`, `encoded_project` を transcript から抽出する。
- 詳細画面のReact Flow解析は Codex セッションを対象に維持し、Claude Code セッションは
  一覧では `parsed=false` として扱う。
- キャッシュキーは provider を区別できる形にし、Codex は既存の `{sessionID}.json` を維持する。
  Claude Code などは `{provider}-{sessionID}.json` を使い、同一IDの衝突を防ぐ。

## 理由

一覧表示のために Claude Code transcript を Codex 詳細解析モデルへ変換すると、
詳細画面のドメイン境界とキャッシュスキーマを不要に広げてしまう。
provider を一覧APIとDTOの境界に持たせることで、既存Codex体験を保ったまま
Claude Code の軽量一覧表示を追加できる。

## 結果

- フロントエンドは Codex / Claude Code を切り替えて一覧を取得できる。
- Codex の既存キャッシュファイル名と詳細解析の互換性を維持できる。
- Claude Code の詳細解析やReact Flow変換は将来の別決定として扱える。

## 関連決定

- ADR 0004: React Flow形式でセッション詳細をキャッシュする
- ADR 0008: セッション一覧をツリービューで表示する
