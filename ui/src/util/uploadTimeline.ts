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

// timelineWindow: the axis bounds — the images' span united with every track,
// padded so boundary handles stay reachable. Empty input: one hour around now.
export function timelineWindow(images: TimedImage[], tracks: EditorTrack[], now = Date.now()): { start: number; end: number } {
  const times = [...images.map((i) => i.time), ...tracks.flatMap((t) => [t.start, t.end])];
  if (times.length === 0) return { start: now - 1_800_000, end: now + 1_800_000 };
  const min = Math.min(...times);
  const max = Math.max(...times);
  const pad = Math.max(600_000, (max - min) * 0.05); // >= 10 min
  return { start: min - pad, end: max + pad };
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

// initialTracks: what the editor opens with. The persisted upload timeline
// wins (the editor reopens exactly as left, incl. disabled lanes); otherwise
// the uploader's own schedule items intersecting the image span are
// pre-populated at their real windows (transcript 21:37).
export function initialTracks(
  persisted: TimelineTrack[] | undefined,
  myItems: ScheduleItem[],
  images: TimedImage[],
  labels: { scheduleItem: (id: string) => string; tag: (id: string) => string },
): EditorTrack[] {
  if (persisted && persisted.length > 0) {
    return persisted.map((t, i) => ({
      key: `p${i}`,
      scheduleItemId: t.scheduleItemId || undefined,
      tagId: t.tagId || undefined,
      label: t.scheduleItemId ? labels.scheduleItem(t.scheduleItemId) : labels.tag(t.tagId ?? ""),
      start: Date.parse(t.start),
      end: Date.parse(t.end),
      enabled: t.enabled,
    }));
  }
  if (images.length === 0) return [];
  const span = { start: Math.min(...images.map((i) => i.time)), end: Math.max(...images.map((i) => i.time)) };
  return myItems
    .filter((item) => Date.parse(item.start) < span.end && Date.parse(item.end) > span.start)
    .map((item) => ({
      key: `s${item.id}`,
      scheduleItemId: item.id,
      label: item.title,
      start: Date.parse(item.start),
      end: Date.parse(item.end),
      enabled: true,
    }));
}

// addTagTrack / addScheduleTrack: the plus button. New tag lanes span the
// image range (user shrinks from there); schedule lanes come in at their real
// window, disabled when they would collide with an enabled schedule lane.
export function addTagTrack(tracks: EditorTrack[], tagId: string, label: string, images: TimedImage[], window: { start: number; end: number }): EditorTrack[] {
  const span = images.length > 0 ? { start: Math.min(...images.map((i) => i.time)), end: Math.max(...images.map((i) => i.time)) + MIN_TRACK_MS } : window;
  return [...tracks, { key: `t${tagId}-${tracks.length}`, tagId, label, start: span.start, end: span.end, enabled: true }];
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
