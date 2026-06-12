import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

test("failed fixture", async ({ browserName: _browserName }, testInfo) => {
	console.log("標準出力マーカー");
	console.error("標準エラーマーカー");
	await testInfo.attach("fixture-artifact", {
		path: fileURLToPath(new URL("./artifact.txt", import.meta.url)),
		contentType: "text/plain",
	});

	expect("actual", "意図した最終失敗").toBe("expected");
});
