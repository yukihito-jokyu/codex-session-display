import type {
	FullResult,
	Reporter,
	TestCase,
	TestError,
	TestResult,
} from "@playwright/test/reporter";

function formatDuration(durationMs: number): string {
	if (durationMs < 1000) {
		return `${durationMs}ms`;
	}
	return `${(durationMs / 1000).toFixed(1)}s`;
}

function formatChunks(chunks: Array<string | Buffer>): string {
	return chunks.map((chunk) => chunk.toString()).join("");
}

function formatError(error: TestError): string {
	const current =
		error.stack ?? error.message ?? error.value ?? "Unknown error";
	if (!error.cause) {
		return current;
	}
	return `${current}\nCaused by:\n${formatError(error.cause)}`;
}

function formatFailure(test: TestCase): string {
	const result = test.results.at(-1);
	if (!result) {
		return `FAILED: ${test.titlePath().filter(Boolean).join(" > ")}`;
	}

	const sections = [
		`FAILED: ${test.titlePath().filter(Boolean).join(" > ")}`,
		...result.errors.map(
			(error, index) => `ERROR ${index + 1}:\n${formatError(error)}`,
		),
	];
	const stdout = formatChunks(result.stdout);
	const stderr = formatChunks(result.stderr);

	if (stdout) {
		sections.push(`STDOUT:\n${stdout}`);
	}
	if (stderr) {
		sections.push(`STDERR:\n${stderr}`);
	}
	if (result.attachments.length > 0) {
		sections.push(
			[
				"ATTACHMENTS:",
				...result.attachments.map((attachment) => {
					const location = attachment.path ?? "(inline attachment)";
					return `- ${attachment.name} [${attachment.contentType}]: ${location}`;
				}),
			].join("\n"),
		);
	}

	return sections.join("\n\n");
}

class CompactReporter implements Reporter {
	private readonly globalErrors: TestError[] = [];
	private readonly tests = new Map<string, TestCase>();
	private startedAt = 0;

	printsToStdio(): boolean {
		return true;
	}

	onBegin(): void {
		this.startedAt = Date.now();
	}

	onTestEnd(test: TestCase, _result: TestResult): void {
		this.tests.set(test.id, test);
	}

	onError(error: TestError): void {
		this.globalErrors.push(error);
	}

	onEnd(result: FullResult): void {
		let passed = 0;
		let flaky = 0;
		let skipped = 0;
		let failed = 0;

		for (const test of this.tests.values()) {
			switch (test.outcome()) {
				case "expected":
					passed += 1;
					break;
				case "flaky":
					flaky += 1;
					break;
				case "skipped":
					skipped += 1;
					break;
				case "unexpected":
					failed += 1;
					break;
			}
		}

		const failures = [...this.tests.values()].filter(
			(test) => test.outcome() === "unexpected",
		);
		for (const failure of failures) {
			console.log(formatFailure(failure));
		}
		for (const [index, error] of this.globalErrors.entries()) {
			console.log(`RUN ERROR ${index + 1}:\n${formatError(error)}`);
		}
		if (
			this.globalErrors.length === 0 &&
			(result.status === "timedout" || result.status === "interrupted")
		) {
			console.log(`RUN STATUS: ${result.status}`);
		}

		const duration = formatDuration(Date.now() - this.startedAt);
		console.log(
			`${passed} passed, ${flaky} flaky, ${skipped} skipped, ${failed} failed (${duration})`,
		);
	}
}

export default CompactReporter;
