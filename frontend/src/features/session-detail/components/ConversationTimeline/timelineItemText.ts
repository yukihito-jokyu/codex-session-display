import type { dto } from "wailsjs/go/models";

export const TIMELINE_PREVIEW_LENGTH = 600;

function* getTimelineItemTextChunks(
	item: dto.ConversationTimelineItem,
): Generator<string> {
	let hasContent = false;
	if (item.body) {
		yield item.body;
		hasContent = true;
	}

	for (const detail of item.details ?? []) {
		if (hasContent) {
			yield "\n\n";
		}
		yield detail.label;
		yield "\n";
		yield detail.value;
		hasContent = true;
	}
}

export function getTimelineItemFullText(
	item: dto.ConversationTimelineItem,
): string {
	return Array.from(getTimelineItemTextChunks(item)).join("");
}

export function getTimelineItemTextLength(
	item: dto.ConversationTimelineItem,
): number {
	let length = 0;
	for (const chunk of getTimelineItemTextChunks(item)) {
		length += chunk.length;
	}
	return length;
}

export function getTimelineItemPreview(
	item: dto.ConversationTimelineItem,
): string {
	let preview = "";
	for (const chunk of getTimelineItemTextChunks(item)) {
		const remainingLength = TIMELINE_PREVIEW_LENGTH - preview.length;
		if (remainingLength <= 0) {
			break;
		}
		preview += chunk.slice(0, remainingLength);
	}
	return preview;
}
