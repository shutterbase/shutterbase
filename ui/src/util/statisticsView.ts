// Pure chart-data transforms for the project statistics dashboard — no DOM,
// unit-tested. Components render exactly what comes out of here.
import { DateTime } from "luxon";
import { StatAssignmentDay, StatDay, StatPhotographer } from "src/api/statistics";
import { AI_COLOR, CATEGORICAL, MANUAL_COLOR, OTHER_COLOR } from "src/util/chartColors";
import { shortDayLabel } from "src/util/dateTimeUtil";

export interface ChartSegment {
  key: string;
  label: string;
  color: string;
  value: number;
}

export interface ChartDay {
  date: string;
  label: string;
  total: number;
  segments: ChartSegment[];
}

export interface SeriesEntry {
  key: string;
  label: string;
  color: string;
}

// fillDayRange returns every ISO date from the first to the last entry so the
// x-axis has no silent holes. Ranges beyond 120 days stay sparse — a column
// chart with hundreds of empty columns reads worse than a gap.
export function fillDayRange(dates: string[]): string[] {
  if (dates.length < 2) return [...dates];
  const sorted = [...dates].sort();
  const start = DateTime.fromISO(sorted[0]);
  const end = DateTime.fromISO(sorted[sorted.length - 1]);
  const span = end.diff(start, "days").days;
  if (span > 120) return sorted;
  const out: string[] = [];
  for (let d = start; d <= end; d = d.plus({ days: 1 })) {
    out.push(d.toISODate() as string);
  }
  return out;
}

export function photographerLabel(p: StatPhotographer): string {
  const name = `${p.firstName} ${p.lastName}`.trim();
  return name || p.copyrightTag || p.id;
}

// foldPhotographers assigns the fixed categorical slots to the top photographers
// (already sorted by imageCount desc) and folds the tail into a gray "Other".
// Color follows the entity: slot order is overall rank, stable across toggles.
export function foldPhotographers(photographers: StatPhotographer[], max = CATEGORICAL.length): SeriesEntry[] {
  const series: SeriesEntry[] = photographers.slice(0, max).map((p, i) => ({
    key: p.id,
    label: photographerLabel(p),
    color: CATEGORICAL[i],
  }));
  if (photographers.length > max) {
    series.push({ key: "other", label: "Other", color: OTHER_COLOR });
  }
  return series;
}

// buildImageDaySeries lays the per-day image counts out as chart columns —
// single accent series when split is off, stacked photographer series when on.
export function buildImageDaySeries(days: StatDay[], series: SeriesEntry[], split: boolean): ChartDay[] {
  const byDate = new Map(days.map((d) => [d.date, d]));
  return fillDayRange(days.map((d) => d.date)).map((date) => {
    const day = byDate.get(date);
    const total = day?.total ?? 0;
    let segments: ChartSegment[] = [];
    if (!split) {
      segments = total > 0 ? [{ key: "total", label: "Images", color: CATEGORICAL[0], value: total }] : [];
    } else if (day) {
      const folded = new Map<string, number>();
      const slotKeys = new Set(series.map((s) => s.key));
      for (const [userId, count] of Object.entries(day.byUser)) {
        const key = slotKeys.has(userId) ? userId : "other";
        folded.set(key, (folded.get(key) ?? 0) + count);
      }
      segments = series.filter((s) => folded.has(s.key)).map((s) => ({ ...s, value: folded.get(s.key) as number }));
    }
    return { date, label: shortDayLabel(date), total, segments };
  });
}

// buildAssignmentDaySeries: manual-vs-AI tagging effort per day.
export function buildAssignmentDaySeries(days: StatAssignmentDay[]): ChartDay[] {
  const byDate = new Map(days.map((d) => [d.date, d]));
  return fillDayRange(days.map((d) => d.date)).map((date) => {
    const day = byDate.get(date);
    const segments: ChartSegment[] = [];
    if (day?.manual) segments.push({ key: "manual", label: "Manual", color: MANUAL_COLOR, value: day.manual });
    if (day?.ai) segments.push({ key: "ai", label: "AI", color: AI_COLOR, value: day.ai });
    return { date, label: shortDayLabel(date), total: (day?.manual ?? 0) + (day?.ai ?? 0), segments };
  });
}

export function peakDay(days: StatDay[]): StatDay | null {
  let peak: StatDay | null = null;
  for (const day of days) {
    if (!peak || day.total > peak.total) peak = day;
  }
  return peak;
}
