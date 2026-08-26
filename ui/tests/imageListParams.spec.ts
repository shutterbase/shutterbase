import { describe, it, expect } from "vitest";
import { applyPersonPause, buildImageListParams } from "src/pages/image/imageListParams";
import { SORT_ORDER } from "src/components/image/sortOrder";

describe("buildImageListParams (UI state -> §4.3 list params)", () => {
  it("requires only projectId and defaults sort to capturedAtCorrected desc", () => {
    expect(buildImageListParams({ projectId: "p1" })).toEqual({
      projectId: "p1",
      sort: "capturedAtCorrected",
      order: "desc",
    });
  });

  it("maps search, AND-tags, orientation and pagination", () => {
    const params = buildImageListParams({
      projectId: "p1",
      search: "sunset",
      tags: [{ id: "t1" }, { id: "t2" }],
      orientation: "portrait",
      sortOrder: SORT_ORDER.OLDEST_FIRST,
      limit: 20,
      offset: 40,
    });
    expect(params).toEqual({
      projectId: "p1",
      search: "sunset",
      tagId: ["t1", "t2"],
      orientation: "portrait",
      sort: "capturedAtCorrected",
      order: "asc",
      limit: 20,
      offset: 40,
    });
  });

  it("drops the neutral orientation and empty search", () => {
    const params = buildImageListParams({ projectId: "p1", search: "", tags: [], excludeTags: [], orientation: "neutral" });
    expect(params.orientation).toBeUndefined();
    expect(params.search).toBeUndefined();
    expect(params.tagId).toBeUndefined();
    expect(params.excludeTagId).toBeUndefined();
  });

  it("maps exclude-tags independently of include-tags", () => {
    const params = buildImageListParams({ projectId: "p1", tags: [{ id: "t1" }], excludeTags: [{ id: "t2" }, { id: "t3" }] });
    expect(params.tagId).toEqual(["t1"]);
    expect(params.excludeTagId).toEqual(["t2", "t3"]);
    expect(buildImageListParams({ projectId: "p1", excludeTags: [{ id: "t2" }] }).tagId).toBeUndefined();
  });

  it("maps the implicit person filter and drops it when unset", () => {
    expect(buildImageListParams({ projectId: "p1", personRef: "person-7" }).personRef).toBe("person-7");
    expect(buildImageListParams({ projectId: "p1" }).personRef).toBeUndefined();
  });

  it("maps the implicit upload filter and drops it when unset", () => {
    expect(buildImageListParams({ projectId: "p1", uploadId: "u-1" }).uploadId).toBe("u-1");
    expect(buildImageListParams({ projectId: "p1" }).uploadId).toBeUndefined();
  });

  it("sends crossProject only together with a person filter", () => {
    expect(buildImageListParams({ projectId: "p1", personRef: "person-7", crossProject: true }).crossProject).toBe("true");
    expect(buildImageListParams({ projectId: "p1", personRef: "person-7" }).crossProject).toBeUndefined();
    expect(buildImageListParams({ projectId: "p1", crossProject: true }).crossProject).toBeUndefined();
  });

  it("maps the updated-sort orders to updatedAt", () => {
    expect(buildImageListParams({ projectId: "p", sortOrder: SORT_ORDER.MOST_RECENTLY_UPDATED })).toMatchObject({ sort: "updatedAt", order: "desc" });
    expect(buildImageListParams({ projectId: "p", sortOrder: SORT_ORDER.LEAST_RECENTLY_UPDATED })).toMatchObject({ sort: "updatedAt", order: "asc" });
  });

  it("serializes repeated tagId as tagId=a&tagId=b (axios indexes:null)", async () => {
    const axios = (await import("axios")).default;
    const params = buildImageListParams({ projectId: "p1", tags: [{ id: "a" }, { id: "b" }] });
    const qs = axios.getUri({ url: "/images", params, paramsSerializer: { indexes: null } });
    expect(qs).toContain("tagId=a");
    expect(qs).toContain("tagId=b");
    expect(qs).not.toContain("tagId[0]");
  });
});

describe("time-range params", () => {
  it("serializes inclusive from/to bounds as RFC3339", () => {
    const params = buildImageListParams({
      projectId: "p1",
      timeFrom: "2026-08-25T22:55:00.000Z",
      timeTo: "2026-08-25T23:10:00.000Z",
    });
    expect(params.from).toBe("2026-08-25T22:55:00.000Z");
    expect(params.to).toBe("2026-08-25T23:10:00.000Z");
  });

  it("omits unset bounds and passes an open-ended single side", () => {
    expect(buildImageListParams({ projectId: "p1" }).from).toBeUndefined();
    expect(buildImageListParams({ projectId: "p1" }).to).toBeUndefined();
    const openFrom = buildImageListParams({ projectId: "p1", timeFrom: "2026-08-25T22:55:00Z" });
    expect(openFrom.from).toBe("2026-08-25T22:55:00Z");
    expect(openFrom.to).toBeUndefined();
  });

  it("person-view pause suspends the time range with the other narrowing filters", () => {
    const paused = applyPersonPause({
      projectId: "p1",
      search: "x",
      tags: [{ id: "t1" }],
      excludeTags: [],
      orientation: "portrait",
      timeFrom: "2026-08-25T22:55:00Z",
      timeTo: "2026-08-25T23:10:00Z",
    });
    expect(paused.search).toBe("");
    expect(paused.timeFrom).toBeUndefined();
    expect(paused.timeTo).toBeUndefined();
  });
});
