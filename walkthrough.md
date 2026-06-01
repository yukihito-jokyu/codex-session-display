# Walkthrough

- `internal/usecase/get_session_detail_test.go` に、`token_count` の直前レコードがノード未生成でも、さらに前のマップ済みノードへフォールバックして解決されることを検証するテストを追加
- `BoundToNodeID` が `node-3` になることと、同ノードにトークンバッジが集約されることを確認
- 検証コマンド:
- `go test ./internal/usecase -run TestGetSessionDetailUseCase_Execute`
- `go test -tags production ./internal/usecase -run TestGetSessionDetailUseCase_Execute`
- `go test ./...`
- `go test -tags production ./...`

`frontend/src/components/ui/FlowCanvas/FlowCanvas.tsx` の選択中インタラクション制御を見直し、CSS による `pointer-events` の遮断と重複した pane click ハンドリングをやめて、React Flow の `onPaneClick` に寄せました。これにより、ノード選択後の詳細パネル表示、閉じる、再選択、余白クリックによる解除の経路が安定します。

`frontend` 配下で `pnpm run lint -- src/components/ui/FlowCanvas/FlowCanvas.tsx src/components/ui/FlowCanvas/FlowCanvas.module.css` を実行し、静的解析は通過しています。加えて `pnpm exec playwright test tests/e2e/session-detail.spec.ts --grep "ノードを選択した際に画面下部に詳細パネルが表示され、閉じる動作が機能すること"` を再実行し、対象 1 ケースのパスを確認しました。

`frontend/src/components/ui/FlowCanvas/FlowCanvas.tsx` と `frontend/src/components/ui/FlowCanvas/FlowCanvas.module.css` を追加で調整し、選択中ノードだけは内部操作を許可しつつ、それ以外のキャンバスクリックでは詳細パネルを閉じるようにしました。これにより `ContextDocNode` の展開・折りたたみと、pane クリックによる選択解除の両方が両立します。

追加検証:
- `VITE_COVERAGE=true npx playwright test tests/e2e/session-detail.spec.ts --grep "ContextDocNode|ノードを選択した際に画面下部に詳細パネル"`
- `task test:e2e:cover`

最終結果:
- `task test:e2e:cover`: 55 passed / 46.4s
