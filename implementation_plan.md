# Issue #171 実装計画

## 目的

セッション詳細画面の表示・操作仕様を維持したまま、下部詳細領域を次の3コンポーネントへ分割する。

- `BottomPanel`: パネル枠、ヘッダー、閉じる操作、単一/分割レイアウト、separator
- `NodeDetail`: ノードのメタデータ、全文、空状態
- `TokenDetail`: トークンサマリー、紐付く各 `token_count` エントリ

各コンポーネントは同名のCSS Moduleを所有し、`SessionDetailMainContent` はキャンバス、下部パネル、右パネルの組み立てに専念する。

## 維持する公開インターフェース

- `SessionDetailPage` と `useSessionDetail` の状態・操作インターフェースは変更しない。
- Playwrightが利用する次のDOM契約を維持する。
  - `bottom-panel`
  - `bottom-panel-single` / `bottom-panel-split`
  - `bottom-panel-node-detail`
  - `bottom-panel-token-detail`
  - `bottom-panel-resizer`
- 表示文言、閉じる操作、トークンバッジ選択、右パネルからの選択連携を維持する。
- separatorの `role`、`aria-orientation`、`aria-label`、`aria-valuemin`、`aria-valuemax`、`aria-valuenow`、`tabIndex` を維持する。

## Props境界

### BottomPanel

- 選択ノード
- 分割表示の有無と分割比率
- ノードに紐付くトークン情報
- 分割コンテナref
- 閉じる、ポインターリサイズ開始、キーボードリサイズの各コールバック

### NodeDetail

- 表示対象ノード
- 分割時の幅指定

### TokenDetail

- 紐付く `TokenCountEntry` の配列
- 最新の累積トークン使用量

## TDDサイクル

1. キャラクタリゼーション
   - separatorの左右矢印キー操作とARIA値更新をPlaywrightで明示的に検証する。
   - ノード詳細のメタデータ、全文、空状態の既存表示を検証する。
2. `BottomPanel` 抽出
   - 下部パネル枠とリサイズ操作を移し、対象E2Eを通す。
3. `NodeDetail` 抽出
   - ノード詳細表示と固有CSSを移し、対象E2Eを通す。
4. `TokenDetail` 抽出
   - トークン表示と数値整形を移し、対象E2Eを通す。
5. リファクタリング
   - `SessionDetailMainContent.module.css` に全体レイアウトだけを残す。
   - Propsや重複を整理し、Biomeと全セッション詳細E2Eを実行する。

## 検証

- `frontend` で `npm run lint`
- `task test:e2e:detail -- session-detail.spec.ts --grep "<対象テスト>"`
- 最終確認で `task test:e2e`

## ドキュメント判断

既存のADR 0012の機能別コンポーネント配置とCSS Module方針に沿う責務分割であり、新しいアーキテクチャ決定や仕様変更はないため、新規ADRおよび仕様書更新は不要と判断する。
