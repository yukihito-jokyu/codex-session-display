# ADR 0017: 開発時におけるViteプロキシ・CORS・WebSocketの設計決定

## ステータス

承認済み

## コンテキスト

Wails (Go + React) を用いたデスクトップアプリケーションの開発において、開発時デブサーバー（Vite）とバックエンド（Go）間の通信設計、CORSの要否、およびViteのHMR（Hot Module Replacement）で使用するWebSocketとWailsの通信の共存方法を決定する必要がある。

## 決定

1. **APIプロキシの不採用**:
   - `vite.config.ts` の `server.proxy` 設定は行わない。
   - 本アプリではフロントエンドとGoバックエンド間の通信に Wails のバインディング（IPC通信）のみを使用するため、フロントエンドからGoバックエンドへの直接のHTTP APIリクエスト（`/api/...` 等）は発生しないため。

2. **完全ローカル完結によるCORS/CSP対策の簡素化**:
   - アプリケーションで使用するすべてのリソース（外部フォント、CSS、各種アセット）をフロントエンドプロジェクト内にローカル配置する。
   - これにより、外部ネットワークへのリクエスト自体が発生しないため、Go側での開発用CORSヘッダー付与やViteプロキシによる回避策、および複雑なCSP（Content Security Policy）設定は不要とする。

3. **HMR用WebSocketとWailsの共存担保**:
   - `vite.config.ts` において、HMR用の接続設定を明示的にローカルに固定する。
   - Wailsのデバッグ通信はWebviewネイティブのIPCチャンネルを使用し、ViteのHMRは指定されたポートのWebSocket（`ws://localhost:5173`）を使用するため、両者はネットワーク競合や干渉を起こさずに共存する。

### `vite.config.ts` の設定定義

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // Wailsが自動生成するGoバインディングへのショートカットエイリアス
      '@wailsjs': path.resolve(__dirname, './wailsjs'),
      // ソースコードへのショートカットエイリアス
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,      // 開発サーバーのポートを5173に固定
    strictPort: true, // ポート競合時に自動で他のポートを使用するのを防ぎ、Wailsの接続ポートとのズレを防止
    hmr: {
      protocol: 'ws',
      host: 'localhost',
      port: 5173,    // HMRのWebSocket接続を開発サーバーと同一ポートに固定
    },
  },
})
```

## 理由

1. **シンプルなアーキテクチャの維持**:
   - 開発時のAPIプロキシを排除することで、設定ファイルの記述を最小限に抑え、不要な通信中継による設定バグや通信レイテンシの排除を実現する。
2. **環境依存・セキュリティリスクの排除**:
   - 完全ローカル完結にすることで、オフライン環境での動作を完全に保証し、外部オリジン接続に対する不要なセキュリティホールや、CORSエラーに起因する開発速度の低下を防ぐ。
3. **ホットリロード（HMR）の動作信頼性**:
   - Wails Webviewはネイティブブラウザと異なり、ホスト名やポートの解決が自動で行われない場合がある。`strictPort: true` と `server.hmr` の明記により、Webview内からでもHMR用のWebSocketが確実に動作し、快適な開発者体験（DX）を維持できる。

## 検討した代替案

| 候補 | 採用見送りの理由 |
| :--- | :--- |
| **ViteでのAPIプロキシ (`server.proxy`) + ローカルGo HTTPサーバー** | モック開発などには有用だが、本アプリはWailsの bindings（IPC）で機能が完結するため、Goで個別のHTTPサーバーを維持するコストや開発時の複雑さが見合わないと判断したため。 |
| **外部フォント/ライブラリのCDN読み込み + CSP緩和** | アプリケーションがインターネット環境に依存することになり、要件である「オフライン環境での動作」を満たせなくなる。またCSPの制限を緩めるセキュリティリスクがあるため。 |
