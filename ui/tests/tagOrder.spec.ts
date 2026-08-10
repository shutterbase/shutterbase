import { describe, it, expect } from "vitest";
import { compareTagOrder, exifKeywordAssignments, groupTagAssignments, tagCategory, tagLabel, toTagOrder } from "src/util/tagOrder";
import { ImageTagAssignment } from "src/types/api";

function assignment(id: string, assignmentType: string, tagType: string, name: string, order?: number | null): ImageTagAssignment {
  return { id, type: assignmentType, tag: { id: `tag-${id}`, name, type: tagType, order } };
}

describe("tagCategory", () => {
  it("maps inferred assignments to ai regardless of tag type", () => {
    expect(tagCategory(assignment("1", "inferred", "manual", "podium"))).toBe("ai");
  });
  it("maps default/template tags to template, others by tag type", () => {
    expect(tagCategory(assignment("1", "default", "default", "20240817"))).toBe("template");
    expect(tagCategory(assignment("2", "scheduled", "default", "quali"))).toBe("template");
    expect(tagCategory(assignment("3", "manual", "manual", "podium"))).toBe("manual");
    expect(tagCategory(assignment("4", "manual", "custom", "review"))).toBe("custom");
  });
});

describe("tagLabel", () => {
  it("prefers displayName, falls back to name when unset or empty", () => {
    expect(tagLabel({ name: "20240817", displayName: "Race day" })).toBe("Race day");
    expect(tagLabel({ name: "20240817", displayName: "" })).toBe("20240817");
    expect(tagLabel({ name: "20240817" })).toBe("20240817");
  });
});

describe("compareTagOrder", () => {
  it("sorts by order asc, ties and unset alphabetical, unset last", () => {
    const tags = [
      { id: "a", name: "zebra", type: "manual" },
      { id: "b", name: "bravo", type: "manual", order: 2 },
      { id: "c", name: "delta", type: "manual", order: 1 },
      { id: "d", name: "alpha", type: "manual" },
      { id: "e", name: "charlie", type: "manual", order: 2 },
    ];
    expect([...tags].sort(compareTagOrder).map((t) => t.name)).toEqual(["delta", "bravo", "charlie", "alpha", "zebra"]);
  });
});

describe("groupTagAssignments", () => {
  it("emits rows in fixed category order, each sorted, empty rows dropped", () => {
    const groups = groupTagAssignments([
      assignment("1", "manual", "custom", "note"),
      assignment("2", "inferred", "manual", "podium"),
      assignment("3", "default", "default", "zulu", 2),
      assignment("4", "default", "default", "alpha"),
      assignment("5", "default", "default", "mike", 1),
    ]);
    expect(groups.map((g) => g.category)).toEqual(["template", "custom", "ai"]);
    expect(groups[0].assignments.map((a) => a.tag.name)).toEqual(["mike", "zulu", "alpha"]);
  });
});

describe("exifKeywordAssignments", () => {
  it("mirrors the EXIF export: default/manual-type tags only, no 'internal', ranked by order", () => {
    const list = exifKeywordAssignments([
      assignment("1", "manual", "custom", "note"),
      assignment("2", "default", "default", "internal"),
      assignment("3", "manual", "manual", "podium", 2),
      assignment("4", "default", "default", "20240817", 1),
      assignment("5", "inferred", "manual", "crowd"), // inferred assignment on a manual tag IS exported
      assignment("6", "default", "template", "$DATE"),
    ]);
    expect(list.map((a) => a.tag.name)).toEqual(["20240817", "podium", "crowd"]);
  });
});

describe("toTagOrder", () => {
  it("parses positive integers and clamps everything else to 0", () => {
    expect(toTagOrder("3")).toBe(3);
    expect(toTagOrder(7)).toBe(7);
    expect(toTagOrder("")).toBe(0);
    expect(toTagOrder(undefined)).toBe(0);
    expect(toTagOrder(null)).toBe(0);
    expect(toTagOrder("-2")).toBe(0);
    expect(toTagOrder("abc")).toBe(0);
  });
});
