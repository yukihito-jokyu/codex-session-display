import { useCallback, useEffect, useState } from "react";
import { GetSessionDetail } from "wailsjs/go/main/App";
import type { dto } from "wailsjs/go/models";

export function useSessionDetail(id: string | undefined) {
	const [sessionData, setSessionData] =
		useState<dto.SessionDetailResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [selectedNode, setSelectedNode] = useState<dto.FlowNode | null>(null);

	const fetchSessionDetail = useCallback((targetId: string) => {
		setLoading(true);
		setError(null);
		GetSessionDetail(targetId)
			.then((data) => {
				setSessionData(data);
				setLoading(false);
			})
			.catch((err) => {
				console.error(err);
				setError(`Failed to fetch session detail: ${err.message || err}`);
				setLoading(false);
			});
	}, []);

	useEffect(() => {
		if (!id) return;
		fetchSessionDetail(id);
	}, [id, fetchSessionDetail]);

	const handleNodeSelect = useCallback((node: dto.FlowNode | null) => {
		setSelectedNode(node);
	}, []);

	const retry = useCallback(() => {
		if (id) {
			fetchSessionDetail(id);
		}
	}, [id, fetchSessionDetail]);

	return {
		sessionData,
		loading,
		error,
		selectedNode,
		handleNodeSelect,
		retry,
	};
}
