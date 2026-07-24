import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { api } from "src/api";
import { ImageTag } from "src/types/api";
import { ImageWithTagsType } from "src/types/custom";
import { buildImageListParams } from "src/pages/image/imageListParams";
import { emitter, showNotificationToast } from "src/boot/mitt";

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
export function updateFilterTags(tags: ImageTag[]) {
  filterTags.value = tags;
}

export const aspectRatioFilter = ref("neutral");
export function updateAspectRatioFilter(aspectRatioState: string) {
  aspectRatioFilter.value = aspectRatioState;
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

export async function loadImages(reload: boolean) {
  // pagination guard only: don't stack "load more" requests, but a reload
  // (new filter/search/sort) must always issue a fresh query.
  if (!reload && loading.value) return;
  const myRequestId = ++requestId;
  loading.value = true;
  try {
    if (reload) page.value = 1;

    filtered.value = !!searchText.value || filterTags.value.length > 0 || aspectRatioFilter.value !== "neutral";

    const params = buildImageListParams({
      projectId: activeProject.value.id,
      search: searchText.value,
      tags: filterTags.value,
      orientation: aspectRatioFilter.value,
      sortOrder: preferredImageSortOrder.value,
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

export async function addImageTag(image: ImageWithTagsType, tag: ImageTag) {
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

// Hotkey handlers: bound to their action ids by Images.vue (useHotkeyAction),
// so they are only active while the images page is mounted. Context gating in
// the dispatcher keeps them silent while the tagging dialog is open.
export function nextImage() {
  if (imageIndex.value < images.value.length - 1) {
    imageIndex.value++;
  }
  if (imageIndex.value === images.value.length - 4) {
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
