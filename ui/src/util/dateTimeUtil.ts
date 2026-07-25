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

export function parseBackendTime(backendTime: string): Date {
  return new Date(Date.parse(backendTime));
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
