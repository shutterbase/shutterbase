// Pure schedule-calendar logic (S15) — SFC-free so vitest can import it
// without DOM/Pinia (same pattern as uploadReview.ts). All rules the calendar
// renders are decided here; the components only paint.

export type OccupancyStatus = "empty" | "partial" | "full" | "over";

export interface ScheduleItemLike {
  id: string;
  start: string;
  end: string;
  cardinality: number;
  assignees: { id: string }[];
}

// occupancyStatus: cardinality is the TARGET, not a cap. over is not an error
// state — it renders violett ("Maximum Power"), full triggers confetti.
export function occupancyStatus(assigned: number, cardinality: number): OccupancyStatus {
  if (assigned === 0) return "empty";
  if (assigned < cardinality) return "partial";
  if (assigned === cardinality) return "full";
  return "over";
}

export const OCCUPANCY_LABEL: Record<OccupancyStatus, string> = {
  empty: "Unassigned",
  partial: "Needs people",
  full: "Fully covered",
  over: "Maximum Power",
};

// Tailwind classes per status: blau leer / gelb teilweise / grün voll /
// violett überbelegt (transcript 06:37–07:26).
export const OCCUPANCY_CLASSES: Record<OccupancyStatus, string> = {
  empty: "border-blue-400 bg-blue-500/15 text-blue-800 dark:text-blue-200",
  partial: "border-yellow-400 bg-yellow-400/20 text-yellow-800 dark:text-yellow-100",
  full: "border-green-500 bg-green-500/15 text-green-800 dark:text-green-200",
  over: "border-violet-500 bg-violet-500/20 text-violet-800 dark:text-violet-200",
};

// startOfDay in LOCAL time — the calendar is a wall-clock view.
export function startOfDay(d: Date): Date {
  const out = new Date(d);
  out.setHours(0, 0, 0, 0);
  return out;
}

export function addDays(d: Date, days: number): Date {
  const out = new Date(d);
  out.setDate(out.getDate() + days);
  return out;
}

function daySpan(from: Date, to: Date, cap: number): Date[] {
  const days: Date[] = [];
  for (let d = startOfDay(from); d.getTime() <= to.getTime() && days.length < cap; d = addDays(d, 1)) {
    days.push(d);
  }
  return days;
}

// calendarDays: the day columns to render. Project period wins; without one
// the span of the existing items; without items the current week (Mon–Sun).
// Capped at 31 columns so a typo'd period cannot render a year.
export function calendarDays(
  project: { startAt?: string | null; endAt?: string | null },
  items: ScheduleItemLike[],
  now: Date = new Date(),
): Date[] {
  const cap = 31;
  if (project.startAt && project.endAt) {
    const from = new Date(project.startAt);
    const to = new Date(project.endAt);
    if (!isNaN(from.getTime()) && !isNaN(to.getTime()) && from.getTime() <= to.getTime()) {
      return daySpan(from, to, cap);
    }
  }
  if (items.length > 0) {
    const from = new Date(Math.min(...items.map((i) => new Date(i.start).getTime())));
    const to = new Date(Math.max(...items.map((i) => new Date(i.end).getTime())));
    // An item ending exactly at midnight should not open an empty extra column.
    const toIncl = to.getTime() === startOfDay(to).getTime() ? new Date(to.getTime() - 1) : to;
    return daySpan(from, toIncl, cap);
  }
  const monday = addDays(startOfDay(now), -((now.getDay() + 6) % 7));
  return daySpan(monday, addDays(monday, 6), 7);
}

// itemsOnDay: items whose [start, end) window intersects the given day.
export function itemsOnDay<T extends ScheduleItemLike>(items: T[], day: Date): T[] {
  const dayStart = startOfDay(day).getTime();
  const dayEnd = addDays(startOfDay(day), 1).getTime();
  return items.filter((i) => new Date(i.start).getTime() < dayEnd && new Date(i.end).getTime() > dayStart);
}

// dayPosition: vertical placement of an item inside one day column, in % of
// the FULL 24h axis (the calendar always shows the whole day and fits the
// viewport — no vertical scrolling), clamped to the day (multi-day items span
// columns).
export function dayPosition(item: ScheduleItemLike, day: Date): { topPct: number; heightPct: number } {
  const windowStart = startOfDay(day).getTime();
  const windowLen = 24 * 3_600_000;
  const from = Math.max(new Date(item.start).getTime(), windowStart);
  const to = Math.min(new Date(item.end).getTime(), windowStart + windowLen);
  const topPct = Math.max(0, ((from - windowStart) / windowLen) * 100);
  const heightPct = Math.max(1.5, ((to - from) / windowLen) * 100);
  return { topPct, heightPct: Math.min(heightPct, 100 - topPct) };
}

// assignLanes: side-by-side lanes for overlapping items in one day column
// (interval partitioning, greedy by start). Returns per-item lane index and
// the total lane count of its overlap cluster.
export function assignLanes(items: ScheduleItemLike[]): Map<string, { lane: number; lanes: number }> {
  const sorted = [...items].sort((a, b) => new Date(a.start).getTime() - new Date(b.start).getTime());
  const laneEnds: number[] = [];
  const laneOf = new Map<string, number>();
  // cluster = maximal run of transitively overlapping items; lanes reset per cluster
  let cluster: string[] = [];
  let clusterEnd = -Infinity;
  const out = new Map<string, { lane: number; lanes: number }>();
  const flush = () => {
    const lanes = Math.max(1, ...cluster.map((id) => (laneOf.get(id) ?? 0) + 1));
    cluster.forEach((id) => out.set(id, { lane: laneOf.get(id) ?? 0, lanes }));
    cluster = [];
    laneEnds.length = 0;
  };
  for (const item of sorted) {
    const start = new Date(item.start).getTime();
    const end = new Date(item.end).getTime();
    if (cluster.length > 0 && start >= clusterEnd) {
      flush();
      clusterEnd = -Infinity;
    }
    let lane = laneEnds.findIndex((e) => e <= start);
    if (lane === -1) {
      lane = laneEnds.length;
      laneEnds.push(end);
    } else {
      laneEnds[lane] = end;
    }
    laneOf.set(item.id, lane);
    cluster.push(item.id);
    clusterEnd = Math.max(clusterEnd, end);
  }
  flush();
  return out;
}

// isAssigned: whether the user covers this item.
export function isAssigned(item: ScheduleItemLike, userId: string | undefined): boolean {
  return !!userId && item.assignees.some((a) => a.id === userId);
}
