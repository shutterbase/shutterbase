import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { api } from "src/api";
import type { TagFacetsResponse } from "src/api/images";
import { ImageTag } from "src/types/api";
import { ImageWithTagsType } from "src/types/custom";
import { applyPersonPause, buildImageListParams } from "src/pages/image/imageListParams";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { canEditImageTag } from "src/pages/upload/uploadUtil";
import { isReviewerOnlyTag } from "src/util/uploadReview";
import { tagLabel } from "src/util/tagOrder";

export { buildImageListParams };

export enum DisplayMode {
  GRID = "grid",
  DETAIL = "detail",
}

const PAGE_SIZE = 20;

export const { activeProject, preferredImageSortOrder, tagStack } = storeToRefs(useUserStore());

export const showUnexpectedErrorMessage = ref(false);
export const unexpectedError = ref(null);

export const taggingDialogVisible = ref(false);

export const images = ref<ImageWithTagsType[]>([]);

export const imageIndex = ref(-1);
export const imageIndices = ref<number[]>([]);
export const multiselectStart = ref<number | null>(null);
export const multiselectEnd = ref<number | null>(null);

export const totalImageCount = ref(0);
const page = ref(1);
export const loading = ref(false);
export const filtered = ref(false);

export const selectedImageIndex = ref(-1);

export const searchText = ref("");
export function updateSearchText(text: string) {
  searchText.value = text;
}

export const filterTags = ref<ImageTag[]>([]);
export const excludeFilterTags = ref<ImageTag[]>([]);
export function updateFilterTags(filter: { include: ImageTag[]; exclude: ImageTag[] }) {
  // no-op guard: the header re-emits on setFilteredTags (grid ⇄ detail resync);
  // swapping in equal-but-new arrays would trigger the reload watcher and
  // reset a deeply scrolled grid to page 1.
  const same = (a: ImageTag[], b: ImageTag[]) => a.length === b.length && a.every((tag, i) => tag.id === b[i].id);
  if (same(filter.include, filterTags.value) && same(filter.exclude, excludeFilterTags.value)) return;
  filterTags.value = filter.include;
  excludeFilterTags.value = filter.exclude;
}

export const aspectRatioFilter = ref("neutral");
export function updateAspectRatioFilter(aspectRatioState: string) {
  aspectRatioFilter.value = aspectRatioState;
}

// The ImagesHeader owns these controls and remounts clean, so a fresh Images
// mount must reset them too — a value surviving here filters the grid
// invisibly (a sticky portrait filter once shrank a 38-photo person view to 5).
export function resetTransientFilters() {
  if (!searchText.value && filterTags.value.length === 0 && excludeFilterTags.value.length === 0 && aspectRatioFilter.value === "neutral") return;
  invalidateGridSnapshot(); // the snapshot was taken under the filters being cleared
  searchText.value = "";
  filterTags.value = [];
  excludeFilterTags.value = [];
  aspectRatioFilter.value = "neutral";
}

// Implicit person filter: set by clicking a face box in the detail view,
// cleared via the chip above the grid. No picker UI — the face IS the picker.
// The value is driven by the route query (?person=) so the browser history
// walks through filter states; Images.vue owns the sync.
export const personFilter = ref<string | null>(null);

// Cross-project scope for the person filter — the grid's ONE exception to the
// hard project filter. Route-driven like the filter itself (?personScope=all).
export const personCrossProject = ref(false);

// Person-view pause: while a person filter is active, the other narrowing
// filters (search/tags/orientation) can be suspended so the face click always
// yields the full gallery. The "Filters" pill toggles it; it re-arms whenever
// the person filter engages anew or is cleared (Images.vue / jumpToImage).
export const personFiltersPaused = ref(true);

// Implicit upload-batch filter: set by "view images" links on an upload or its
// kanban card, cleared via the chip above the grid. Route-driven (?upload=)
// like the person filter; Images.vue owns the sync.
export const uploadFilter = ref<string | null>(null);

// Semantic "ask" filter: the AI server ranks the project's images by their
// description for a free-text query; the grid shows that set under its
// normal sort, combinable with every other filter. Route-driven (?ask=).
export const askFilter = ref<string | null>(null);

// --- grid snapshot -----------------------------------------------------------
// Applying the person filter replaces the loaded (possibly deeply scrolled)
// grid. A snapshot of that state lets "clear filter" / browser-back land on
// the exact position instead of page 1.
// ponytail: single snapshot — any other filter/sort change invalidates it.

interface GridSnapshot {
  images: ImageWithTagsType[];
  page: number;
  total: number;
  imageIndex: number;
  scrollY: number;
}

let gridSnapshot: GridSnapshot | null = null;

export function snapshotGrid() {
  gridSnapshot = {
    images: images.value,
    page: page.value,
    total: totalImageCount.value,
    imageIndex: imageIndex.value,
    scrollY: window.scrollY,
  };
}

