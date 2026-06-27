// Extracted from ImagesHeader.vue so non-SFC modules (store, query logic, tests)
// can import it without pulling in a Vue single-file component.
export enum SORT_ORDER {
  LATEST_FIRST = "latestFirst",
  OLDEST_FIRST = "oldestFirst",
  MOST_RECENTLY_UPDATED = "mostRecentlyUpdated",
  LEAST_RECENTLY_UPDATED = "leastRecentlyUpdated",
}
