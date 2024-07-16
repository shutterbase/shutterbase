import pb from "src/boot/pocketbase";
import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import { ImageTagAssignmentType, ImageWithTagsType } from "src/types/custom";
import { ImageTagsResponse } from "src/types/pocketbase";
import { emitter } from "src/boot/mitt";
import { HotkeyEvent, onHotkey } from "src/util/keyEvents";

export enum DisplayMode {
  GRID = "grid",
  DETAIL = "detail",
}

export const { activeProject, preferredImageSortOrder } = storeToRefs(useUserStore());

export const showUnexpectedErrorMessage = ref(false);
export const unexpectedError = ref(null);

export const taggingDialogVisible = ref(false);

export const images = ref<ImageWithTagsType[]>([]);
export const imageIndex = ref(-1);
export const totalImageCount = ref(0);
const page = ref(1);
export const loading = ref(false);
export const filtered = ref(false);

export const selectedImageIndex = ref(-1);

export const searchText = ref("");
export function updateSearchText(text: string) {
  searchText.value = text;
}

export async function triggerInfiniteScroll() {
  if (totalImageCount.value > 0 && images.value.length < totalImageCount.value) {
    loadImages(false);
  }
}

function getFilter() {
  const and = [];
  and.push(`project='${activeProject.value.id}'`);

  if (searchText.value) {
    filtered.value = true;
  } else {
    filtered.value = false;
  }

  if (searchText.value) {
    and.push(`(computedFileName ~ '${searchText.value}' || fileName ~ '%${searchText.value}%')`);
  }
  return `(${and.join(" && ")})`;
}

function getSort() {
  return preferredImageSortOrder.value === SORT_ORDER.LATEST_FIRST ? "-capturedAtCorrected" : "capturedAtCorrected";
}

export async function loadImages(reload: boolean) {
  if (loading.value) return;
  loading.value = true;
  try {
    if (reload) page.value = 1;
    const result = await pb.collection<ImageWithTagsType>("images").getList(page.value, 20, {
      filter: getFilter(),
      sort: getSort(),
      expand: "camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag",
    });
    totalImageCount.value = result.totalItems;
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

export async function addImageTag(image: ImageWithTagsType, tag: ImageTagsResponse) {
  if (!image) {
    return;
  }

  try {
    const result = await pb.collection("image_tag_assignments").create<ImageTagAssignmentType>({
      image: image.id,
      imageTag: tag.id,
      type: "manual",
    });
    result.expand = { imageTag: tag };
    const editedImageIndex = images.value.findIndex((i) => i.id === image.id);
    images.value[editedImageIndex].expand.image_tag_assignments_via_image.push(result);
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