export function invalidateGridSnapshot() {
  gridSnapshot = null;
}

// restoreGridSnapshot puts the saved grid back and returns the scroll offset
// to restore, or null when there is nothing to restore.
export function restoreGridSnapshot(): number | null {
  if (!gridSnapshot) return null;
  const snap = gridSnapshot;
  gridSnapshot = null;
  images.value = snap.images;
  page.value = snap.page;
  totalImageCount.value = snap.total;
  imageIndex.value = snap.imageIndex;
  return snap.scrollY;
}

export async function triggerInfiniteScroll() {
  if (totalImageCount.value > 0 && images.value.length < totalImageCount.value) {
    loadImages(false);
  }
}

// latest-wins guard: every call gets an id; only the newest call's response is
// allowed to mutate state, so a filter/search/sort change mid-flight is never
// dropped and a stale in-flight response is discarded.
let requestId = 0;

// shared filter/sort state → buildImageListParams input; one source of truth
// for the list, the facets and the deep-link position queries
function currentFilterInput() {
  const input = {
    projectId: activeProject.value.id,
    search: searchText.value,
    tags: filterTags.value,
    excludeTags: excludeFilterTags.value,
    personRef: personFilter.value ?? undefined,
    crossProject: personCrossProject.value,
    uploadId: uploadFilter.value ?? undefined,
    ask: askFilter.value ?? undefined,
    orientation: aspectRatioFilter.value,
    sortOrder: preferredImageSortOrder.value,
  };
  return personFilter.value && personFiltersPaused.value ? applyPersonPause(input) : input;
}

export async function loadImages(reload: boolean) {
  // no active project (fresh account following a permalink) — the query is
  // meaningless and would 400; jumpToImage switches the project first
  if (!activeProject.value?.id) return;
  // pagination guard only: don't stack "load more" requests, but a reload
  // (new filter/search/sort) must always issue a fresh query.
  if (!reload && loading.value) return;
  const myRequestId = ++requestId;
  loading.value = true;
  try {
    if (reload) {
      page.value = 1;
      soloImage.value = false;
    }

    filtered.value =
      !!searchText.value ||
      filterTags.value.length > 0 ||
      excludeFilterTags.value.length > 0 ||
      aspectRatioFilter.value !== "neutral" ||
      !!personFilter.value ||
      !!uploadFilter.value ||
      !!askFilter.value;

    const params = buildImageListParams({
      ...currentFilterInput(),
      limit: PAGE_SIZE,
      offset: (page.value - 1) * PAGE_SIZE,
    });

    const result = await api.images.list(params);
    if (myRequestId !== requestId) return; // superseded by a newer call, discard
    totalImageCount.value = result.total;
    page.value++;

    if (reload) {
      images.value = [];
    }
    images.value.push(...result.items);
  } catch (error: any) {
    if (myRequestId !== requestId) return;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    if (myRequestId === requestId) loading.value = false;
  }
}

// --- deep link / permalink resolution ------------------------------------------
// A shared /images?image=<id> URL must open for a recipient whose loaded pages,
// active project or filters don't contain the image. jumpToImage resolves it:
// fetch the image (403/404 → toast), switch the active project if the link
// points into another of the viewer's projects, ask the API where the image
// sits under the current sort, and extend the loaded window up to it. When the
// position is unknown (filtered out, or deeper than the server's scan bound)
// the detail opens solo: the image alone, without grid context.
// ponytail: solo beyond 2000 — add backward pagination if deep links into old
// archives ever matter.

const JUMP_CONTEXT_MAX = 2000;
const LIST_MAX_LIMIT = 500; // server-side limit clamp

// solo = the images array holds only the deep-linked image; leaving the detail
// view (or any reload) restores a real grid.
export const soloImage = ref(false);

export type JumpResult = { status: "jumped" | "solo" | "unavailable"; projectSwitched: boolean };

export async function jumpToImage(imageId: string): Promise<JumpResult> {
  let img: ImageWithTagsType;
  try {
    img = await api.images.get(imageId);
  } catch (error: any) {
    showNotificationToast({
      headline: error?.response?.status === 403 ? "You don't have access to this image's project" : "This image no longer exists",
      type: "warning",
    });
    return { status: "unavailable", projectSwitched: false };
  }

  let projectSwitched = false;
  if (img.project && img.project.id !== activeProject.value?.id) {
    // the link points into another of the viewer's projects (non-members got
    // the 403 above) — switch, dropping filters that belonged to the old one
    useUserStore().setProject(img.project);
    personFilter.value = null;
    personCrossProject.value = false;
    uploadFilter.value = null;
    askFilter.value = null;
    personFiltersPaused.value = true;
    resetTransientFilters();
    projectSwitched = true;
    await loadImages(true);
    if (images.value.some((i) => i.id === imageId)) return { status: "jumped", projectSwitched };
  }

  let position = -1;
  try {
    position = await api.images.position({ ...buildImageListParams(currentFilterInput()), imageId });
  } catch {
    // fall through to solo
  }
  if (position >= 0 && position < JUMP_CONTEXT_MAX) {
    await loadImagesUntil(position + 1);
    if (images.value.some((i) => i.id === imageId)) return { status: "jumped", projectSwitched };
  }

  images.value = [img];
  totalImageCount.value = 1;
  page.value = 1;
  soloImage.value = true;
  return { status: "solo", projectSwitched };
}

