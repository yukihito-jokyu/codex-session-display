import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

const frontendDir = fileURLToPath(new URL("../../", import.meta.url));

function runReporterFixture(name: string): {
	exitCode: number;
	output: string;
} {
	const env: NodeJS.ProcessEnv = { ...process.env, CI: "" };
	delete env.FORCE_COLOR;
	delete env.NO_COLOR;

	try {
		const output = execFileSync(
			process.execPath,
			[
				"./node_modules/@playwright/test/cli.js",
				"test",
				"--config",
				"tests/reporter-fixtures/playwright.config.ts",
				`${name}.spec.ts`,
			],
			{
				cwd: frontendDir,
				encoding: "utf8",
				env,
				stdio: ["ignore", "pipe", "pipe"],
			},
		);

		return { exitCode: 0, output };
	} catch (error) {
		if (
			typeof error === "object" &&
			error !== null &&
			"status" in error &&
			"stdout" in error &&
			"stderr" in error
		) {
			const stdout = String(error.stdout);
			const stderr = String(error.stderr);
			return {
				exitCode: typeof error.status === "number" ? error.status : 1,
				output: `${stdout}${stderr}`,
			};
		}
		throw error;
	}
}

test("成功時はテスト単位のログを出さず1行のサマリーを表示する", () => {
	const result = runReporterFixture("success");

	expect(result.exitCode).toBe(0);
	expect(result.output.trim()).toMatch(
		/^1 passed, 0 flaky, 0 skipped, 0 failed \(\d+(?:\.\d+)?(?:ms|s)\)$/,
	);
	expect(result.output).not.toContain("成功fixture");
});

test("skip時は詳細を出さずskip件数を1行のサマリーへ反映する", () => {
	const result = runReporterFixture("skipped");

	expect(result.exitCode).toBe(0);
	expect(result.output.trim()).toMatch(
		/^0 passed, 0 flaky, 1 skipped, 0 failed \(\d+(?:\.\d+)?(?:ms|s)\)$/,
	);
	expect(result.output).not.toContain("skip fixture");
});

test("flaky時は途中の失敗詳細を出さずflaky件数へ反映する", () => {
	const result = runReporterFixture("flaky");

	expect(result.exitCode).toBe(0);
	expect(result.output.trim()).toMatch(
		/^0 passed, 1 flaky, 0 skipped, 0 failed \(\d+(?:\.\d+)?(?:ms|s)\)$/,
	);
	expect(result.output).not.toContain("flaky fixture");
	expect(result.output).not.toContain("初回リトライ前の出力");
	expect(result.output).not.toContain("初回だけ失敗");
});

test("最終失敗時は原因調査に必要な情報と成果物パスを表示する", () => {
	const result = runReporterFixture("failed");

	expect(result.exitCode).not.toBe(0);
	expect(result.output).toContain("failed fixture");
	expect(result.output).toContain("意図した最終失敗");
	expect(result.output).toContain("failed.spec.ts");
	expect(result.output).toContain("標準出力マーカー");
	expect(result.output).toContain("標準エラーマーカー");
	expect(result.output).toContain("fixture-artifact");
	expect(result.output).toMatch(/artifact\.txt|attachments[/\\]/);
	expect(result.output).not.toContain("展開してはいけない成果物本文");
	expect(result.output.trim()).toMatch(
		/0 passed, 0 flaky, 0 skipped, 1 failed \(\d+(?:\.\d+)?(?:ms|s)\)$/,
	);
});

test("最終失敗したテストのブラウザconsoleとpage errorだけを表示する", () => {
	const result = runReporterFixture("failed-browser");

	expect(result.exitCode).not.toBe(0);
	expect(result.output).toContain(
		"[BROWSER CONSOLE] error: ブラウザconsoleマーカー",
	);
	expect(result.output).toContain(
		"[BROWSER ERROR] Error: ブラウザpage errorマーカー",
	);
});

test("テスト実行基盤の異常を表示して非0で終了する", () => {
	const result = runReporterFixture("infrastructure-error");

	expect(result.exitCode).not.toBe(0);
	expect(result.output).toContain("存在しない実行基盤モジュール");
	expect(result.output.trim()).toMatch(
		/0 passed, 0 flaky, 0 skipped, 0 failed \(\d+(?:\.\d+)?(?:ms|s)\)$/,
	);
});
