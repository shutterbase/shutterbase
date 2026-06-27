import { SORT_ORDER } from "src/components/image/sortOrder";
import type { ImageListParams } from "src/api/images";

// Pure mapping of UI filter/sort state onto the typed list contract (§4.3).
// Kept SFC- and store-free so it is trivially unit-testable.
export function buildImageListParams(input: {
  projectId: string;
  search?: string;
  tags?: { id: string }[];
  orientation?: string;
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
  if (input.orientation && input.orientation !== "neutral") {
    params.orientation = input.orientation as "portrait" | "landscape";
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