// loadImagesUntil extends the loaded window to at least count images (server
// pages cap at LIST_MAX_LIMIT per request) so a deep-linked image gets its real
// grid context around it. Leaves the page counter consistent for infinite scroll.
async function loadImagesUntil(count: number) {
  const target = Math.min(Math.ceil(count / PAGE_SIZE) * PAGE_SIZE, JUMP_CONTEXT_MAX);
  if (images.value.length >= target) return;
  const myRequestId = ++requestId;
  loading.value = true;
  try {
    while (images.value.length < target) {
      const params = buildImageListParams({
        ...currentFilterInput(),
        limit: Math.min(target - images.value.length, LIST_MAX_LIMIT),
        offset: images.value.length,
      });
      const result = await api.images.list(params);
      if (myRequestId !== requestId) return;
      totalImageCount.value = result.total;
      images.value.push(...result.items);
      if (result.items.length === 0) break;
    }
    page.value = Math.floor(images.value.length / PAGE_SIZE) + 1;
  } catch (error: any) {
    if (myRequestId !== requestId) return;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    if (myRequestId === requestId) loading.value = false;
  }
}

// --- tag facets ---------------------------------------------------------------
// Per-tag counts under the CURRENT filter, feeding the tag popover: hide tags
// that would empty the view, show what each +/− filter would leave. Fetched on
// popover open (force) and on selection changes; the key memo skips refetches
// when nothing filter-relevant changed (e.g. grid ⇄ detail resync re-emits).

export const tagFacets = ref<TagFacetsResponse | null>(null);
let lastFacetsKey = "";

export async function loadTagFacets(force = false) {
  const params = buildImageListParams(currentFilterInput());
  const key = JSON.stringify(params);
  if (!force && key === lastFacetsKey && tagFacets.value) return;
  lastFacetsKey = key;
  try {
    tagFacets.value = await api.images.tagFacets(params);
  } catch {
    // facets are decoration — the popover degrades to the plain tag list
    tagFacets.value = null;
    lastFacetsKey = "";
  }
}

