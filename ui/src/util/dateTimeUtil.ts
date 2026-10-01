import { DateTime } from "luxon";
import { TimeOffset } from "src/types/api";

export function dateFromUnix(unixTime: number): string {
  const date = new Date(unixTime * 1000);
  return DateTime.fromJSDate(date).toFormat("dd.LL.iiii");
}

export function timeFromUnix(unixTime: number): string {
  const date = new Date(unixTime * 1000);
  return DateTime.fromJSDate(date).toFormat("HH:mm:ss");
}

export function dateTimeFromUnix(unixTime: number): string {
  const date = new Date(unixTime * 1000);
  return DateTime.fromJSDate(date).toFormat("dd.LL.iiii HH:mm:ss");
}

export function dateFromBackend(backendTime: string): string {
  return DateTime.fromJSDate(parseBackendTime(backendTime)).toFormat("dd.LL.iiii");
}

export function timeFromBackend(backendTime: string): string {
  return DateTime.fromJSDate(parseBackendTime(backendTime)).toFormat("HH:mm:ss");
}

export function dateTimeFromBackend(backendTime: string): string {
  return DateTime.fromJSDate(parseBackendTime(backendTime)).toFormat("dd.LL.iiii HH:mm:ss");
}

export function dateTimeToBackendString(date: Date): string {
  return DateTime.fromJSDate(date).toFormat("yyyy-MM-dd HH:mm:ss");
}

/** "2026-08-11" → "Di 11.08." — compact axis/day label from a plain ISO date. */
export function shortDayLabel(isoDate: string): string {
  return DateTime.fromISO(isoDate).toFormat("ccc dd.LL.");
}

export function parseBackendTime(backendTime: string): Date {
  return new Date(Date.parse(backendTime));
}

/**
 * Signed, human-readable offset the time-sync applied to a capture time —
 * corrected minus original, whole seconds: "+1h 02m 03s", "-45s", "±0s".
 */
export function appliedTimeOffset(capturedAt: string, capturedAtCorrected: string): string {
  const seconds = Math.round((parseBackendTime(capturedAtCorrected).getTime() - parseBackendTime(capturedAt).getTime()) / 1000);
  if (seconds === 0) return "±0s";
  const sign = seconds < 0 ? "-" : "+";
  const total = Math.abs(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  if (h > 0) return `${sign}${h}h ${pad(m)}m ${pad(s)}s`;
  if (m > 0) return `${sign}${m}m ${pad(s)}s`;
  return `${sign}${s}s`;
}

/**
 * Whole unix seconds from a backend timestamp string. Backend timestamps carry
 * sub-second precision (Postgres keeps microseconds), so anything feeding a
 * BigInt/i64 must truncate first — BigInt() throws a RangeError on a fraction.
 */
export function backendTimeToUnixSeconds(backendTime: string): number {
  return Math.floor(parseBackendTime(backendTime).getTime() / 1000);
}

/** Map API time offsets onto the WASM `TimeOffsetResult` shape (whole-second bigints). */
export function toWasmTimeOffsets(offsets: TimeOffset[]) {
  return offsets.map((timeOffset) => ({
    free: (): void => {},
    time_offset: BigInt(Math.round(timeOffset.timeOffset)),
    server_time: BigInt(backendTimeToUnixSeconds(timeOffset.serverTime)),
    camera_time: BigInt(backendTimeToUnixSeconds(timeOffset.cameraTime)),
  }));
}

export function timeOffsetUpToDate(timeOffset: TimeOffset): boolean {
  const serverTime = parseBackendTime(timeOffset.serverTime);
  return DateTime.fromJSDate(serverTime) > DateTime.now().minus({ hours: 24 });
}

/**
 * ISO string → value for `<input type="datetime-local">` (local wall clock,
 * minute precision). Empty string for null/invalid, as the input expects.
 */
export function isoToLocalInput(iso?: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const pad = (n: number, width = 2) => String(n).padStart(width, "0");
  // The year is padded to 4 digits: datetime-local requires it, and an unpadded
  // "999" is a value the control silently refuses to show — which then fails
  // the round-trip guard below and deletes the bound.
  return `${pad(d.getFullYear(), 4)}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// The canonical shape of a datetime-local value: YYYY-MM-DDTHH:mm. Anything the
// browser produces is minute-precision, so only the first 16 characters take
// part in the round-trip check.
const LOCAL_INPUT_SHAPE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/;

/**
 * `<input type="datetime-local">` value → ISO string (UTC), null when empty or
 * unparseable.
 *
 * `endOfMinute` widens the instant to the last millisecond of the entered
 * minute. The backend applies the `to` bound as an inclusive `LTE`, and both
 * the input and the slider only carry minute precision — so without this every
 * photo captured inside the final minute of a range is silently dropped
 * ("through 23:59" excluding 23:59:40).
 *
 * DST spring-forward gap: a wall-clock time inside the gap (e.g. 02:30 on the
 * transition day, which never happens) is rolled forward by the Date parser,
 * shifting the filter by an hour with nothing visible on screen. That is the one
 * mismatch worth rejecting, and it is detected by formatting the parsed instant
 * back to a local input string and comparing the minute.
 *
 * Only the MINUTE is compared, on purpose. An exact string comparison also
 * rejected perfectly valid values — one carrying seconds
 * ("2026-08-11T10:00:30"), or a date-only one ("2026-08-11", which `new Date`
 * reads as UTC midnight and would shift by the UTC offset). Rejecting those
 * returns null, the caller treats null as "no bound", and the range silently
 * opens to every photo — a much worse outcome than accepting them.
 */
export function localInputToIso(value?: string | null, endOfMinute = false): string | null {
  if (!value) return null;
  const d = new Date(value);
  if (isNaN(d.getTime())) return null;
  if (LOCAL_INPUT_SHAPE.test(value) && isoToLocalInput(d.toISOString()) !== value.slice(0, 16)) {
    // A full minute-precision value that does not map back to itself: the
    // entered wall clock does not exist (DST gap).
    return null;
  }
  if (endOfMinute) {
    d.setSeconds(59, 999);
  }
  return d.toISOString();
}

/**
 * Last millisecond of the minute a `datetime-local` value names, as ISO.
 * Returns null for empty/invalid input. Used for the inclusive `to` bound.
 */
export function localInputToIsoInclusive(value?: string | null): string | null {
  return localInputToIso(value, true);
}

/**
 * Last millisecond of the minute an INSTANT falls in, as ISO. For callers that
 * already hold an ISO timestamp (the slider, and the inverted-range clamp) and
 * need the same inclusive bound the inputs produce — one rule, so the two
 * producers cannot drift.
 *
 * Pure epoch arithmetic, deliberately NOT `d.setSeconds(59, 999)`: that setter
 * re-resolves the LOCAL wall clock, so an instant inside the DST repeated hour
 * is rewound by the DST offset. In Europe/Berlin, 2026-10-25T01:30Z (local 02:30
 * CET, the second pass through 02:00–03:00) came back as 00:30:59.999Z — silently
 * dropping the last 30 real minutes of the `to` bound.
 */
export function isoToEndOfMinute(iso?: string | null): string | null {
  if (!iso) return null;
  const ms = new Date(iso).getTime();
  if (isNaN(ms)) return null;
  return new Date(Math.floor(ms / 60_000) * 60_000 + 59_999).toISOString();
}
