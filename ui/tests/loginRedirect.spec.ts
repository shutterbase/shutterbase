import { describe, expect, it } from "vitest";
import { sanitizeRedirect } from "src/util/loginRedirect";

describe("sanitizeRedirect", () => {
  it("keeps same-app absolute paths", () => {
    expect(sanitizeRedirect("/images?image=abc123")).toBe("/images?image=abc123");
    expect(sanitizeRedirect("/projects")).toBe("/projects");
  });

  it("rejects everything else", () => {
    expect(sanitizeRedirect(undefined)).toBe("/");
    expect(sanitizeRedirect(null)).toBe("/");
    expect(sanitizeRedirect("")).toBe("/");
    expect(sanitizeRedirect("https://evil.example/phish")).toBe("/");
    expect(sanitizeRedirect("//evil.example/phish")).toBe("/");
    expect(sanitizeRedirect(["/a", "/b"])).toBe("/");
  });
});