export async function addImageTag(image: ImageWithTagsType, tag: ImageTag) {
  // Every tag assignment (dialog, hotkey, repeat-last) funnels through here, so
  // this is the one place the review freeze has to be honored client-side.
  if (!canEditImageTag(image, tag)) {
    showNotificationToast({
      headline: isReviewerOnlyTag(tag.name) ? `Only a project admin can set '${tagLabel(tag)}'` : `'${tagLabel(tag)}' is frozen while the upload is in review`,
      type: "warning",
    });
    return;
  }
  const applyTag = async (image: ImageWithTagsType, tag: ImageTag) => {
    const assignment = await api.imageTagAssignments.create({
      imageId: image.id,
      imageTagId: tag.id,
      type: "manual",
    });
    const editedImageIndex = images.value.findIndex((i) => i.id === image.id);
    images.value[editedImageIndex].tags.push(assignment);
    images.value[editedImageIndex].updatedAt = new Date().toISOString();
  };

  try {
    const imageApplyList: ImageWithTagsType[] = [];
    for (const idx of imageIndices.value) {
      const i = images.value[idx];
      if (!i.tags.some((a) => a.tag.id === tag.id)) {
        imageApplyList.push(images.value[idx]);
      }
    }
    if (image !== null && !imageApplyList.includes(image)) {
      imageApplyList.push(image);
    }

    for (const img of imageApplyList) {
      await applyTag(img, tag);
    }
    emitter.emit("reset-tagging-dialog");
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- AI detection state -----------------------------------------------------

// global queue positions for the loaded pending images (imageId -> 1-based)
export const aiPositions = ref<Record<string, number>>({});
export const aiQueueTotal = ref(0);

// refreshAiPositions re-reads status + position for every loaded image that is
// still in flight. One batched call; grid badges and the sidebar read the map.
export async function refreshAiPositions() {
  const inFlight = images.value.filter((i) => i.aiStatus === "pending" || i.aiStatus === "processing").map((i) => i.id);
  if (inFlight.length === 0) {
    aiPositions.value = {};
    return;
  }
  try {
    const status = await api.ai.queueStatus(activeProject.value.id, inFlight.slice(0, 200));
    const map: Record<string, number> = {};
    for (const item of status.items) {
      if (item.position) map[item.imageId] = item.position;
      const img = images.value.find((i) => i.id === item.imageId);
      if (img) img.aiStatus = item.status ?? null;
    }
    aiPositions.value = map;
    aiQueueTotal.value = status.queueTotal;
  } catch {
    // positions are decoration — never surface an error for them
  }
}

// applyAiEvent patches a websocket image/changed event into the loaded list.
// A "done" image is refetched so its fresh inferred tags appear live.
export function applyAiEvent(data: { projectId: string; imageId: string; status: string }) {
  if (data.projectId !== activeProject.value?.id) return;
  const img = images.value.find((i) => i.id === data.imageId);
  if (!img) return;
  img.aiStatus = (data.status || null) as ImageWithTagsType["aiStatus"];
  if (data.status === "done") {
    api.images
      .get(data.imageId)
      .then((fresh) => {
        const idx = images.value.findIndex((i) => i.id === data.imageId);
        if (idx !== -1) images.value[idx] = fresh;
      })
      .catch(() => undefined);
  }
}

// rerunAiSelection re-queues the current selection (multi-select + current).
export async function rerunAiSelection() {
  const targets = new Set(imageIndices.value);
  if (imageIndex.value !== -1) targets.add(imageIndex.value);
  const ids = [...targets].map((i) => images.value[i]?.id).filter((id): id is string => !!id);
  if (ids.length === 0) return;
  try {
    const queued = await api.ai.rerunBatch(activeProject.value.id, ids);
    showNotificationToast({ headline: `AI detection queued for ${queued} image${queued === 1 ? "" : "s"}`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// Hotkey handlers: bound to their action ids by Images.vue (useHotkeyAction),
// so they are only active while the images page is mounted. Context gating in
// the dispatcher keeps them silent while the tagging dialog is open.
export function nextImage() {
  if (imageIndex.value < images.value.length - 1) {
    imageIndex.value++;
  }
  if (imageIndex.value >= images.value.length - 4) {
    triggerInfiniteScroll();
  }
  emitter.emit("update-image-grid-scroll-position");
}

export function previousImage() {
  if (imageIndex.value > 0) {
    imageIndex.value--;
  }
  emitter.emit("update-image-grid-scroll-position");
}

export function previousRow() {
  if (imageIndex.value - 4 >= 0) {
    imageIndex.value -= 4;
  } else {
    imageIndex.value = 0;
  }
  emitter.emit("update-image-grid-scroll-position");
}

export function nextRow() {
  if (imageIndex.value + 4 < images.value.length) {
    imageIndex.value += 4;
  } else {
    imageIndex.value = images.value.length - 1;
  }
  if (imageIndex.value >= images.value.length - 4) {
    triggerInfiniteScroll();
  }
  emitter.emit("update-image-grid-scroll-position");
}

export function repeatLastTagAssignment() {
  const image = images.value[imageIndex.value];
  if (!image) {
    return;
  }

  const lastAppliedTag = tagStack.value[tagStack.value.length - 1];
  if (!lastAppliedTag) {
    return;
  }

  if (image.tags.some((a) => a.tag.id === lastAppliedTag.id)) {
    return;
  }
  addImageTag(image, lastAppliedTag);
}

// Tag hotkey actuation: toggle the named tag on the current image. Assigning
// goes through addImageTag (multi-select aware); removing strips the tag from
// the current image and any multi-selected images carrying it.
export async function toggleTagByName(tagName: string) {
  const image = images.value[imageIndex.value];
  if (!image) {
    return;
  }
  const userStore = useUserStore();
  const tag = userStore.projectTags.find((t) => t.name.toLowerCase() === tagName.toLowerCase());
  if (!tag) {
    showNotificationToast({ headline: `Tag '${tagName}' not found in this project`, type: "warning" });
    return;
  }
  if (!image.tags.some((a) => a.tag.id === tag.id)) {
    await addImageTag(image, tag);
    return;
  }
  if (!canEditImageTag(image, tag)) {
    showNotificationToast({ headline: `'${tag.name}' is frozen while the upload is in review`, type: "warning" });
    return;
  }
  const targets = new Set(imageIndices.value);
  targets.add(imageIndex.value);
  try {
    for (const idx of targets) {
      const img = images.value[idx];
      const assignment = img?.tags.find((a) => a.tag.id === tag.id);
      if (!assignment) continue;
      await api.imageTagAssignments.remove(assignment.id);
      img.tags.splice(
        img.tags.findIndex((a) => a.id === assignment.id),
        1,
      );
      img.updatedAt = new Date().toISOString();
    }
    showNotificationToast({ headline: `Tag ${tag.name} removed`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
