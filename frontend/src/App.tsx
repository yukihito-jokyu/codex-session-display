import { useEffect } from "react";
import { HashRouter, Route, Routes, useNavigate } from "react-router-dom";
import { FrontendReady, ResolveSessionIDFromPath } from "wailsjs/go/main/App";
import { EventsOn } from "wailsjs/runtime/runtime";
import styles from "./App.module.css";
import { TitleBar } from "./components/ui/TitleBar/TitleBar";
import { SessionDetailPage } from "./features/session-detail/SessionDetailPage";
import { SessionListPage } from "./features/session-list/SessionListPage";

function AppRoutes() {
	const navigate = useNavigate();

	useEffect(() => {
		const unsubscribe = EventsOn(
			"open-session-file",
			async (filePath: string) => {
				try {
					const sessionID = await ResolveSessionIDFromPath(filePath);
					navigate(`/sessions/${sessionID}`);
				} catch (error) {
					console.error("Failed to open session file", error);
				}
			},
		);

		void FrontendReady().catch((error) => {
			console.error("Failed to notify frontend readiness", error);
		});

		return unsubscribe;
	}, [navigate]);

	return (
		<div className={styles.appContainer}>
			<TitleBar />
			<main className={styles.mainContent}>
				<Routes>
					<Route path="/" element={<SessionListPage />} />
					<Route path="/sessions" element={<SessionDetailPage />} />
					<Route path="/sessions/:id" element={<SessionDetailPage />} />
				</Routes>
			</main>
		</div>
	);
}

function App() {
	return (
		<HashRouter>
			<AppRoutes />
		</HashRouter>
	);
}

export default App;
