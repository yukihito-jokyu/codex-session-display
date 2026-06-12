# 良いテストと悪いテスト

## 良いテスト

**インテグレーションスタイル**：内部のモックではなく、実際のインターフェースを通じてテストする。

```typescript
// GOOD: 観察可能な振る舞いをテスト
test("ユーザーは有効なカートでチェックアウトできる", async () => {
  const cart = createCart();
  cart.add(product);
  const result = await checkout(cart, paymentMethod);
  expect(result.status).toBe("confirmed");
});
```

特徴：

- ユーザー/呼び出し元が気にする振る舞いをテスト
- パブリックAPIのみ使用
- 内部リファクタリング後も生き残る
- HOWではなくWHATを記述
- テスト1つにつき1つの論理的アサーション

## 悪いテスト

**実装詳細テスト**：内部構造に結合している。

```typescript
// BAD: 実装の詳細をテスト
test("checkoutはpaymentService.processを呼び出す", async () => {
  const mockPayment = jest.mock(paymentService);
  await checkout(cart, payment);
  expect(mockPayment.process).toHaveBeenCalledWith(cart.total);
});
```

危険信号：

- 内部の共同作業者をモックしている
- プライベートメソッドをテストしている
- 呼び出し回数/順序をアサートしている
- 振る舞いが変わっていないのにリファクタリングでテストが壊れる
- テスト名がWHATではなくHOWを記述
- インターフェースの代わりに外部手段で検証している

```typescript
// BAD: インターフェースをバイパスして検証
test("createUserはデータベースに保存する", async () => {
  await createUser({ name: "Alice" });
  const row = await db.query("SELECT * FROM users WHERE name = ?", ["Alice"]);
  expect(row).toBeDefined();
});

// GOOD: インターフェースを通じて検証
test("createUserによりユーザーが取得可能になる", async () => {
  const user = await createUser({ name: "Alice" });
  const retrieved = await getUser(user.id);
  expect(retrieved.name).toBe("Alice");
});
```
