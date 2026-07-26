// Pure presentation helpers for AI detection state — unit-tested, no DOM.
import { AiStatus } from "src/types/api";

// aiBadgeLabel is the short text next to the status icon: the queue position
// for pending images ("#12"), nothing otherwise.
export function aiBadgeLabel(status: AiStatus | null | undefined, position?: number): string {
  if (status === "pending" && position && position > 0) return `#${position}`;
  return "";
}

export function aiBadgeTitle(status: AiStatus | null | undefined, position?: number, error?: string): string {
  switch (status) {
    case "pending":
      return position && position > 0 ? `AI detection queued — position ${position}` : "AI detection queued";
    case "processing":
      return "AI detection running";
    case "done":
      return "AI detection done";
    case "error":
      return error ? `AI detection failed: ${error}` : "AI detection failed";
    default:
      return "";
  }
}

export interface RelativeBox {
  x: number;
  y: number;
  w: number;
  h: number;
}

// faceBoxStyle maps a relative (0..1) bbox to CSS percentages. Valid because
// the overlay wrapper hugs the rendered <img> exactly (inline-block, no
// object-fit crop) — the img box IS the image coordinate space.
export function faceBoxStyle(box: RelativeBox): Record<string, string> {
  const pct = (v: number) => `${(Math.max(0, Math.min(1, v)) * 100).toFixed(2)}%`;
  return { left: pct(box.x), top: pct(box.y), width: pct(box.w), height: pct(box.h) };
}

// Summary line for an upload's AI rollup: "12/40 done · 213 ahead".
export function aiUploadSummary(s: { pending: number; processing: number; done: number; error: number; ahead: number }): string {
  const total = s.pending + s.processing + s.done + s.error;
  if (total === 0) return "";
  let text = `${s.done}/${total} done`;
  if (s.error > 0) text += ` · ${s.error} failed`;
  if (s.pending + s.processing > 0 && s.ahead > 0) text += ` · ${s.ahead} ahead`;
  return text;
}
