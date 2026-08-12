// Pure slideshow logic — SFC-free so it is trivially unit-testable.

export interface SlideshowConfig {
  /** seconds each image is fully visible */
  showSeconds: number;
  /** crossfade duration in seconds */
  transitionSeconds: number;
  /** Ken Burns pan/zoom on the visible image */
  kenBurns: "off" | "subtle" | "strong";
  /** restart from the first image after the last one */
  loop: boolean;
}

export const DEFAULT_SLIDESHOW_CONFIG: SlideshowConfig = {
  showSeconds: 6,
  transitionSeconds: 1.5,
  kenBurns: "subtle",
  loop: true,
};

/** How many upcoming images are kept decoded ahead of the one on screen. */
export const PRELOAD_HEADROOM = 5;

/** Reserved management tag: these shots never leave the house. */
export const INTERNAL_TAG = "internal";

/**
 * Images marked internal are never shown in a slideshow. Combo tags
 * ("trip|internal") are checked per part, exactly like the EXIF keyword export
 * (api/internal/exif/inject.go) — the marker cannot be smuggled in as a part.
 */
export function isInternalImage(image: { tags?: { tag?: { name?: string } }[] }): boolean {
  return (image.tags ?? []).some((assignment) => (assignment.tag?.name ?? "").split("|").some((part) => part.trim().toLowerCase() === INTERNAL_TAG));
}

/** Indices to have preloaded for `current` (the next `headroom` loaded images). */
export function preloadIndices(current: number, loadedCount: number, headroom: number = PRELOAD_HEADROOM): number[] {
  const indices: number[] = [];
  for (let i = current + 1; i <= current + headroom && i < loadedCount; i++) {
    indices.push(i);
  }
  return indices;
}

/**
 * Fetch the next page once the preload window would run into the end of the
 * loaded list (double headroom, so fetching starts well before display catches
 * up) — but only while the server has more.
 */
export function shouldFetchMore(current: number, loadedCount: number, total: number, headroom: number = PRELOAD_HEADROOM): boolean {
  if (loadedCount >= total) return false;
  return current + 2 * headroom >= loadedCount;
}

/** Next slide index; null ends the show (last image, loop off). */
export function nextSlideIndex(current: number, loadedCount: number, loop: boolean): number | null {
  if (loadedCount === 0) return null;
  if (current + 1 < loadedCount) return current + 1;
  return loop ? 0 : null;
}

export function previousSlideIndex(current: number, loadedCount: number, loop: boolean): number {
  if (current > 0) return current - 1;
  return loop && loadedCount > 0 ? loadedCount - 1 : 0;
}

/** Deterministic Ken Burns variant (0..3) so a slide always animates the same way. */
export function kenBurnsVariant(index: number): number {
  return ((index % 4) + 4) % 4;
}
