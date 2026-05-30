# フロントエンド開発ルール (Frontend Development Rules)

このドキュメントは、`codex-session-display` の React / TypeScript フロントエンド開発における設計原則、コーディング規約、およびテスト方針を定義します。エージェントはコードの追加や変更を行う際、常にこのルールを遵守しなければなりません。

---

## 1. 技術スタックと構造

1. **基本構成**
   - React (v18) + TypeScript + Vite によるSPA構成。
   - コンポーネントは `frontend/src/components/ui/` 配下に機能単位のディレクトリを作成して配置します（例: `Toolbar/Toolbar.tsx`）。

2. **ルーティング**
   - デスクトップアプリ（Wails）特有の制約（ローカルのファイルプロトコルで動作する）があるため、`BrowserRouter` ではなく **`HashRouter`**（`react-router-dom`）を必ず使用してください。

3. **Wails との通信 (IPC)**
   - バックエンドのGoメソッドを呼び出す際は、Wailsが自動生成する `frontend/wailsjs/go/` 配下のバインディング（例: `import { ListSessions } from "wailsjs/go/main/App"`）を使用します。
   - `wailsjs` ディレクトリ内のファイルを直接手動で編集しないでください。

4. **状態管理 (State Management)**
   - 原則として、Reactの標準Hooks（`useState`, `useReducer`, `useEffect` 等）や `Context API` を使用して状態を管理します。
   - 状態管理が複雑化し、複数コンポーネント間で大規模な状態の同期が必要になった場合のみ、`Zustand` などの軽量な状態管理ライブラリの導入を許容します。ReduxやMobXなどの重厚なフレームワークの導入は避けてください。

---

## 2. スタイリング規約 (CSS)

1. **CSS Modules の使用**
   - スタイルのスコープをコンポーネント内に閉じるため、**CSS Modules**（`*.module.css`）を基本とします。
   - CSSファイルはコンポーネントファイルと同じディレクトリに配置します（例: `Toolbar.tsx` と `Toolbar.module.css`）。
   - クラス名はキャメルケースを推奨します（例: `styles.listPage`）。

2. **デザインシステムと共通変数**
   - 色、背景、枠線、各種状態（ノードのカテゴリ別カラー等）を定義する際は、`frontend/src/styles/variables.css` の CSS変数を必ず使用してください。
   - 例: `background: var(--bg-header);`, `border: 1px solid var(--border-color);`
   - ハードコーディングされたカラーコード（例: `#30363d` などの生の値）の使用は極力避けてください。

3. **ライト・ダークテーマのサポート**
   - `body.light-theme` クラスの有無によって変数が切り替わるため、テーマに関わらず正しく表示されるようにCSS変数を活用したスタイリングを徹底します。

4. **スタイリング手法**
   - 原則として Vanilla CSS（通常のCSS機能）を使用し、Flexbox / Grid を活用してモダンでプレミアムなレスポンシブデザインを実現します。
   - ユーザーからの明示的な指示がない限り、Tailwind CSS などの外部ユーティリティファーストフレームワークは導入しないでください。

---

## 3. コードスタイルとツールチェーン (Biome)

1. **コードのフォーマットと静的解析**
   - コードの整形および静的解析ツールとして **Biome** を使用します。プロジェクトルートではなく `frontend` ディレクトリ配下で動作します。
   - コード変更後は必ず `npm run lint`（チェック）または `npm run lint:write`（自動修正）を実行してください。

2. **コードスタイルルール**
   - インデント: **タブ (Tab)** を使用します。
   - 文字列のクォート: JavaScript / TypeScript では **ダブルクォート (")** を使用します。
   - インポート順序: Biome の `organizeImports` が有効なため、自動的にソートされます。

3. **厳格な静的解析ルール (Strict Biome Linter Rules)**
   - **VCS 連携**: 有効化。プロジェクトルートの `.gitignore` を自動検知して除外対象を連動させます。
   - **非 null アサーション禁止 (`noNonNullAssertion`)**: エラーとして有効。TypeScript の `!` アサーションはランタイムクラッシュの要因となるため原則禁止し、安全な存在チェック（`if`）やオプショナルチェイニング（`?.`）を使用します。
   - **フック依存配列の厳格化 (`useExhaustiveDependencies`)**: エラーとして有効。フック内での古いクロージャ問題を防ぐため、依存配列は漏れなく指定する必要があります。関数の無駄な再生成を防ぐため `useCallback` / `useMemo` を適切に使用してください。
   - **アクセシビリティ (`a11y`)**: `useSemanticElements`（セマンティック要素の強制）のみ `off` とし、その他の推奨アクセシビリティルール（ボタンの `type` 属性必須化、SVGの代替テキスト `<title>` 追加、クリックイベントを持つ `div` 等への `tabIndex` および `onKeyDown` によるキーボード操作対応）はエラーとして有効です。
   - **CSS `!important` 禁止 (`noImportantStyles`)**: エラーとして有効。CSS カスケードの崩壊を防ぐため `!important` は使用せず、CSS セレクタの組み合わせ等により詳細度を制御します。

---

## 4. テスト方針 (Playwright E2E)

1. **E2E テストの配置と分割**
   - Playwright による E2E テストは `frontend/tests/e2e/` ディレクトリに配置します。
   - テストケースが肥大化するのを防ぐため、機能ごとにファイルを適切に分割します（例: `session-list.spec.ts`, `session-detail.spec.ts`）。

2. **Wails API のモック化**
   - E2E テストはバックエンドのモックを利用してフロントエンド単体でテストを実行できるように設計します。
   - `frontend/tests/e2e/helpers/mock-wails.ts` を使用して、Wails API のモック（`window.go` などのインジェクション）を事前に行います。テストを追加する際は、必要に応じてこのモック定義を拡張してください。

3. **テストの更新タイミング**
   - UIのDOM構造（セレクタ）の変更、画面遷移や新機能の追加、またはWails API（Goバックエンドからエクスポートされるバインディング）に変更があった場合、**必ずそれらの変更と同時に（同一のコミットやプルリクエスト内で）** E2Eテストおよびモックコード（`mock-wails.ts`）を更新してください。常にテストが通る状態を維持します。

---

## 5. コード変更時の検証フロー

フロントエンドのコード（`src/` 内のコンポーネントやテストコードなど）を書き換えた場合、必ずコミットやプッシュを行う前に以下の検証（Lintおよびテスト）を実行してください。

1. **リンターの実行**
   - `frontend` ディレクトリ配下で `npm run lint`（Biome による静的解析とフォーマットチェック。または `task lint`）を実行し、問題がないか確認します。自動修正を行う場合は `npm run lint:write` を実行します。
2. **テストの実行**
   - `npx playwright test`（または `task test:e2e`）を実行し、すべてのE2Eテストがパスすることを確認します。

