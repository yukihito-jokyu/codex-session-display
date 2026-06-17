import { useCallback, useEffect, useState } from "react";
import { ApplyUpdate, CheckUpdate } from "../../../../wailsjs/go/main/App";
import type { dto } from "../../../../wailsjs/go/models";
import { EventsOn } from "../../../../wailsjs/runtime/runtime";

export interface UpdateProgress {
	status: "idle" | "downloading" | "extracting" | "restarting" | "error";
	progress: number;
	error?: string;
}

export function useUpdate() {
	const [updateResult, setUpdateResult] = useState<dto.UpdateResult | null>(
		null,
	);
	const [checking, setChecking] = useState(false);
	const [progress, setProgress] = useState<UpdateProgress>({
		status: "idle",
		progress: 0,
	});

	const check = useCallback(async () => {
		setChecking(true);
		try {
			const res = await CheckUpdate();
			setUpdateResult(res);
		} catch (err) {
			console.error("Failed to check for updates", err);
		} finally {
			setChecking(false);
		}
	}, []);

	const apply = useCallback(async () => {
		if (!updateResult?.downloadUrl) return;

		setProgress({ status: "downloading", progress: 0 });

		const unsubscribe = EventsOn("update-progress", (data: unknown) => {
			console.log("[useUpdate] Received progress event:", data);
			if (data && typeof data === "object") {
				const progressData = data as { status?: string; progress?: number };
				let status:
					| "idle"
					| "downloading"
					| "extracting"
					| "restarting"
					| "error" = "idle";
				const rawStatus = progressData.status;

				if (rawStatus === "downloading" || rawStatus === "download_complete") {
					status = "downloading";
				} else if (rawStatus === "extracting") {
					status = "extracting";
				} else if (rawStatus === "restarting") {
					status = "restarting";
				} else if (rawStatus === "error") {
					status = "error";
				}

				setProgress({
					status,
					progress:
						typeof progressData.progress === "number"
							? progressData.progress
							: 0,
				});
			}
		});

		try {
			await ApplyUpdate(updateResult.downloadUrl);
		} catch (err) {
			console.error("Failed to apply update", err);
			const errorMessage = err instanceof Error ? err.message : String(err);
			setProgress({
				status: "error",
				progress: 0,
				error: errorMessage,
			});
			unsubscribe();
		}

		return () => {
			unsubscribe();
		};
	}, [updateResult]);

	useEffect(() => {
		check();
	}, [check]);

	return {
		updateResult,
		checking,
		progress,
		apply,
		check,
	};
}
