import { DateTime } from "luxon";

export function toDateTime(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.DATETIME_SHORT_WITH_SECONDS);
}

export function toDate(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.DATE_SHORT);
}

export function toTime(dateTime: string): string {
  return DateTime.fromISO(dateTime).setLocale("en-GB").toLocaleString(DateTime.TIME_WITH_SECONDS);
}
