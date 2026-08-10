import { EmbeddedTag, ImageTagAssignment } from "src/types/api";

// Tag category rows for the image detail sidebar. "template" covers the
// official template-driven tags (tag type default/template), "ai" is any
// inferred assignment regardless of the tag it points at.
export type TagCategory = "template" | "manual" | "custom" | "ai";
export const TAG_CATEGORIES: TagCategory[] = ["template", "manual", "custom", "ai"];

// tagLabel is what the UI renders for a tag: the optional displayName when
// set, otherwise the name. Matching, exports, and hotkey bindings always use
// the name itself.
export function tagLabel(tag: { name: string; displayName?: string }): string {
  return tag.displayName || tag.name;
}

export function tagCategory(assignment: ImageTagAssignment): TagCategory {
  if (assignment.type === "inferred") return "ai";
  switch (assignment.tag.type) {
    case "custom":
      return "custom";
    case "manual":
      return "manual";
    default:
      return "template";
  }
}

// compareTagOrder ranks tags: lower order first, ties alphabetical; tags
// without an order come after all ranked ones, alphabetical. Mirrors the
// server-side EXIF keyword ordering (api/internal/exif/inject.go).
export function compareTagOrder(a: EmbeddedTag, b: EmbeddedTag): number {
  const ra = a.order ?? Number.POSITIVE_INFINITY;
  const rb = b.order ?? Number.POSITIVE_INFINITY;
  if (ra !== rb) return ra - rb;
  return a.name.localeCompare(b.name);
}

// toTagOrder coerces a form value (free-text input) into an order rank: a
// positive integer, or 0 for empty/invalid input (= "clear" on update, to be
// dropped on create).
export function toTagOrder(value: unknown): number {
  const n = typeof value === "string" ? parseInt(value, 10) : typeof value === "number" ? value : NaN;
  return Number.isInteger(n) && n > 0 ? n : 0;
}

// exifKeywordAssignments mirrors the EXIF keyword export
// (api/internal/exif/inject.go): only assignments of default/manual-type tags,
// never the internal management tag, ranked by the tags' order.
export function exifKeywordAssignments(assignments: ImageTagAssignment[]): ImageTagAssignment[] {
  return assignments.filter((a) => (a.tag.type === "default" || a.tag.type === "manual") && a.tag.name !== "internal").sort((a, b) => compareTagOrder(a.tag, b.tag));
}

export function groupTagAssignments(assignments: ImageTagAssignment[]): { category: TagCategory; assignments: ImageTagAssignment[] }[] {
  return TAG_CATEGORIES.map((category) => ({
    category,
    assignments: assignments.filter((a) => tagCategory(a) === category).sort((a, b) => compareTagOrder(a.tag, b.tag)),
  })).filter((group) => group.assignments.length > 0);
}
