import { SORT_ORDER } from "src/components/image/sortOrder";
import type { ImageListParams } from "src/api/images";

type ImageFilterInput = Parameters<typeof buildImageListParams>[0];

// Person-view pause: while browsing one person's photos the other narrowing
// criteria (search text, include/exclude tags, aspect ratio, time range) are
// suspended so a face click always yields the full gallery. Upload context
// passes through.
export function applyPersonPause(input: ImageFilterInput): ImageFilterInput {
  return { ...input, search: "", tags: [], excludeTags: [], orientation: "neutral", timeFrom: undefined, timeTo: undefined };
}

// Pure mapping of UI filter/sort state onto the typed list contract (§4.3).
// Kept SFC- and store-free so it is trivially unit-testable.
export function buildImageListParams(input: {
  projectId: string;
  search?: string;
  tags?: { id: string }[];
  excludeTags?: { id: string }[];
  personRef?: string;
  crossProject?: boolean;
  uploadId?: string;
  orientation?: string;
  timeFrom?: string;
  timeTo?: string;
  sortOrder?: SORT_ORDER;
  limit?: number;
  offset?: number;
}): ImageListParams {
  const params: ImageListParams = { projectId: input.projectId };

  if (input.search) {
    params.search = input.search;
  }
  if (input.tags && input.tags.length > 0) {
    params.tagId = input.tags.map((t) => t.id); // repeated -> AND
  }
  if (input.excludeTags && input.excludeTags.length > 0) {
    params.excludeTagId = input.excludeTags.map((t) => t.id); // repeated -> NOT any
  }
  if (input.personRef) {
    params.personRef = input.personRef;
    // the one exception to the hard project filter — meaningless without a person
    if (input.crossProject) {
      params.crossProject = "true";
    }
  }
  if (input.uploadId) {
    params.uploadId = input.uploadId;
  }
  if (input.orientation && input.orientation !== "neutral") {
    params.orientation = input.orientation as "portrait" | "landscape";
  }
  if (input.timeFrom) {
    params.from = input.timeFrom; // inclusive RFC3339 bound on capturedAtCorrected
  }
  if (input.timeTo) {
    params.to = input.timeTo;
  }

  switch (input.sortOrder) {
    case SORT_ORDER.OLDEST_FIRST:
      params.sort = "capturedAtCorrected";
      params.order = "asc";
      break;
    case SORT_ORDER.MOST_RECENTLY_UPDATED:
      params.sort = "updatedAt";
      params.order = "desc";
      break;
    case SORT_ORDER.LEAST_RECENTLY_UPDATED:
      params.sort = "updatedAt";
      params.order = "asc";
      break;
    case SORT_ORDER.LATEST_FIRST:
    default:
      params.sort = "capturedAtCorrected";
      params.order = "desc";
      break;
  }

  if (input.limit !== undefined) params.limit = input.limit;
  if (input.offset !== undefined) params.offset = input.offset;

  return params;
}
