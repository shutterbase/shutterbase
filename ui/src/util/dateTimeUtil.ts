import { DateTime } from "luxon";
import { TimeOffsetsResponse } from "src/types/pocketbase";

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

export function parseBackendTime(backendTime: string): Date {
  return new Date(Date.parse(backendTime));
}

export function timeOffsetUpToDate(timeOffset: TimeOffsetsResponse): boolean {
  const serverTime = parseBackendTime(timeOffset.serverTime);
  return DateTime.fromJSDate(serverTime) > DateTime.now().minus({ hours: 24 });
}
