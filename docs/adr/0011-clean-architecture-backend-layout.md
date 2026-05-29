# ADR 0011: クリーンアーキテクチャに準拠したバックエンドのディレクトリ構造とパッケージ構成

## ステータス

承認済み

## コンテキスト

Wails (Go + React) を用いたデスクトップアプリケーション開発において、バックエンドのディレクトリ構造およびパッケージ構成を決定する必要がある。
将来的なUIフレームワークの変更（Wailsからの移行やCLI化）や、ユニットテストの容易性を担保するため、Wails（外部フレームワーク）への依存と、JSONLのパースや画像生成などのコアビジネスロジックを分離したい。

## 決定

クリーンアーキテクチャの精神（依存性分離、依存性逆転の原則）を採用しつつ、Goプロジェクトとして不必要に階層が深くならないよう、フラット化して最適化したパッケージ構成を導入する。

### 構成ルール
1. **レイヤーの整理**
   - コアのドメインロジックやデータ定義を `domain` に、ユースケースを `usecase` に、データアクセスを `repository` に、共通の画像描画エンジン等を `utils/exporter` に配置する。
2. **中間ディレクトリの排除**
   - `usecase` 内の `port/` や `interactor/` などの細分化したディレクトリは作成せず、`usecase` 直下にポート（インターフェース定義）とユースケース（具象処理）のGoファイルをフラットに配置する。
   - `adapter/` ディレクトリは作成せず、`repository` は `internal/repository` 直下に配置する。
   - `controller/` ディレクトリは作成せず、Wailsが直接バインドするルートの `app.go`（Wails App）がコントローラを兼ね、直接 `usecase` のロジックを呼び出す。
3. **DTO（データ転送オブジェクト）の配置**
   - フロントエンドとの通信等で用いるDTO（React Flow構造に合わせた型定義など）は、`usecase` ではなく `domain/dto` に配置する。

## 理由

### 採用理由

1. **Wails（外部フレームワーク）との境界分離**
   - 画像生成を行う `exporter` を `utils/exporter` に分離し、OSダイアログ呼び出しやファイル出力といったフレームワーク/OS依存の処理は `app.go` に留める。`exporter` 自体は `[]byte` を返す純粋なGoグラフィックスロジックとすることで、Wails環境なしでも完全なユニットテストを可能にする。
2. **適度なフラット化による開発効率の向上**
   - クリーンアーキテクチャを厳格に適用するとパッケージやファイルが過度に細分化されボイラープレートコードが増大する。インターフェースとユースケースを `usecase` パッケージ直下に並べ、`adapter` フォルダを排すことで、見通しが良く実装しやすいフラット構造を実現する。
3. **依存関係の単一方向化**
   - 依存方向が `app.go (Wails)` → `usecase` → `domain` となるようにし、下位レイヤーが上位の技術詳細（ファイルシステム、描画ライブラリ、Wails SDK）に依存しない構造を担保する。

### 検討した代替案

| 案 | 内容 | 採用見送りの理由 |
|------|------|------------------|
| **案A** | Wails標準のフラット構造（`app.go`にパース、キャッシュ、画像生成などのすべての処理を集約） | Wails SDKとの結合度が極めて高くなり、UIなしでのテスト実行やCLI化といった再利用が困難になるため。 |
| **案B** | 厳格なクリーンアーキテクチャ（`internal/adapter/repository`, `internal/usecase/interactor`, `internal/usecase/port` などのディレクトリ構築） | 本アプリケーションの規模に対してディレクトリ構造が複雑かつ深くなりすぎ、コードの移動やファイル追加のコストが高くなるため。 |

## 結果

### ディレクトリ構造
```
/ (root)
├── main.go                      (Wails起動・DI設定)
├── app.go                       (Wails App定義。コントローラとしてusecaseを直接呼び出し)
├── frontend/                    (Vite + React フロントエンド)
└── internal/
    ├── domain/                  (Domain Entities & DTO)
    │   ├── model/               (Session, Turn などのコアドメイン型)
    │   └── dto/                 (SessionSummary, SessionDetailResponse などの通信用DTO)
    │
    ├── usecase/                 (ユースケース。インターフェースと処理ロジックを直下に並べる)
    │   ├── session_repo.go      (Port: セッションログの読み込み用 interface)
    │   ├── cache_repo.go        (Port: キャッシュの読み書き用 interface)
    │   ├── exporter.go          (Port: 画像エクスポート用 interface)
    │   ├── list_sessions.go     (Interactor: セッション一覧取得ロジック)
    │   ├── get_session_detail.go(Interactor: セッション詳細取得・キャッシュ制御ロジック)
    │   └── export_stats.go      (Interactor: 統計画像エクスポートロジック)
    │
    ├── repository/              (Portの実装: ファイルシステム操作)
    │   ├── session_fs.go        (~/.codex/sessions のスキャン・JSONLパース)
    │   └── cache_fs.go          (~/.codex-display のキャッシュ保存)
    │
    └── utils/                   (共通ユーティリティ)
        └── exporter/            (Portの実装: 画像生成エンジン)
            └── png_exporter.go  (PNG画像バイト生成)
```

- パッケージ間の結合度が低減され、バックエンドの各機能はGoの標準 `go test` で完全にテスト可能となる。
- Wails以外のUIやCLIを作成したくなった場合でも、`internal/usecase` や `internal/domain` をそのまま再利用できる。

## 関連決定

なし
