import styles from "./CorpusAnalyzePage.module.css";
import { useCorpusAnalyze } from "./hooks/useCorpusAnalyze";

export function CorpusAnalyzePage() {
	const { loading, error, result, projectSource, setProjectSource, analyze } =
		useCorpusAnalyze();

	return (
		<div className={styles.container}>
			<div className={styles.toolbar}>
				<div className={styles.toolbarLeft}>
					<span className={styles.sourceLabel}>プロジェクトソース:</span>
					<select
						value={projectSource}
						onChange={(e) =>
							setProjectSource(e.target.value as "config" | "home")
						}
						className={styles.select}
						disabled={loading}
					>
						<option value="home">~/.claude/projects</option>
						<option value="config">CLAUDE_CONFIG_DIR/projects</option>
					</select>
				</div>
				<button
					type="button"
					onClick={analyze}
					className={styles.btnAnalyze}
					disabled={loading}
				>
					{loading ? "分析中..." : "履歴分析を実行"}
				</button>
			</div>

			{error && (
				<div className={styles.error} role="alert">
					<span>⚠️</span>
					<span>{error}</span>
				</div>
			)}

			{loading && (
				<div className={styles.loader}>
					<div className={styles.spinner} />
					<span>
						Claude Code のセッション履歴をストリーミング分析しています...
					</span>
				</div>
			)}

			{!loading && result && (
				<>
					<div className={styles.summaryGrid}>
						<div className={styles.summaryCard}>
							<span className={styles.summaryLabel}>
								総セッションファイル数
							</span>
							<span
								className={`${styles.summaryValue} ${styles.summaryValueHighlight}`}
							>
								{result.totalFiles}
							</span>
						</div>
						<div className={styles.summaryCard}>
							<span className={styles.summaryLabel}>
								総トランスクリプト行数
							</span>
							<span className={styles.summaryValue}>
								{result.totalLines.toLocaleString()}
							</span>
						</div>
						<div className={styles.summaryCard}>
							<span className={styles.summaryLabel}>パースエラー発生数</span>
							<span
								className={`${styles.summaryValue} ${result.parseErrors.length > 0 ? styles.summaryValueAlert : ""}`}
							>
								{result.parseErrors.reduce((sum, e) => sum + e.count, 0)}
							</span>
						</div>
					</div>

					<div className={styles.section}>
						<h2 className={styles.sectionTitle}>
							📂 コーパススキーマ表 (Field Paths & Types)
						</h2>
						<div className={styles.tableWrapper}>
							<table className={styles.table}>
								<thead>
									<tr>
										<th>フィールドパス (Field Path)</th>
										<th>検出された型と頻度 (Type Distribution)</th>
									</tr>
								</thead>
								<tbody>
									{Object.entries(result.fieldPaths).map(([path, types]) => (
										<tr key={path}>
											<td className={styles.codePath}>{path}</td>
											<td>
												{Object.entries(types).map(([type, count]) => (
													<span key={type} style={{ marginRight: "12px" }}>
														<code
															style={{
																background: "var(--mono-bg)",
																padding: "2px 6px",
																borderRadius: "4px",
																marginRight: "4px",
															}}
														>
															{type}
														</code>
														({count.toLocaleString()})
													</span>
												))}
											</td>
										</tr>
									))}
									{Object.keys(result.fieldPaths).length === 0 && (
										<tr>
											<td colSpan={2} className={styles.emptyText}>
												スキーマが検出されませんでした。
											</td>
										</tr>
									)}
								</tbody>
							</table>
						</div>
					</div>

					<div className={styles.subGrid}>
						<div className={styles.section}>
							<h3 className={styles.sectionTitle}>
								📊 イベント出現頻度 (Content Types)
							</h3>
							<div className={styles.tableWrapper}>
								<table className={styles.table}>
									<thead>
										<tr>
											<th>タイプ (Content Type)</th>
											<th>出現回数 (Count)</th>
										</tr>
									</thead>
									<tbody>
										{Object.entries(result.contentTypes).map(
											([type, count]) => (
												<tr key={type}>
													<td>
														<strong>{type}</strong>
													</td>
													<td>{count.toLocaleString()}</td>
												</tr>
											),
										)}
										{Object.keys(result.contentTypes).length === 0 && (
											<tr>
												<td colSpan={2} className={styles.emptyText}>
													データがありません。
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>

						<div className={styles.section}>
							<h3 className={styles.sectionTitle}>
								🛠️ ツール使用頻度 (Tool Calls)
							</h3>
							<div className={styles.tableWrapper}>
								<table className={styles.table}>
									<thead>
										<tr>
											<th>ツール名 (Tool Name)</th>
											<th>実行回数 (Count)</th>
										</tr>
									</thead>
									<tbody>
										{Object.entries(result.toolNames).map(([name, count]) => (
											<tr key={name}>
												<td>
													<code className={styles.codePath}>{name}</code>
												</td>
												<td>{count.toLocaleString()}</td>
											</tr>
										))}
										{Object.keys(result.toolNames).length === 0 && (
											<tr>
												<td colSpan={2} className={styles.emptyText}>
													ツールは使用されていません。
												</td>
											</tr>
										)}
									</tbody>
								</table>
							</div>
						</div>
					</div>

					<div className={styles.section}>
						<h2 className={styles.sectionTitle}>
							🔒 プライバシー保護確認 (Privacy Protection Metrics)
						</h2>
						<div className={styles.privacyAlert}>
							<span className={styles.privacyAlertIcon}>🛡️</span>
							<div>
								<strong>匿名性と安全性の保証:</strong>{" "}
								このレポートには、実際のプロジェクトパス、コマンド全文、テキスト本文、およびツール出力は一切含まれていません。これらはすべて長さバケット、または不可逆な
								SHA-256 ハッシュ値に置き換えられ、外部への情報漏洩を防ぎます。
							</div>
						</div>

						<div className={styles.subGrid}>
							<div>
								<h4 style={{ marginBottom: "8px" }}>
									👤 ユーザーメッセージ長分布
								</h4>
								<div className={styles.tableWrapper}>
									<table className={styles.table}>
										<thead>
											<tr>
												<th>文字数バケット</th>
												<th>出現回数</th>
											</tr>
										</thead>
										<tbody>
											{Object.entries(result.privacyMetrics.textLengthDist).map(
												([bucket, count]) => (
													<tr key={bucket}>
														<td>{bucket} 文字</td>
														<td>{count.toLocaleString()}</td>
													</tr>
												),
											)}
										</tbody>
									</table>
								</div>
							</div>

							<div>
								<h4 style={{ marginBottom: "8px" }}>
									🧠 思考プロンプト(thinking)長分布
								</h4>
								<div className={styles.tableWrapper}>
									<table className={styles.table}>
										<thead>
											<tr>
												<th>文字数バケット</th>
												<th>出現回数</th>
											</tr>
										</thead>
										<tbody>
											{Object.entries(
												result.privacyMetrics.thinkingLengthDist,
											).map(([bucket, count]) => (
												<tr key={bucket}>
													<td>{bucket} 文字</td>
													<td>{count.toLocaleString()}</td>
												</tr>
											))}
										</tbody>
									</table>
								</div>
							</div>
						</div>

						<div className={styles.subGrid} style={{ marginTop: "12px" }}>
							<div>
								<h4 style={{ marginBottom: "8px" }}>
									💻 コマンド実行ハッシュ (不可逆)
								</h4>
								<div className={styles.tableWrapper}>
									<table className={styles.table}>
										<thead>
											<tr>
												<th>コマンドハッシュ (SHA-256)</th>
												<th>実行頻度</th>
											</tr>
										</thead>
										<tbody>
											{Object.entries(
												result.privacyMetrics.commandHashDist,
											).map(([hash, count]) => (
												<tr key={hash}>
													<td
														className={styles.codePath}
														style={{ fontSize: "12px" }}
													>
														{hash}
													</td>
													<td>{count.toLocaleString()}</td>
												</tr>
											))}
											{Object.keys(result.privacyMetrics.commandHashDist)
												.length === 0 && (
												<tr>
													<td colSpan={2} className={styles.emptyText}>
														データがありません。
													</td>
												</tr>
											)}
										</tbody>
									</table>
								</div>
							</div>

							<div>
								<h4 style={{ marginBottom: "8px" }}>
									📦 ツール実行出力ハッシュ (不可逆)
								</h4>
								<div className={styles.tableWrapper}>
									<table className={styles.table}>
										<thead>
											<tr>
												<th>出力ハッシュ (SHA-256)</th>
												<th>出現頻度</th>
											</tr>
										</thead>
										<tbody>
											{Object.entries(result.privacyMetrics.toolOutputDist).map(
												([hash, count]) => (
													<tr key={hash}>
														<td
															className={styles.codePath}
															style={{ fontSize: "12px" }}
														>
															{hash}
														</td>
														<td>{count.toLocaleString()}</td>
													</tr>
												),
											)}
											{Object.keys(result.privacyMetrics.toolOutputDist)
												.length === 0 && (
												<tr>
													<td colSpan={2} className={styles.emptyText}>
														データがありません。
													</td>
												</tr>
											)}
										</tbody>
									</table>
								</div>
							</div>
						</div>

						{result.parseErrors.length > 0 && (
							<div style={{ marginTop: "12px" }}>
								<h4
									style={{
										marginBottom: "8px",
										color: "var(--node-warning-text)",
									}}
								>
									⚠️ パースエラー検出ファイル
								</h4>
								<div className={styles.tableWrapper}>
									<table className={styles.table}>
										<thead>
											<tr>
												<th>ファイル識別子 (匿名ID)</th>
												<th>エラー発生行数</th>
											</tr>
										</thead>
										<tbody>
											{result.parseErrors.map((err) => (
												<tr key={err.fileId}>
													<td className={styles.codePath}>{err.fileId}</td>
													<td>{err.count.toLocaleString()}</td>
												</tr>
											))}
										</tbody>
									</table>
								</div>
							</div>
						)}
					</div>
				</>
			)}
		</div>
	);
}
