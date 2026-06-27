import { describe, it, expect } from "vitest";
import { buildImageListParams } from "src/pages/image/imageListParams";
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
    const params = buildImageListParams({ projectId: "p1", search: "", tags: [], orientation: "neutral" });
    expect(params.orientation).toBeUndefined();
    expect(params.search).toBeUndefined();
    expect(params.tagId).toBeUndefined();
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
