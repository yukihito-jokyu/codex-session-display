import { HashRouter, Route, Routes } from "react-router-dom";
import styles from "./App.module.css";
import { TitleBar } from "./components/ui/TitleBar/TitleBar";
import { SessionDetailPage } from "./features/session-detail/SessionDetailPage";
import { SessionListPage } from "./features/session-list/SessionListPage";

function App() {
	return (
		<HashRouter>
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
		</HashRouter>
	);
}

export default App;
