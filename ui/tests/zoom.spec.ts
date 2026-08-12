import { describe, it, expect } from "vitest";
import { MAX_ZOOM, ZOOM_RESET, panBy, zoomAt } from "src/util/zoom";

// viewport 400×300 throughout
const W = 400;
const H = 300;

describe("zoomAt", () => {
  it("keeps the point under the cursor fixed while scaling", () => {
    const state = zoomAt(ZOOM_RESET, 100, 50, 2, W, H);
    expect(state.scale).toBe(2);
    // content point that was at (100, 50) must still project to (100, 50):
    // p = c * scale + t, with c = (p - t_old) / scale_old = (100, 50)
    expect(100 * 2 + state.tx).toBe(100);
    expect(50 * 2 + state.ty).toBe(50);
  });

  it("clamps scale to the [1, MAX_ZOOM] range and resets offsets at scale 1", () => {
    expect(zoomAt(ZOOM_RESET, 0, 0, 100, W, H).scale).toBe(MAX_ZOOM);
    const zoomed = zoomAt(ZOOM_RESET, 200, 150, 2, W, H);
    expect(zoomAt(zoomed, 0, 0, 0.5, W, H)).toEqual(ZOOM_RESET);
  });

  it("keeps the cursor anchor when zooming at a corner", () => {
    // zooming at the bottom-right corner pulls the content's corner there
    const state = zoomAt(ZOOM_RESET, W, H, 2, W, H);
    expect(state.tx).toBe(W * (1 - 2));
    expect(state.ty).toBe(H * (1 - 2));
  });

  it("lets either image edge reach the viewport centre", () => {
    const zoomed = zoomAt(ZOOM_RESET, 200, 150, 2, W, H); // scaled content 800×600
    expect(panBy(zoomed, 10_000, 0, W, H).tx).toBe(W / 2); // left edge at the centre
    expect(panBy(zoomed, -10_000, 0, W, H).tx + W * 2).toBe(W / 2); // right edge at the centre
    expect(panBy(zoomed, 0, 10_000, W, H).ty).toBe(H / 2);
    expect(panBy(zoomed, 0, -10_000, W, H).ty + H * 2).toBe(H / 2);
  });
});

describe("zoomAt with content smaller than the stage", () => {
  it("moves within the stage plus the pan slack", () => {
    // fitted image 200×100 in a 400×300 stage, zoomed 2× → scaled 400×200
    const zoomed = zoomAt(ZOOM_RESET, 0, 0, 2, W, H, 200, 100);
    expect(zoomed.scale).toBe(2);
    expect(zoomed.tx).toBe(0); // scaled width equals the stage → nothing to pan into
    // bottom edge of the stage (100) plus half a stage height of slack
    expect(panBy(zoomed, 0, 10_000, W, H, 200, 100).ty).toBe(100 + H / 2);
    expect(panBy(zoomed, 0, -10_000, W, H, 200, 100).ty).toBe(-H / 2);
  });
});

describe("panBy", () => {
  it("moves within bounds and stops at the edges", () => {
    const zoomed = zoomAt(ZOOM_RESET, 200, 150, 2, W, H);
    const moved = panBy(zoomed, -10, -10, W, H);
    expect(moved.tx).toBe(zoomed.tx - 10);
    expect(moved.ty).toBe(zoomed.ty - 10);
    const pinned = panBy(zoomed, 10_000, 10_000, W, H);
    expect(pinned.tx).toBe(W / 2);
    expect(pinned.ty).toBe(H / 2);
    const pinnedFar = panBy(zoomed, -10_000, -10_000, W, H);
    expect(pinnedFar.tx).toBe(W * (1 - 2) - W / 2);
    expect(pinnedFar.ty).toBe(H * (1 - 2) - H / 2);
  });
});
