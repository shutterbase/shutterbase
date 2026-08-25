import { describe, it, expect } from "vitest";
import { applyPersonPause, buildImageListParams } from "src/pages/image/imageListParams";
import { SORT_ORDER } from "src/components/image/sortOrder";

const filtered = {
  projectId: "p1",
  search: "FSG",
  tags: [{ id: "t1" }, { id: "t2" }],
  excludeTags: [{ id: "t3" }],
  personRef: "person-a",
  crossProject: true,
  uploadId: "u1",
  orientation: "portrait",
  sortOrder: SORT_ORDER.LATEST_FIRST,
};

describe("applyPersonPause (person-view filter suspension)", () => {
  it("strips search, include/exclude tags and orientation", () => {
    expect(applyPersonPause(filtered)).toEqual({
      projectId: "p1",
      search: "",
      tags: [],
      excludeTags: [],
      personRef: "person-a",
      crossProject: true,
      uploadId: "u1",
      orientation: "neutral",
      sortOrder: SORT_ORDER.LATEST_FIRST,
    });
  });

  it("does not mutate the input", () => {
    const copy = structuredClone(filtered);
    applyPersonPause(filtered);
    expect(filtered).toEqual(copy);
  });

  it("keeps the upload context and person scope", () => {
    const paused = applyPersonPause(filtered);
    expect(paused.uploadId).toBe("u1");
    expect(paused.personRef).toBe("person-a");
    expect(paused.crossProject).toBe(true);
  });

  it("serializes to a person-only query (no search/tag/orientation params)", () => {
    const params = buildImageListParams({ ...applyPersonPause(filtered), limit: 20, offset: 0 });
    expect(params.personRef).toBe("person-a");
    expect(params.crossProject).toBe("true");
    expect(params.uploadId).toBe("u1");
    expect(params.search).toBeUndefined();
    expect(params.tagId).toBeUndefined();
    expect(params.excludeTagId).toBeUndefined();
    expect(params.orientation).toBeUndefined();
  });

  it("is a no-op on an already-narrow input", () => {
    const bare = { projectId: "p1", personRef: "person-a", orientation: "neutral" };
    expect(applyPersonPause(bare)).toEqual({ ...bare, search: "", tags: [], excludeTags: [] });
  });
});
