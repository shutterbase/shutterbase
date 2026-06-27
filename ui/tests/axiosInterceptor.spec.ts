import { describe, it, expect, vi } from "vitest";

// quasar/wrappers pulls browser-only internals; the boot wrapper is just identity.
vi.mock("quasar/wrappers", () => ({ boot: (fn: any) => fn }));

import { http } from "src/boot/axios";

function rejectedHandler() {
  return (http.interceptors.response as any).handlers[0].rejected as (e: any) => Promise<any>;
}

function stubLocation(pathname: string) {
  const assign = vi.fn();
  Object.defineProperty(window, "location", { value: { pathname, assign }, writable: true });
  return assign;
}

describe("axios 401 interceptor", () => {
  it("redirects to /login on a 401 when not already on the login page", async () => {
    const assign = stubLocation("/images");
    await expect(rejectedHandler()({ response: { status: 401 } })).rejects.toBeDefined();
    expect(assign).toHaveBeenCalledWith("/login");
  });

  it("does not redirect when already on /login (no loop)", async () => {
    const assign = stubLocation("/login");
    await expect(rejectedHandler()({ response: { status: 401 } })).rejects.toBeDefined();
    expect(assign).not.toHaveBeenCalled();
  });

  it("passes non-401 errors through untouched", async () => {
    const assign = stubLocation("/images");
    await expect(rejectedHandler()({ response: { status: 500 } })).rejects.toBeDefined();
    expect(assign).not.toHaveBeenCalled();
  });
});
