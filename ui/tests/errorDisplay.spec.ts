import { describe, expect, it } from "vitest";
import { errorHeadline, errorDetails } from "src/util/errorDisplay";

// The regression this guards: WASM QR failures (e.g. the rqrr ECC error) are
// plain js Errors — no response.data, message/stack non-enumerable — and the
// old modal showed an empty headline and "{}" details for them.
describe("errorHeadline", () => {
  it("uses the message of a plain Error (WASM failure mode)", () => {
    const e = new Error("QR decode failed: could not correct errors in data (ECC) — the photo is likely blurry");
    expect(errorHeadline(e)).toContain("QR decode failed");
    expect(errorHeadline(e)).toContain("ECC");
  });

  it("uses the Go API error body message", () => {
    expect(errorHeadline({ message: "Request failed with status code 409", response: { data: { error: "conflict", message: "resource already exists" } } })).toBe(
      "resource already exists",
    );
  });

  it("maps legacy field errors", () => {
    expect(errorHeadline({ response: { data: { name: { code: "required", message: "cannot be empty" } } } })).toBe("Error on field 'name': cannot be empty");
  });

  it("falls back to the axios message when the body has no message", () => {
    expect(errorHeadline({ message: "Network Error", response: { data: {} } })).toBe("Network Error");
  });

  it("accepts plain strings and never returns empty", () => {
    expect(errorHeadline("boom")).toBe("boom");
    expect(errorHeadline(null)).toBe("Unexpected Error");
    expect(errorHeadline({})).toBe("Unexpected Error");
  });
});

describe("errorDetails", () => {
  it("shows stack or name+message for a plain Error, never '{}'", () => {
    const details = errorDetails(new Error("no QR code found"));
    expect(details).toContain("no QR code found");
    expect(details).not.toBe("{}");
  });

  it("shows status and body for API errors", () => {
    const details = errorDetails({ message: "Request failed", response: { status: 409, data: { error: "conflict" } } });
    expect(details).toContain("409");
    expect(details).toContain("conflict");
  });

  it("handles null and plain objects", () => {
    expect(errorDetails(null)).toBe("No details available");
    expect(errorDetails({ a: 1 })).toContain('"a": 1');
  });
});
