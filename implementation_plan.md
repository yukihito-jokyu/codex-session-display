# Issue #187 実装計画

## 目的

`Last Token Consumption per Index` に、各 `token_count` 時点の入力コンテキスト使用率を
右軸のパーセント系列として追加する。既存のトークン数系列と共通選択を維持し、
算出不能な点と100%を超える点を正確に扱う。

## 公開インターフェース

`TokenCountEntry` に `model_context_window` を追加する。

- `token_count.info.model_context_window` が正の値なら使用する。
- 欠落または0以下なら、所属ターンの
  `task_started.model_context_window` が正の値の場合に限りフォールバックする。
- どちらも有効でなければ0を返し、フロントエンドでは算出不能として扱う。

`last_token_usage.input_tokens` はキャッシュ済み入力を含む記録値をそのまま使用し、
`cached_input_tokens` を差し引かない。使用率はフロントエンドで
`input_tokens / model_context_window * 100` として導出し、入力または分母が0以下なら
`undefined` とする。

キャッシュスキーマを6へ更新し、スキーマ5以前のキャッシュは再解析する。
Wails生成型は生成コマンドで同期し、手動編集しない。

## 振る舞い

- 既存の Total / Input / Output / Reasoning / Cached は左軸のトークン数系列として維持する。
- `Context Usage (%)` を右軸へ追加し、入力トークン数を有効なコンテキスト上限で除算する。
- `token_count.info.model_context_window` をターンの上限より優先する。
- キャッシュ済み入力トークンを入力トークン数から差し引かない。
- 入力トークン数または有効な上限がない点は描画せず、同インデックスのツールチップでは
  `Context Usage (%): N/A` と表示する。
- 100%を超える使用率を丸めず、右軸の自動範囲で実値を表示する。
- Context Usage の有効な点は既存系列と同じクリック・キーボード操作で共通選択できる。
- Context Usage の算出不能点は選択用の点を表示しない。

## TDDサイクル

1. トレーサー弾: token_count固有の上限を公開する
   - RED: `token_count.info.model_context_window` が
     `TokenCountEntry.model_context_window` に出力されるGoテストを追加する。
   - GREEN: DTOと組み立て処理へ最小実装し、キャッシュスキーマを6へ更新する。
2. ターン上限へのフォールバック
   - RED: token_count側の上限欠落時は所属ターンのtask_started値を使い、
     両方無効なら0になるGoテストを追加する。
   - GREEN: 正の値だけを採用する上限解決を実装する。
3. Context Usage系列
   - RED: 右軸、系列名、期待使用率、算出不能点の非表示とツールチップの`N/A`を
     Playwrightで検証する。
   - GREEN: チャートデータ、右軸、系列、ツールチップを最小実装する。
4. 100%超過と共通選択
   - RED: キャッシュ入力を差し引かない100%超過値と、Context Usage点からの
     共通選択をPlaywrightで検証する。
   - GREEN: 値をクランプせず、既存の`TokenChartDot`選択経路へ接続する。
5. リファクタリングと回帰
   - 使用率計算とツールチップ表示を小さな純粋処理へ整理する。
   - 各リファクタリング後に対象GoテストとPlaywrightテストを再実行する。

## ドキュメント

- `docs/requirements.md`: 分母の優先順位、算出式、算出不能、100%超過、左右軸、
  共通選択を追加する。
- `docs/detailed-design.md`: `TokenCountEntry` 契約、使用率導出、Rechartsの左右軸、
  ツールチップ、キャッシュスキーマ6を追加する。
- `docs/adr/0031-token-count-context-window.md`: token_count単位の上限をDTOへ保持し、
  表示用の使用率をフロントエンドで導出する決定を記録する。

## 検証

- 各RED/GREENで対象GoテストまたはPlaywrightテストを実行する。
- `go test ./...`
- `go test -tags production ./...`
- `golangci-lint run`
- `frontend` で `npm run lint`
- `frontend` で `npm run build`
- `task test:e2e`
- `go-review` と `frontend-review` で差分をレビューする。

## 優先する振る舞い

1. token_count固有値を優先し、所属ターン値へ正しくフォールバックすること
2. 算出不能を0%と混同しないこと
3. キャッシュ入力を差し引かず、100%超過値を維持すること
4. 既存系列とContext Usage系列の共通選択を維持すること
