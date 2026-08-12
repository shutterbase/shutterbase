// Client-side mirror of the server's upload review rules (api/internal/
// authorization). The server is authoritative — this exists so the UI never
// offers an action that would come back 403.
import { UploadState } from "src/types/api";

// Reserved per-project tags only a reviewer may assign: a tagging error found
// during review, and the image itself being rejected.
export const REVIEW_ERROR_TAG = "error";
export const REVIEW_REJECTED_TAG = "rejected";

export function isReviewErrorTag(name: string): boolean {
  return name.toLowerCase() === REVIEW_ERROR_TAG;
}

export function isReviewRejectedTag(name: string): boolean {
  return name.toLowerCase() === REVIEW_REJECTED_TAG;
}

export function isReviewerOnlyTag(name: string): boolean {
  return isReviewErrorTag(name) || isReviewRejectedTag(name);
}

export const UPLOAD_STATES: UploadState[] = ["open", "ready", "reviewed"];

export const UPLOAD_STATE_LABEL: Record<UploadState, string> = {
  open: "Open",
  ready: "Ready for review",
  reviewed: "Reviewed",
};

export const UPLOAD_STATE_HINT: Record<UploadState, string> = {
  open: "Being tagged by the photographer",
  ready: "Submitted — waiting for a project admin",
  reviewed: "Accepted by a project admin",
};

// An upload that left "open" is submitted: its official tags are frozen.
export function isUploadSubmitted(state: UploadState | undefined): boolean {
  return state === "ready" || state === "reviewed";
}

export interface TagEditContext {
  reviewEnabled: boolean;
  uploadState?: UploadState;
  tagType: string; // template | default | manual | custom
  tagName: string;
  isReviewer: boolean; // projectAdmin or global admin
  isEditor: boolean; // projectEditor or higher
}

// Mirrors authorization.CanAssignTag: with the review flow on, a non-reviewer
// loses the reserved review tags entirely, and every non-custom ("official",
// exported) tag once the upload has been submitted.
export function canEditTag(ctx: TagEditContext): boolean {
  if (!ctx.isEditor) return false;
  if (!ctx.reviewEnabled || ctx.isReviewer) return true;
  if (isReviewerOnlyTag(ctx.tagName)) return false;
  return !isUploadSubmitted(ctx.uploadState) || ctx.tagType === "custom";
}

export interface TransitionActor {
  isReviewer: boolean;
  isOwner: boolean;
}

// Mirrors authorization.CanTransitionUpload: the photographer may only submit;
// sending back, accepting and reopening belong to the reviewer.
export function allowedTransitions(state: UploadState, actor: TransitionActor): UploadState[] {
  if (actor.isReviewer) return UPLOAD_STATES.filter((s) => s !== state);
  if (actor.isOwner && state === "open") return ["ready"];
  return [];
}

export const TRANSITION_LABEL: Record<UploadState, string> = {
  open: "Send back",
  ready: "Mark ready",
  reviewed: "Accept",
};

// Mirrors authorization.CanAddImagesToUpload: a submitted upload takes no
// further images from the photographer — only a reviewer can still add.
export function canAddImages(input: { reviewEnabled: boolean; uploadState?: UploadState; isReviewer: boolean }): boolean {
  return !input.reviewEnabled || !isUploadSubmitted(input.uploadState) || input.isReviewer;
}

// Compact human duration for the metric badges: "1h 12m", "4m 30s", "12s".
export function formatDuration(seconds: number): string {
  if (!seconds || seconds < 0) return "0s";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

// Images per second is the metric that was asked for, but at a realistic tagging
// pace it reads as 0.08 — so show the per-minute rate alongside it.
export function formatTaggingRate(imagesPerSecond: number): string {
  if (!imagesPerSecond) return "–";
  return `${imagesPerSecond.toFixed(3)} img/s (${(imagesPerSecond * 60).toFixed(1)}/min)`;
}
