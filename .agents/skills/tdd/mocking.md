# モックを使うタイミング

**システム境界**でのみモックする：

- 外部API（決済、メールなど）
- データベース（場合により - テスト用DBを優先）
- 時間 / ランダム性
- ファイルシステム（場合により）

モックしないもの：

- 自分のクラス/モジュール
- 内部の共同作業者
- 自分で制御できるもの

## モック容易性のための設計

システム境界では、モックしやすいインターフェースを設計する：

**1. 依存性注入を使用**

外部依存を内部で作成するのではなく、引数として渡す：

```typescript
// モックしやすい
function processPayment(order, paymentClient) {
  return paymentClient.charge(order.total);
}

// モック困難
function processPayment(order) {
  const client = new StripeClient(process.env.STRIPE_KEY);
  return client.charge(order.total);
}
```

**2. 汎用フェッチャーよりSDKスタイルのインターフェースを優先**

条件付きロジックを持つ1つの汎用関数の代わりに、外部操作ごとに固有の関数を作成：

```typescript
// GOOD: 各関数が独立してモック可能
const api = {
  getUser: (id) => fetch(`/users/${id}`),
  getOrders: (userId) => fetch(`/users/${userId}/orders`),
  createOrder: (data) => fetch('/orders', { method: 'POST', body: data }),
};

// BAD: モックに内部で条件付きロジックが必要
const api = {
  fetch: (endpoint, options) => fetch(endpoint, options),
};
```

SDKアプローチのメリット：
- 各モックは1つの特定の形状を返す
- テストセットアップに条件付きロジックが不要
- テストがどのエンドポイントを実行するか見やすい
- エンドポイントごとの型安全性
