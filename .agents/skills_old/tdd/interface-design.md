# テスト可能性のためのインターフェース設計

良いインターフェースはテストを自然にする：

1. **依存関係を作成するのではなく、受け取る**

   ```typescript
   // テスト可能
   function processOrder(order, paymentGateway) {}

   // テスト困難
   function processOrder(order) {
     const gateway = new StripeGateway();
   }
   ```

2. **副作用を発生させるのではなく、結果を返す**

   ```typescript
   // テスト可能
   function calculateDiscount(cart): Discount {}

   // テスト困難
   function applyDiscount(cart): void {
     cart.total -= discount;
   }
   ```

3. **小さな表面積**
   - メソッドが少ない = 必要なテストが少ない
   - パラメータが少ない = シンプルなテストセットアップ
