# ADR 0012: React フロントエンドのディレクトリ構造と命名規則の決定

## ステータス

承認済み

## コンテキスト

Wails (Go + React) を用いたデスクトップアプリケーション開発において、React フロントエンドのディレクトリ構造、コンポーネント配置方針、および CSS Modules のファイル命名規則を決定する必要がある。
本アプリケーションは、React Flow を用いたセッションログの複雑な可視化を行うため、多数のカスタムノードや状態管理が必要となる。これらを整理し、保守性と拡張性を高めるための構造を定義する。

## 決定

React フロントエンドの構成として、以下のルールを導入する。

### 構成ルール

1. **機能（Feature）ベースのディレクトリ構造の採用**
   - ドメイン・画面単位で機能を集約する `features/` ディレクトリを導入する。
   - セッション一覧および詳細機能に関連するコンポーネント、フック、型定義は `src/features/sessions/` 配下に集約し、凝集度を高める。
   - **フック（コンポーザブル）の配置**: 
     - 機能に特化したカスタムフック（コンポーザブル）は、機能ディレクトリ配下の `hooks/`（例: `src/features/sessions/hooks/`）に配置する。
     - 複数機能から共通利用される汎用フック（例: `useTheme`）は、グローバル共通の `src/hooks/` に配置する。


2. **カスタムノードのグローバル共通UIとしての配置**
   - React Flowで表示される多数のカスタムノード（`ActionNode`, `ReasoningNode`, `TurnEventNode` 等）は、将来的な再利用性や見通しの良さを考慮し、グローバル共通UIとして `src/components/nodes/` 配下に配置する。
   - 各カスタムノードはフォルダ（例: `src/components/nodes/ActionNode/`）に分割し、ロジックとスタイルをカプセル化する。

3. **CSS Modulesのファイル命名規則（PascalCaseかつコンポーネント名と完全一致）**
   - CSS Modulesのファイル名は、TypeScriptのコンポーネント名と完全に一致させる（PascalCase）。
   - 例: `ActionNode.tsx` に対応するスタイルファイルは `ActionNode.module.css` とする。

4. **CSS Modulesのクラス名命名規則（camelCase）**
   - CSS Modules内で定義するクラス名は `camelCase` とする。
   - 例: `.actionButton` や `.titleText`。
   - これにより、JS/TS側で `styles.actionButton` のようにドット記法で直感的にアクセスできる。ファイルスコープを持つため、BEM等の複雑な命名は避け、フラットでシンプルな命名を推奨する。

## 理由

### 採用理由

1. **機能ごとの凝集度向上**
   - 「セッション」という本アプリのコアビジネスドメインに対し、一覧表示や詳細表示、キャンバス制御などの関連ロジックを `features/sessions/` にまとめることで、影響範囲が局所化され開発効率が向上する。
2. **コンポーネントの整理と見通しの確保**
   - カスタムノードは13種類以上に及び、これらを機能ディレクトリ直下に置くとファイルが乱雑になる。グローバル共通UIの `components/nodes/` に分離することで、キャンバス本体（`features/sessions/components/FlowCanvas/`）とノード描画の責務を綺麗に分離できる。
3. **エディタにおける識別性の向上**
   - スタイルファイルを `style.module.css` などとせず `[ComponentName].module.css` とすることで、エディタで複数のスタイルファイルを開いた際にもタブ名だけで対象コンポーネントを判別できる。
4. **JS/TSにおける参照のスマートさ**
   - クラス名を `camelCase` にすることで、JavaScript/TypeScript側でのアクセス時に `styles.myClass` とドット記法を用いることができ、ブラケット表記（`styles['my-class']`）を避けることができる。

## 結果

### ディレクトリ構造

```
frontend/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html
├── src/
│   ├── main.tsx                      (Viteエントリポイント)
│   ├── App.tsx                       (メインアプリケーション、ルーター、Providers設定)
│   ├── App.module.css                (グローバルスタイル・ベースレイアウト)
│   │
│   ├── components/                   (グローバル共通UIコンポーネント)
│   │   ├── ui/                       (ボタン、インプットなどのアトミックUI)
│   │   │   └── Button/
│   │   │       ├── Button.tsx
│   │   │       └── Button.module.css
│   │   │
│   │   └── nodes/                    (React Flow カスタムノード群)
│   │       ├── SessionMetaNode/
│   │       │   ├── SessionMetaNode.tsx
│   │       │   └── SessionMetaNode.module.css
│   │       ├── TurnEventNode/
│   │       ├── TurnContextNode/
│   │       ├── ContextDocNode/
│   │       ├── UserMessageNode/
│   │       ├── AgentMessageNode/
│   │       ├── DeveloperMessageNode/
│   │       ├── UserApiMessageNode/
│   │       ├── ReasoningNode/
│   │       ├── ActionNode/
│   │       │   ├── ActionNode.tsx
│   │       │   └── ActionNode.module.css
│   │       ├── WebSearchActionNode/
│   │       ├── ItemCompletedNode/
│   │       ├── ExternalEventNode/
│   │       ├── GenericNode/
│   │       └── TokenBadge/           (カスタムノード等に紐づくトークンバッジ)
│   │
│   ├── features/                     (画面・機能モジュール)
│   │   └── sessions/                 (セッション一覧・詳細機能)
│   │       ├── components/           (セッション機能固有のコンポーネント)
│   │       │   ├── SessionListPage/  (セッション一覧画面)
│   │       │   ├── SessionDetailPage/ (セッション詳細画面)
│   │       │   ├── FlowCanvas/       (React Flow キャンバスコンポーネント)
│   │       │   ├── RightPanel/       (統計・トークンカウント情報パネル)
│   │       │   └── BottomPanel/      (ノード詳細表示パネル)
│   │       ├── hooks/                (セッション機能固有のカスタムフック)
│   │       │   ├── useSessions.ts    (セッション一覧取得・検索)
│   │       │   └── useSessionDetail.ts (セッション詳細取得・状態管理)
│   │       ├── types.ts              (セッション機能固有の型定義)
│   │       └── index.ts              (外部公開インターフェース)
│   │
│   ├── hooks/                        (グローバル共通カスタムフック)
│   │   └── useNotification.ts
│   │
│   ├── styles/                       (グローバルスタイル・テーマ設定)
│   │   ├── variables.css             (CSSカスタムプロパティ/変数)
│   │   └── index.css                 (CSSリセットと共通ベーススタイル)
│   │
│   └── wailsjs/                      (Wails自動生成のGoバインディング)
│       └── go/
│           └── main/
│               └── App.js
```

## 関連決定

- [[ADR-0011: クリーンアーキテクチャに準拠したバックエンドのディレクトリ構造とパッケージ構成]]
