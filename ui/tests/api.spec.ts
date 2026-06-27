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

import * as images from "src/api/images";
import * as imageTags from "src/api/imageTags";
import * as imageTagAssignments from "src/api/imageTagAssignments";
import * as users from "src/api/users";
import * as auth from "src/api/auth";
import * as timeOffsets from "src/api/timeOffsets";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("api.images", () => {
  it("list GETs /images with repeated-tag serializer and params", async () => {
    http.get.mockResolvedValue({ data: { items: [], total: 0, limit: 100, offset: 0 } });
    const res = await images.list({ projectId: "p1", tagId: ["a", "b"], sort: "capturedAt", order: "asc" });
    expect(http.get).toHaveBeenCalledWith(
      "/images",
      expect.objectContaining({
        params: expect.objectContaining({ projectId: "p1", tagId: ["a", "b"], sort: "capturedAt", order: "asc" }),
        paramsSerializer: { indexes: null },
      })
    );
    expect(res.total).toBe(0);
  });

  it("create POSTs /images and unwraps data", async () => {
    http.post.mockResolvedValue({ data: { id: "img1" } });
    const img = await images.create({ fileName: "a.jpg", storageId: "ab/x.jpg", size: 1, cameraId: "c", uploadId: "u", projectId: "p" });
    expect(http.post).toHaveBeenCalledWith("/images", expect.objectContaining({ storageId: "ab/x.jpg", projectId: "p" }));
    expect(img.id).toBe("img1");
  });

  it("remove DELETEs /images/:id", async () => {
    http.delete.mockResolvedValue({});
    await images.remove("img1");
    expect(http.delete).toHaveBeenCalledWith("/images/img1");
  });

  it("propagates request errors", async () => {
    http.get.mockRejectedValue(new Error("boom"));
    await expect(images.get("x")).rejects.toThrow("boom");
  });
});

describe("api.imageTags", () => {
  it("create maps projectId into the body", async () => {
    http.post.mockResolvedValue({ data: { id: "t1" } });
    await imageTags.create({ name: "review", type: "custom", projectId: "p1" });
    expect(http.post).toHaveBeenCalledWith("/image-tags", expect.objectContaining({ name: "review", type: "custom", projectId: "p1" }));
  });
});

describe("api.imageTagAssignments", () => {
  it("create POSTs the idempotent assignment body", async () => {
    http.post.mockResolvedValue({ data: { id: "a1", type: "manual", tag: { id: "t1" } } });
    const a = await imageTagAssignments.create({ imageId: "i1", imageTagId: "t1", type: "manual" });
    expect(http.post).toHaveBeenCalledWith("/image-tag-assignments", { imageId: "i1", imageTagId: "t1", type: "manual" });
    expect(a.id).toBe("a1");
  });
});

describe("api.auth", () => {
  it("login POSTs /auth/login with identifier+password", async () => {
    http.post.mockResolvedValue({ data: { id: "u1" } });
    await auth.login({ identifier: "max@x.de", password: "pw" });
    expect(http.post).toHaveBeenCalledWith("/auth/login", { identifier: "max@x.de", password: "pw" });
  });

  it("me GETs /users/me", async () => {
    http.get.mockResolvedValue({ data: { id: "u1" } });
    const u = await auth.me();
    expect(http.get).toHaveBeenCalledWith("/users/me");
    expect(u.id).toBe("u1");
  });

  it("stopImpersonate DELETEs /auth/impersonate", async () => {
    http.delete.mockResolvedValue({ data: { id: "real" } });
    await auth.stopImpersonate();
    expect(http.delete).toHaveBeenCalledWith("/auth/impersonate");
  });
});

describe("api.users", () => {
  it("setActiveProject PATCHes /users/me/active-project", async () => {
    http.patch.mockResolvedValue({});
    await users.setActiveProject("p9");
    expect(http.patch).toHaveBeenCalledWith("/users/me/active-project", { projectId: "p9" });
  });
});

describe("api.timeOffsets", () => {
  it("create POSTs /time-offsets without a client-computed offset", async () => {
    http.post.mockResolvedValue({ data: { id: "to1" } });
    await timeOffsets.create({ cameraId: "c1", serverTime: "2026-01-01T00:00:00Z", cameraTime: "2026-01-01T00:00:05Z" });
    const body = http.post.mock.calls[0][1];
    expect(http.post.mock.calls[0][0]).toBe("/time-offsets");
    expect(body).not.toHaveProperty("timeOffset");
    expect(body).toMatchObject({ cameraId: "c1" });
  });
});
