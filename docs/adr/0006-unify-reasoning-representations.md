# ADR 0006: reasoningの二重表現の統合

## ステータス

承認済み

## コンテキスト

Codex CLIのJSONLでは、同じ推論プロセスが2つのレコード型で表現される：

- `event_msg(type=agent_reasoning)`: 推論テキスト全文
- `response_item(type=reasoning)`: 推論サマリー

これらをどのように可視化するか決定する必要がある。

## 決定

`agent_reasoning` と `response_item(type=reasoning)` を**統合して1つのノード**として扱う。

- ハーネス上には `agent_reasoning` のテキストを表示
- クリック展開時には `response_item(reasoning)` のサマリーも併せて表示

## 理由

### 採用理由

1. **同一推論プロセスの異なる表現**
   - 両者が同じ推論プロセスの異なる表現であり、分離しても意味がない

2. **要件の両立**
   - 推論テキスト全文（`agent_reasoning`）を表示しつつ、サマリーも表示したいという要件がある

3. **UIのシンプルさ**
   - ノード数を増やさず、UIをシンプルに保てる

4. **ユーザーの混乱回避**
   - 2つのノードを並列表示すると、どちらが「正しい」推論内容かユーザーが混乱する可能性がある

### 検討した代替案

| 案 | 内容 | 採用見送りの理由 |
|------|------|------------------|
| 案B | agent_reasoning と response_item(reasoning) を別々のノードとして表示し、それぞれの役割を明確にする | 同じ推論プロセスの2つの表現を分離しても意味がない、ノード数が増える、ユーザーが混乱する可能性 |

## 結果

- `agent_reasoning` と `response_item(reasoning)` を出現順で1:1ペアにする
- ペアリングできない場合は単独で保持
- 統合ノードの `summary` には `agent_reasoning.text` の先頭を設定
- 統合ノードの `fullText` には `agent_reasoning.text` 全文 + `reasoning.summary` 全文を設定
- スタンドアロンAR（RIなし）: `agent_reasoning` テキストのみ
- スタンドアロンRI（ARなし・暗号化済み）: 「（暗号化済み・表示不可）」を表示

## 関連決定

なし
