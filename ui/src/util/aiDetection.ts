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

// cropSide is the edge length (in image px) of the square face crop: margin ×
// the face's larger edge, clamped to fit the image. 0 when uncomputable.
function cropSide(box: RelativeBox, imageWidth: number, imageHeight: number, margin: number): number {
  if (!imageWidth || !imageHeight) return 0;
  const edge = Math.max(box.w * imageWidth, box.h * imageHeight);
  if (edge <= 0) return 0;
  return Math.min(edge * margin, Math.min(imageWidth, imageHeight));
}

// faceRendition picks the smallest thumbnail rendition whose crop region still
// fills a tile of tilePx — deep crops need a larger source or they pixelate.
// The math is scale-invariant, so rendition dimensions work as well as
// original ones. Falls back to "512" without dimensions.
export function faceRendition(box: RelativeBox, imageWidth: number, imageHeight: number, tilePx = 512, margin = 2.75): string {
  const side = cropSide(box, imageWidth, imageHeight, margin);
  if (!side) return "512";
  const neededLongEdge = (tilePx * Math.max(imageWidth, imageHeight)) / side;
  return String([512, 1024, 2048].find((s) => s >= neededLongEdge) ?? 2048);
}

// faceCropStyle positions an <img> inside a square, overflow-hidden tile so
// the face bbox sits centered with generous margin — a face crop without
// server-side cropping. Returns percentage styles for the img and for the
// face box drawn inside the crop, or null when the image dimensions are
// unknown (caller falls back to object-cover).
export function faceCropStyle(box: RelativeBox, imageWidth: number, imageHeight: number, margin = 2.75): { img: Record<string, string>; box: Record<string, string> } | null {
  const side = cropSide(box, imageWidth, imageHeight, margin);
  if (!side) return null;
  const faceW = box.w * imageWidth;
  const faceH = box.h * imageHeight;
  const clamp = (v: number, max: number) => Math.max(0, Math.min(v, max));
  const left = clamp((box.x + box.w / 2) * imageWidth - side / 2, imageWidth - side);
  const top = clamp((box.y + box.h / 2) * imageHeight - side / 2, imageHeight - side);
  const pct = (v: number) => `${((v / side) * 100).toFixed(2)}%`;
  return {
    img: { width: pct(imageWidth), left: `-${pct(left)}`, top: `-${pct(top)}` },
    box: { left: pct(box.x * imageWidth - left), top: pct(box.y * imageHeight - top), width: pct(faceW), height: pct(faceH) },
  };
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
