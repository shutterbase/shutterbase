import { describe, it, expect } from "vitest";
import { extractKeyFacts, formatExifDateTime, parseExifDateTime, pickTag, type ExifGroups } from "src/util/exifInspect";

const sample: ExifGroups = {
  ExifTool: { ExifToolVersion: 12.65 },
  IFD0: { Make: "Canon", Model: "Canon EOS R5", Artist: "Max Mustermann", Copyright: "FSG 2026" },
  ExifIFD: {
    DateTimeOriginal: "2026:08:09 16:03:12+02:00",
    CreateDate: "2026:08:09 14:01:09+02:00",
    ExposureTime: "1/1600",
    FNumber: 2.8,
    ISO: 400,
    FocalLength: "70.0 mm",
    LensModel: "RF70-200mm F2.8 L IS USM",
    SerialNumber: "123456789",
  },
  IPTC: { Keywords: ["FSG", "autocross", "by_mm"] },
  File: { ImageWidth: 8192, ImageHeight: 5464, MIMEType: "image/jpeg" },
};

describe("pickTag", () => {
  it("finds a tag regardless of its group", () => {
    expect(pickTag(sample, ["LensModel"])).toBe("RF70-200mm F2.8 L IS USM");
  });

  it("respects name priority order", () => {
    expect(pickTag(sample, ["Model", "Make"])).toBe("Canon EOS R5");
  });

  it("returns undefined when nothing matches", () => {
    expect(pickTag(sample, ["NoSuchTag"])).toBeUndefined();
  });
});

describe("parseExifDateTime / formatExifDateTime", () => {
  it("parses EXIF datetimes with offset and keeps the wall clock", () => {
    expect(formatExifDateTime("2026:08:09 16:03:12+02:00")).toBe("09.08.2026 16:03:12");
  });

  it("parses EXIF datetimes without offset", () => {
    expect(formatExifDateTime("2026:08:09 16:03:12")).toBe("09.08.2026 16:03:12");
  });

  it("passes through unparsable strings raw", () => {
    expect(formatExifDateTime("0000:00:00 00:00:00")).toBe("0000:00:00 00:00:00");
    expect(parseExifDateTime("garbage")).toBeNull();
  });

  it("is undefined for empty values", () => {
    expect(formatExifDateTime(undefined)).toBeUndefined();
    expect(formatExifDateTime("")).toBeUndefined();
  });
});

describe("extractKeyFacts", () => {
  const facts = extractKeyFacts(sample);

  it("extracts camera without duplicating the make", () => {
    expect(facts.camera).toBe("Canon EOS R5");
  });

  it("extracts lens, serial, exposure basics", () => {
    expect(facts.lens).toBe("RF70-200mm F2.8 L IS USM");
    expect(facts.bodySerial).toBe("123456789");
    expect(facts.exposureTime).toBe("1/1600");
    expect(facts.aperture).toBe("2.8");
    expect(facts.iso).toBe("400");
    expect(facts.focalLength).toBe("70.0 mm");
    expect(facts.dimensions).toBe("8192 × 5464");
  });

  it("reports original vs corrected capture time with the applied shift", () => {
    expect(facts.originalCaptureTime).toBe("09.08.2026 14:01:09");
    expect(facts.correctedCaptureTime).toBe("09.08.2026 16:03:12");
    expect(facts.timeShift).toBe("+2h 02m 03s");
  });

  it("reports no shift when both times match", () => {
    const same = extractKeyFacts({
      ExifIFD: { DateTimeOriginal: "2026:08:09 14:01:09", CreateDate: "2026:08:09 14:01:09" },
    });
    expect(same.timeShift).toBeUndefined();
  });

  it("collects keywords from arrays and splits string keyword lists", () => {
    expect(facts.keywords).toEqual(["FSG", "autocross", "by_mm"]);
    expect(extractKeyFacts({ IPTC: { Keywords: "a; b; c" } }).keywords).toEqual(["a", "b", "c"]);
    expect(extractKeyFacts({}).keywords).toEqual([]);
  });

  it("prepends make when the model does not carry it", () => {
    expect(extractKeyFacts({ IFD0: { Make: "NIKON CORPORATION", Model: "Z 9" } }).camera).toBe("NIKON CORPORATION Z 9");
  });
});
