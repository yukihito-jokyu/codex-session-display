import { useState } from "react";
import { AnalyzeClaudeCorpus } from "wailsjs/go/main/App";

export interface ParseErrorInfo {
	fileId: string;
	count: number;
}

export interface TypeCount {
	[type: string]: number;
}

export interface PrivacyMetrics {
	textLengthDist: { [bucket: string]: number };
	thinkingLengthDist: { [bucket: string]: number };
	commandHashDist: { [hash: string]: number };
	toolOutputDist: { [hash: string]: number };
}

export interface AnalyzeResult {
	totalFiles: number;
	totalLines: number;
	parseErrors: ParseErrorInfo[];
	fieldPaths: { [path: string]: TypeCount };
	contentTypes: { [type: string]: number };
	toolNames: { [name: string]: number };
	usageKeys: { [key: string]: number };
	privacyMetrics: PrivacyMetrics;
}

export function useCorpusAnalyze() {
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [result, setResult] = useState<AnalyzeResult | null>(null);
	const [projectSource, setProjectSource] = useState<"config" | "home">("home");

	const analyze = async () => {
		setLoading(true);
		setError(null);
		setResult(null);

		try {
			// AnalyzeOptions と同等のパラメータを渡す
			const res = await AnalyzeClaudeCorpus({ projectSource });
			setResult(res as AnalyzeResult);
		} catch (err: unknown) {
			console.error("Corpus analysis failed", err);
			const errMsg = err instanceof Error ? err.message : String(err);
			setError(errMsg || "分析の実行中にエラーが発生しました");
		} finally {
			setLoading(false);
		}
	};

	return {
		loading,
		error,
		result,
		projectSource,
		setProjectSource,
		analyze,
	};
}
