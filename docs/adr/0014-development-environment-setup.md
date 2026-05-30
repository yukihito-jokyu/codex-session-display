# ADR 0014: 開発環境セットアップ手順の決定

## ステータス

承認済み

## コンテキスト

デスクトップアプリケーションの開発において、開発環境のセットアップ手順、要求されるツールチェーンのバージョン、タスクランナー、およびホットリロードの仕組みを決定する必要がある。

## 決定

以下の通り開発環境の構成を決定する。

1. **要求ツールチェーンバージョン**:
   - Go: `1.26` 以上
   - Node.js: `24` 以上 (LTS)

2. **タスクランナーとして Taskfile の採用**:
   - 従来の Makefile に代わり、Goフレンドリーで記述が簡潔な `Taskfile.yml` (go-task) を採用する。
   - 実装時に作成する `Taskfile.yml` の定義案は以下の通りとする。

```yaml
version: '3'

tasks:
  setup:
    desc: Setup development dependencies for backend and frontend
    cmds:
      - go install github.com/wailsapp/wails/v2/cmd/wails@latest
      - go mod download
      - cd frontend && npm install

  dev:
    desc: Start development server with hot-reload
    cmds:
      - wails dev

  build:
    desc: Build production binary
    cmds:
      - wails build

  test:
    desc: Run tests for backend
    cmds:
      - go test ./...

  lint:
    desc: Run linters for backend
    cmds:
      - golangci-lint run

  clean:
    desc: Clean build artifacts
    cmds:
      - rm -rf build/
```

3. **ホットリロード (hot-reload) の仕組み**:
   - `wails dev` コマンドにより起動する。
   - バックエンド (Go) は Wails CLI がファイル変更を検知して自動的に再ビルドおよびアプリを再起動する。
   - フロントエンド (Vite+React) は Vite の HMR (Hot Module Replacement) によってリロードなしで画面が即座に更新される。これら両方のホットリロードが連動する。
