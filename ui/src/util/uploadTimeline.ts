// Pure logic of the upload tagging timeline (S15) — SFC-free for vitest, same
// pattern as uploadReview.ts/schedule.ts. The editor component only renders
// and forwards gestures; every rule lives here so the UI can never offer a
// state the server would 400.

import { ScheduleItem, TimelineTrack } from "src/types/api";

export interface EditorTrack {
  key: string; // stable render key
  scheduleItemId?: string;
  tagId?: string;
  label: string;
  start: number; // epoch ms
  end: number; // epoch ms
  enabled: boolean;
}

export interface TimedImage {
  id: string;
  time: number; // corrected capture time, epoch ms
}

export const isScheduleTrack = (t: Pick<EditorTrack, "scheduleItemId">): boolean => !!t.scheduleItemId;

// Minimum track length the editor allows (keeps handles grabbable).
export const MIN_TRACK_MS = 60_000;

// timelineWindow: the axis bounds — EXACTLY the span from the first to the
// last picture (plus the minimum track length so the last image stays
// coverable under the [start, end) semantics). Tracks never widen the axis:
// a schedule item reaching past the photos would otherwise stretch the whole
// editor and break Expand. Track-union fallback only when there are no timed
// images at all; one hour around now when there is nothing.
export function timelineWindow(images: TimedImage[], tracks: EditorTrack[], now = Date.now()): { start: number; end: number } {
  if (images.length > 0) {
    const times = images.map((i) => i.time);
    return { start: Math.min(...times), end: Math.max(...times) + MIN_TRACK_MS };
  }
  if (tracks.length > 0) {
    return {
      start: Math.min(...tracks.map((t) => t.start)),
      end: Math.max(...tracks.map((t) => t.end)),
    };
  }
  return { start: now - 1_800_000, end: now + 1_800_000 };
}

// scheduleNeighbors: the enabled schedule tracks other than `track`, the
// mutually-exclusive set it must not intersect (transcript 24:55).
function scheduleNeighbors(track: EditorTrack, tracks: EditorTrack[]): EditorTrack[] {
  if (!isScheduleTrack(track)) return [];
  return tracks.filter((t) => t.key !== track.key && isScheduleTrack(t) && t.enabled);
}

// clampWindow: the [min, max] range this track's edges may occupy — the axis
// window, shrunk by enabled schedule siblings for schedule tracks.
export function clampWindow(track: EditorTrack, tracks: EditorTrack[], window: { start: number; end: number }): { start: number; end: number } {
  let start = window.start;
  let end = window.end;
  for (const n of scheduleNeighbors(track, tracks)) {
    if (n.end <= track.start && n.end > start) start = n.end; // neighbor before
    if (n.start >= track.end && n.start < end) end = n.start; // neighbor after
  }
  return { start, end };
}

// expandTrack: the one-click Expand (transcript 22:11): grow to the maximum
// size that does not compromise another enabled schedule track — tag tracks
// expand to the full window.
export function expandTrack(track: EditorTrack, tracks: EditorTrack[], window: { start: number; end: number }): EditorTrack {
  const bounds = clampWindow(track, tracks, window);
  return { ...track, start: bounds.start, end: bounds.end };
}

// moveEdge: drag or nudge one edge to a target time, constrained by the clamp
// window and the minimum length. Returns the adjusted track.
export function moveEdge(track: EditorTrack, edge: "start" | "end", to: number, tracks: EditorTrack[], window: { start: number; end: number }): EditorTrack {
  const bounds = clampWindow(track, tracks, window);
  if (edge === "start") {
    const start = Math.min(Math.max(to, bounds.start), track.end - MIN_TRACK_MS);
    return { ...track, start };
  }
  const end = Math.max(Math.min(to, bounds.end), track.start + MIN_TRACK_MS);
  return { ...track, end };
}

// stepEdgeByImages: keyboard nudging steps in PICTURES, not minutes — the
// photographer thinks in frames, and a minute may hold 0 or 30 of them.
// `delta` is the arrow direction (negative = left) times the step size; the
// edge snaps onto an image time so each step moves the covered set by exactly
// that many pictures ([start, end): the in-point lands ON its first image, the
// out-point on the first excluded one). Past the last image the edge parks at
// the axis end so the final picture stays coverable.
export function stepEdgeByImages(images: TimedImage[], track: EditorTrack, edge: "start" | "end", delta: number, window: { start: number; end: number }): number {
  const times = images.map((i) => i.time).sort((a, b) => a - b);
  if (times.length === 0) return edge === "start" ? track.start : track.end;
  const current = edge === "end" ? times.filter((t) => t < track.end).length : times.findIndex((t) => t >= track.start);
  const index = Math.min(Math.max((current === -1 ? times.length : current) + delta, 0), times.length);
  return index < times.length ? times[index] : window.end;
}

