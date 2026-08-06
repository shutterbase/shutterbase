// Pure zoom/pan math for ZoomableImage. The viewport is the clip container
// (w × h); the content (base size contentW × contentH, defaulting to the
// viewport) is transformed with `translate(tx, ty) scale(scale)` from the
// top-left origin. Offsets are clamped so the scaled content never leaves the
// viewport: it covers it when larger, stays inside it when smaller — the
// latter happens when the zoom stage expands beyond the fitted image.

export interface ZoomState {
  scale: number;
  tx: number;
  ty: number;
}

export const MIN_ZOOM = 1;
export const MAX_ZOOM = 8;

export const ZOOM_RESET: ZoomState = { scale: 1, tx: 0, ty: 0 };

function clampOffset(value: number, viewport: number, scaledContent: number): number {
  const lo = Math.min(0, viewport - scaledContent);
  const hi = Math.max(0, viewport - scaledContent);
  return Math.min(hi, Math.max(lo, value));
}

// zoomAt scales to targetScale (clamped to [MIN_ZOOM, MAX_ZOOM]) while keeping
// the content point under the viewport point (px, py) fixed.
export function zoomAt(state: ZoomState, px: number, py: number, targetScale: number, width: number, height: number, contentW = width, contentH = height): ZoomState {
  const scale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, targetScale));
  if (scale === 1) return ZOOM_RESET;
  return {
    scale,
    tx: clampOffset(px - ((px - state.tx) / state.scale) * scale, width, contentW * scale),
    ty: clampOffset(py - ((py - state.ty) / state.scale) * scale, height, contentH * scale),
  };
}

export function panBy(state: ZoomState, dx: number, dy: number, width: number, height: number, contentW = width, contentH = height): ZoomState {
  return {
    scale: state.scale,
    tx: clampOffset(state.tx + dx, width, contentW * state.scale),
    ty: clampOffset(state.ty + dy, height, contentH * state.scale),
  };
}
