import {
	type KeyboardEventHandler,
	type PointerEventHandler,
	useEffect,
	useRef,
	useState,
} from "react";

const TIMELINE_MIN_WIDTH = 320;
const TIMELINE_INITIAL_RATIO = 0.3;
const TIMELINE_MAX_RATIO = 0.5;
const TIMELINE_KEYBOARD_STEP = 16;
const TIMELINE_STORAGE_KEY = "session-detail.timeline-width";

function getViewportWidth() {
	return window.innerWidth;
}

function getTimelineMaxWidth(viewportWidth: number) {
	return Math.floor(viewportWidth * TIMELINE_MAX_RATIO);
}

function clampTimelineWidth(width: number, viewportWidth: number) {
	return Math.min(
		Math.max(width, TIMELINE_MIN_WIDTH),
		getTimelineMaxWidth(viewportWidth),
	);
}

function getInitialTimelineWidth(viewportWidth: number) {
	const storedWidth = Number.parseInt(
		localStorage.getItem(TIMELINE_STORAGE_KEY) ?? "",
		10,
	);
	const initialWidth = Number.isNaN(storedWidth)
		? Math.floor(viewportWidth * TIMELINE_INITIAL_RATIO)
		: storedWidth;

	return clampTimelineWidth(initialWidth, viewportWidth);
}

export function useTimelineResize() {
	const [viewportWidth, setViewportWidth] = useState(getViewportWidth);
	const [timelineWidth, setTimelineWidth] = useState(() =>
		getInitialTimelineWidth(viewportWidth),
	);
	const timelineMaxWidth = getTimelineMaxWidth(viewportWidth);
	const stopTimelineResizeRef = useRef<(() => void) | null>(null);

	useEffect(() => {
		return () => stopTimelineResizeRef.current?.();
	}, []);

	useEffect(() => {
		const handleResize = () => {
			const nextViewportWidth = getViewportWidth();
			setViewportWidth(nextViewportWidth);
			setTimelineWidth((currentWidth) => {
				const nextWidth = clampTimelineWidth(currentWidth, nextViewportWidth);
				localStorage.setItem(TIMELINE_STORAGE_KEY, String(nextWidth));
				return nextWidth;
			});
		};

		window.addEventListener("resize", handleResize);
		return () => window.removeEventListener("resize", handleResize);
	}, []);

	const updateTimelineWidth = (width: number) => {
		const nextWidth = clampTimelineWidth(width, viewportWidth);
		localStorage.setItem(TIMELINE_STORAGE_KEY, String(nextWidth));
		setTimelineWidth(nextWidth);
	};

	const startTimelineResize: PointerEventHandler<HTMLDivElement> = (event) => {
		event.preventDefault();
		const startX = event.clientX;
		const startWidth = timelineWidth;

		const handlePointerMove = (pointerEvent: PointerEvent) => {
			updateTimelineWidth(startWidth + pointerEvent.clientX - startX);
		};
		const stopTimelineResize = () => {
			window.removeEventListener("pointermove", handlePointerMove);
			window.removeEventListener("pointerup", stopTimelineResize);
			window.removeEventListener("pointercancel", stopTimelineResize);
			stopTimelineResizeRef.current = null;
		};

		stopTimelineResizeRef.current?.();
		stopTimelineResizeRef.current = stopTimelineResize;
		window.addEventListener("pointermove", handlePointerMove);
		window.addEventListener("pointerup", stopTimelineResize);
		window.addEventListener("pointercancel", stopTimelineResize);
	};

	const handleTimelineResizerKeyDown: KeyboardEventHandler<HTMLDivElement> = (
		event,
	) => {
		if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") {
			return;
		}

		event.preventDefault();
		const direction = event.key === "ArrowLeft" ? -1 : 1;
		updateTimelineWidth(timelineWidth + direction * TIMELINE_KEYBOARD_STEP);
	};

	return {
		timelineWidth,
		timelineMinWidth: TIMELINE_MIN_WIDTH,
		timelineMaxWidth,
		startTimelineResize,
		handleTimelineResizerKeyDown,
	};
}
