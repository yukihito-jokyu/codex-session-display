# 0033 Claude Code transcript detail normalization

## Status

Accepted

## Context

ADR 0032 では Claude Code 対応を一覧表示に限定し、詳細画面の React Flow 変換は将来の別決定として扱った。

Issue 226 では Claude Code セッション詳細画面で transcript を時系列に確認し、thinking、assistant text、tool_use、tool_result、usage、cost を既存の詳細画面で扱う必要がある。Claude Code transcript は Codex の `session_meta` / `turn_context` / `response_item` 構造を持たないため、Codex parser の turn 構築へ直接流用すると provider 固有の意味が失われる。

## Decision

Claude Code transcript は provider 固有の正規化ステップで `SessionDetailResponse` に変換する。

- `user` の text content をターン境界として扱う。
- `assistant.message.content` の `thinking`, `text`, `tool_use` を timeline item と node に分ける。
- `tool_use.id` と `tool_result.tool_use_id` を照合し、対応 edge と timeline detail を作る。
- usage は `message.id` 単位で重複排除し、`TokenCountEntry` と `Statistics` に反映する。
- Claude 固有の `cache_read_input_tokens` と `total_cost_usd` は `TranscriptStats` として返す。
- 詳細 API は後方互換の `GetSessionDetail(id)` を Codex 固定で残し、provider 指定の `GetSessionDetailByProvider(provider, id)` を追加する。
- provider 別キャッシュを使い、Claude 詳細は `claude-{sessionID}.json` に保存する。

## Consequences

- 既存の Codex 詳細画面と E2E の入口を維持したまま、Claude Code の transcript も同じ詳細 DTO で表示できる。
- Claude Code の turn は Codex の turn と完全に同じ意味ではないため、provider 固有の解釈は正規化層に閉じ込める。
- `SessionDetailResponse` のキャッシュスキーマを更新し、古い詳細キャッシュは再生成する。
