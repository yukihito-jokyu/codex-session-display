import { expect, test } from "@playwright/test";

test.describe.configure({ retries: 1 });

test("flaky fixture", ({ browserName: _browserName }, testInfo) => {
	if (testInfo.retry === 0) {
		console.log("初回リトライ前の出力");
	}
	expect(testInfo.retry, "初回だけ失敗").toBe(1);
});
