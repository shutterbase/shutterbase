import { describe, it, expect, vi, beforeEach } from "vitest";

const { http } = vi.hoisted(() => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    patch: vi.fn(),
  },
}));

vi.mock("src/boot/axios", () => ({
  http,
  API_BASE: "/api/v1",
  websocketUrl: () => "ws://localhost/ws",
}));

import * as dev from "src/api/dev";

beforeEach(() => {
  vi.clearAllMocks();
  http.post.mockResolvedValue({ data: {} });
});

describe("api.dev — builds the right /dev/* requests", () => {
  it("login POSTs /dev/login with the role", async () => {
    await dev.login({ role: "projectEditor" });
    expect(http.post).toHaveBeenCalledWith("/dev/login", { role: "projectEditor" });
  });

  it("impersonate POSTs /dev/impersonate/:userId", async () => {
    await dev.impersonate("u-123");
    expect(http.post).toHaveBeenCalledWith("/dev/impersonate/u-123");
  });

  it("roleToggle POSTs /dev/role", async () => {
    await dev.roleToggle();
    expect(http.post).toHaveBeenCalledWith("/dev/role", { role: undefined });
  });

  it("timeOffset POSTs /dev/time-offset with drift + stale", async () => {
    await dev.timeOffset({ cameraId: "cam1", driftSeconds: 42, stale: true });
    expect(http.post).toHaveBeenCalledWith("/dev/time-offset", { cameraId: "cam1", driftSeconds: 42, stale: true });
  });

  it("images POSTs /dev/images and unwraps the result", async () => {
    http.post.mockResolvedValue({ data: { created: 2, imageIds: ["a", "b"] } });
    const res = await dev.images({ uploadId: "up1", count: 2 });
    expect(http.post).toHaveBeenCalledWith("/dev/images", { uploadId: "up1", count: 2 });
    expect(res.created).toBe(2);
  });

  it("infer POSTs /dev/infer/:imageId with optional tags", async () => {
    await dev.infer("img1", ["Podium"]);
    expect(http.post).toHaveBeenCalledWith("/dev/infer/img1", { tags: ["Podium"] });
    await dev.infer("img2");
    expect(http.post).toHaveBeenCalledWith("/dev/infer/img2", {});
  });

  it("syncTags / defaultTags / reseed hit their maintenance routes", async () => {
    await dev.syncTags();
    expect(http.post).toHaveBeenCalledWith("/dev/sync-tags");
    await dev.defaultTags("p1");
    expect(http.post).toHaveBeenCalledWith("/dev/default-tags", { projectId: "p1" });
    await dev.reseed();
    expect(http.post).toHaveBeenCalledWith("/dev/reseed");
  });

  it("clock POSTs /dev/clock for freeze and reset", async () => {
    await dev.clock({ at: "2030-01-02T03:04:05Z" });
    expect(http.post).toHaveBeenCalledWith("/dev/clock", { at: "2030-01-02T03:04:05Z" });
    await dev.clock({ reset: true });
    expect(http.post).toHaveBeenCalledWith("/dev/clock", { reset: true });
  });

  it("apiKey POSTs /dev/api-key and unwraps the token", async () => {
    http.post.mockResolvedValue({ data: { token: "k.s", keyId: "k" } });
    const res = await dev.apiKey("dl");
    expect(http.post).toHaveBeenCalledWith("/dev/api-key", { name: "dl" });
    expect(res.token).toBe("k.s");
  });
});
