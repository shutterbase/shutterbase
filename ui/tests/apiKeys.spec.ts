import { describe, expect, it } from "vitest";
import { canManageApiKeys, keyStatus, sortApiKeys } from "src/util/apiKeys";
import { ApiKey } from "src/types/api";

function key(id: string, over: Partial<ApiKey> = {}): ApiKey {
  return {
    id,
    keyId: `kid-${id}`,
    name: id,
    userId: "u1",
    revoked: false,
    createdAt: "2026-07-01T10:00:00.000Z",
    updatedAt: "2026-07-01T10:00:00.000Z",
    ...over,
  };
}

// Mirrors the server's "admin or self" gate on POST/DELETE /api-keys, so the page
// never renders a Create/Revoke button that would come back 403.
describe("canManageApiKeys", () => {
  const me = { id: "u1", isAdmin: false };
  const admin = { id: "boss", isAdmin: true };

  it("lets a user manage their own keys", () => {
    expect(canManageApiKeys(me, "u1")).toBe(true);
  });

  it("refuses another user's keys", () => {
    expect(canManageApiKeys(me, "u2")).toBe(false);
  });

  it("lets a platform admin manage anyone's", () => {
    expect(canManageApiKeys(admin, "u1")).toBe(true);
    expect(canManageApiKeys(admin, "boss")).toBe(true);
  });

  it("refuses when logged out or without a target", () => {
    expect(canManageApiKeys(null, "u1")).toBe(false);
    expect(canManageApiKeys(me, "")).toBe(false);
  });
});

describe("sortApiKeys", () => {
  it("puts active keys before revoked ones", () => {
    const sorted = sortApiKeys([key("old-revoked", { revoked: true }), key("live")]);
    expect(sorted.map((k) => k.id)).toEqual(["live", "old-revoked"]);
  });

  it("orders newest first within a group", () => {
    const sorted = sortApiKeys([
      key("older", { createdAt: "2026-07-01T10:00:00.000Z" }),
      key("newest", { createdAt: "2026-07-20T10:00:00.000Z" }),
      key("middle", { createdAt: "2026-07-10T10:00:00.000Z" }),
    ]);
    expect(sorted.map((k) => k.id)).toEqual(["newest", "middle", "older"]);
  });

  it("a freshly revoked key drops below every live one, however new", () => {
    const sorted = sortApiKeys([key("revoked-today", { revoked: true, createdAt: "2026-07-25T10:00:00.000Z" }), key("live-ancient", { createdAt: "2020-01-01T10:00:00.000Z" })]);
    expect(sorted.map((k) => k.id)).toEqual(["live-ancient", "revoked-today"]);
  });

  it("does not mutate its input", () => {
    const input = [key("b", { createdAt: "2026-07-01T10:00:00.000Z" }), key("a", { createdAt: "2026-07-20T10:00:00.000Z" })];
    sortApiKeys(input);
    expect(input.map((k) => k.id)).toEqual(["b", "a"]);
  });
});

describe("keyStatus", () => {
  it("distinguishes a key that was never used from one in use", () => {
    expect(keyStatus(key("fresh"))).toBe("unused");
    expect(keyStatus(key("used", { lastUsedAt: "2026-07-20T10:00:00.000Z" }))).toBe("active");
  });

  it("revoked wins over everything", () => {
    expect(keyStatus(key("dead", { revoked: true, lastUsedAt: "2026-07-20T10:00:00.000Z" }))).toBe("revoked");
  });
});