// setEnabled: enabling a schedule track is refused while it would overlap
// another enabled schedule track (the mirror of the server's 400).
export function setEnabled(track: EditorTrack, enabled: boolean, tracks: EditorTrack[]): EditorTrack | null {
  if (enabled && isScheduleTrack(track)) {
    const overlap = scheduleNeighbors({ ...track, enabled: true }, tracks).some((n) => n.start < track.end && n.end > track.start);
    if (overlap) return null;
  }
  return { ...track, enabled };
}

// imagesInTrack: the covered set — [start, end), matching the server.
export function imagesInTrack(images: TimedImage[], track: EditorTrack): TimedImage[] {
  return images.filter((i) => i.time >= track.start && i.time < track.end);
}

// boundaryImages: what the in/out handles preview (transcript 23:06 / 27:16):
// the last image before the window, first/last inside, first after.
export function boundaryImages(images: TimedImage[], track: EditorTrack): { before?: TimedImage; first?: TimedImage; last?: TimedImage; after?: TimedImage } {
  const sorted = [...images].sort((a, b) => a.time - b.time);
  const inside = sorted.filter((i) => i.time >= track.start && i.time < track.end);
  return {
    before: [...sorted].reverse().find((i) => i.time < track.start),
    first: inside[0],
    last: inside[inside.length - 1],
    after: sorted.find((i) => i.time >= track.end),
  };
}

// initialTracks: what the editor opens with. The persisted upload timeline is
// restored verbatim (the editor reopens exactly as left, incl. disabled lanes),
// then the uploader's own schedule items intersecting the image span are merged
// on top at their real windows (transcript 21:37) — an item assigned AFTER the
// timeline was first applied still shows up on its own. Already-present items
// are skipped by addScheduleTrack, so restored lanes keep their edited
// in/out points and their enabled flag.
export function initialTracks(
  persisted: TimelineTrack[] | undefined,
  myItems: ScheduleItem[],
  images: TimedImage[],
  labels: { scheduleItem: (id: string) => string; tag: (id: string) => string },
): EditorTrack[] {
  const restored = (persisted ?? []).map((t, i) => ({
    key: `p${i}`,
    scheduleItemId: t.scheduleItemId || undefined,
    tagId: t.tagId || undefined,
    label: t.scheduleItemId ? labels.scheduleItem(t.scheduleItemId) : labels.tag(t.tagId ?? ""),
    start: Date.parse(t.start),
    end: Date.parse(t.end),
    enabled: t.enabled,
  }));
  if (images.length === 0) return restored;
  const span = { start: Math.min(...images.map((i) => i.time)), end: Math.max(...images.map((i) => i.time)) };
  return myItems.filter((item) => Date.parse(item.start) < span.end && Date.parse(item.end) > span.start).reduce(addScheduleTrack, restored);
}

// addTagTrack / addScheduleTrack: the lane pickers. New tag lanes span the
// full axis (the image range — user shrinks from there); schedule lanes come
// in at their real window, disabled when they would collide with an enabled
// schedule lane.
export function addTagTrack(tracks: EditorTrack[], tagId: string, label: string, window: { start: number; end: number }): EditorTrack[] {
  return [...tracks, { key: `t${tagId}-${tracks.length}`, tagId, label, start: window.start, end: window.end, enabled: true }];
}

export function addScheduleTrack(tracks: EditorTrack[], item: ScheduleItem): EditorTrack[] {
  if (tracks.some((t) => t.scheduleItemId === item.id)) return tracks;
  const candidate: EditorTrack = {
    key: `s${item.id}`,
    scheduleItemId: item.id,
    label: item.title,
    start: Date.parse(item.start),
    end: Date.parse(item.end),
    enabled: true,
  };
  const collides = tracks.some((t) => isScheduleTrack(t) && t.enabled && t.start < candidate.end && t.end > candidate.start);
  return [...tracks, collides ? { ...candidate, enabled: false } : candidate];
}

// toApiTracks: serialize for PUT /uploads/:id/timeline.
export function toApiTracks(tracks: EditorTrack[]): TimelineTrack[] {
  return tracks.map((t) => ({
    scheduleItemId: t.scheduleItemId,
    tagId: t.tagId,
    start: new Date(t.start).toISOString(),
    end: new Date(t.end).toISOString(),
    enabled: t.enabled,
  }));
}

// hasScheduleOverlap: mirror of the server-side ErrScheduleOverlap check.
export function hasScheduleOverlap(tracks: EditorTrack[]): boolean {
  const enabled = tracks.filter((t) => isScheduleTrack(t) && t.enabled).sort((a, b) => a.start - b.start);
  for (let i = 1; i < enabled.length; i++) {
    if (enabled[i].start < enabled[i - 1].end) return true;
  }
  return false;
}
