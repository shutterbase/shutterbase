import { DateTime } from "luxon";
import { appliedTimeOffset } from "src/util/dateTimeUtil";

// exiftool -j -a -u -g1 output: one object whose keys are family-1 groups
// (IFD0, ExifIFD, Canon, File, Composite, ...), each mapping tag name → value.
export type ExifGroups = Record<string, Record<string, unknown>>;

/** First value found for any of `names`, groups in exiftool order, names by priority. */
export function pickTag(meta: ExifGroups, names: string[]): unknown {
  for (const name of names) {
    for (const group of Object.values(meta)) {
      if (group && typeof group === "object" && name in group) {
        return group[name];
      }
    }
  }
  return undefined;
}

/** EXIF datetime ("2026:08:09 14:03:12.34+02:00") → DateTime keeping the string's zone. */
export function parseExifDateTime(value: unknown): DateTime | null {
  if (typeof value !== "string") return null;
  const iso = value.replace(/^(\d{4}):(\d{2}):(\d{2}) /, "$1-$2-$3T");
  const dt = DateTime.fromISO(iso, { setZone: true });
  return dt.isValid ? dt : null;
}

/** German-format EXIF datetime as the camera wall clock; raw string when unparsable. */
export function formatExifDateTime(value: unknown): string | undefined {
  if (value === undefined || value === null || value === "") return undefined;
  const dt = parseExifDateTime(value);
  return dt ? dt.toFormat("dd.LL.yyyy HH:mm:ss") : String(value);
}

export interface ExifKeyFacts {
  camera?: string;
  bodySerial?: string;
  lens?: string;
  exposureTime?: string;
  aperture?: string;
  iso?: string;
  focalLength?: string;
  dimensions?: string;
  originalCaptureTime?: string;
  correctedCaptureTime?: string;
  /** Signed shift corrected−original (e.g. "+1h 02m 03s"), only when both parse and differ. */
  timeShift?: string;
  artist?: string;
  copyright?: string;
  keywords: string[];
}

const str = (v: unknown): string | undefined => (v === undefined || v === null || v === "" ? undefined : String(v));

/**
 * The headline facts of an inspected image. "Corrected" capture time is
 * DateTimeOriginal (the field Shutterbase rewrites on export); "original" is
 * CreateDate, which export leaves untouched — on an unexported photo both carry
 * the camera time and no shift is reported.
 */
export function extractKeyFacts(meta: ExifGroups): ExifKeyFacts {
  const make = str(pickTag(meta, ["Make"]));
  const model = str(pickTag(meta, ["Model", "CameraModelName"]));
  const camera = model && make && !model.startsWith(make) ? `${make} ${model}` : (model ?? make);

  const width = str(pickTag(meta, ["ImageWidth", "ExifImageWidth"]));
  const height = str(pickTag(meta, ["ImageHeight", "ExifImageHeight"]));

  const corrected = pickTag(meta, ["DateTimeOriginal"]);
  const original = pickTag(meta, ["CreateDate", "DateTimeDigitized"]);
  const correctedDt = parseExifDateTime(corrected);
  const originalDt = parseExifDateTime(original);
  let timeShift: string | undefined;
  if (correctedDt && originalDt && !correctedDt.equals(originalDt)) {
    timeShift = appliedTimeOffset(originalDt.toISO() as string, correctedDt.toISO() as string);
  }

  const rawKeywords = pickTag(meta, ["Keywords", "Subject", "XPKeywords"]);
  const keywords = Array.isArray(rawKeywords) ? rawKeywords.map(String) : typeof rawKeywords === "string" && rawKeywords !== "" ? rawKeywords.split(/[;,] ?/) : [];

  return {
    camera,
    bodySerial: str(pickTag(meta, ["SerialNumber", "BodySerialNumber", "InternalSerialNumber"])),
    lens: str(pickTag(meta, ["LensModel", "LensID", "Lens", "LensType"])),
    exposureTime: str(pickTag(meta, ["ExposureTime", "ShutterSpeed", "ShutterSpeedValue"])),
    aperture: str(pickTag(meta, ["FNumber", "Aperture", "ApertureValue"])),
    iso: str(pickTag(meta, ["ISO"])),
    focalLength: str(pickTag(meta, ["FocalLength"])),
    dimensions: width && height ? `${width} × ${height}` : undefined,
    originalCaptureTime: formatExifDateTime(original),
    correctedCaptureTime: formatExifDateTime(corrected),
    timeShift,
    artist: str(pickTag(meta, ["Artist", "By-line"])),
    copyright: str(pickTag(meta, ["Copyright", "CopyrightNotice"])),
    keywords,
  };
}
