import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { api } from "src/api";
import { ImageTag } from "src/types/api";
import { ImageWithTagsType } from "src/types/custom";
import { buildImageListParams } from "src/pages/image/imageListParams";
import { emitter } from "src/boot/mitt";
import { HotkeyEvent, onHotkey } from "src/util/keyEvents";

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

export async function loadImages(reload: boolean) {
  if (loading.value) return;
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
    totalImageCount.value = result.total;
    page.value++;

    if (reload) {
      images.value = [];
    }
    images.value.push(...result.items);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    loading.value = false;
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

onHotkey({ key: "ArrowRight", modifierKeys: [] }, nextImage);
onHotkey({ key: "l", modifierKeys: [] }, nextImage);
function nextImage(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }
  event.event.preventDefault();
  if (imageIndex.value < images.value.length - 1) {
    imageIndex.value++;
  }
  if (imageIndex.value === images.value.length - 4) {
    triggerInfiniteScroll();
  }
  emitter.emit("update-image-grid-scroll-position");
}

onHotkey({ key: "ArrowLeft", modifierKeys: [] }, previousImage);
onHotkey({ key: "h", modifierKeys: [] }, previousImage);
function previousImage(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }
  event.event.preventDefault();
  if (imageIndex.value > 0) {
    imageIndex.value--;
  }
  emitter.emit("update-image-grid-scroll-position");
}

onHotkey({ key: "ArrowUp", modifierKeys: [] }, previousRow);
onHotkey({ key: "k", modifierKeys: [] }, previousRow);
function previousRow(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }
  event.event.preventDefault();
  if (imageIndex.value - 4 >= 0) {
    imageIndex.value -= 4;
  } else {
    imageIndex.value = 0;
  }
  emitter.emit("update-image-grid-scroll-position");
}

onHotkey({ key: "ArrowDown", modifierKeys: [] }, nextRow);
onHotkey({ key: "j", modifierKeys: [] }, nextRow);
function nextRow(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }
  event.event.preventDefault();
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

onHotkey({ key: "s", modifierKeys: [] }, repeatLastTagAssignment);
function repeatLastTagAssignment(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }
  event.event.preventDefault();
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
