<template>
  <div class="flex">
    <Sidebar :item="item" />
    <div v-if="item" class="flex-1 flex items-center justify-center mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8">
      <div class="relative">
        <img :src="item.downloadUrls['2048']" alt="Centered Image" class="max-w-full max-h-[52rem] mx-auto drop-shadow-lg" />
      </div>
    </div>
  </div>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  <TaggingDialog ref="taggingDialog" :shown="taggingDialogVisible" @close="hideTaggingDialog" @selected="addImageTag" :image="item" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ImageWithTagsType, ImageTagAssignmentType } from "src/types/custom";
import Sidebar from "src/components/image/Sidebar.vue";
import TaggingDialog from "src/components/image/TaggingDialog.vue";
import pb from "src/boot/pocketbase";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
import { debug } from "src/util/logger";
import { T } from "app/dist/spa/assets/keyboard-CdObzdke";
import { HotkeyEvent, onHotkey } from "src/util/keyEvents";
import { ImageTagAssignmentsRecord, ImageTagsResponse } from "src/types/pocketbase";
const route = useRoute();

type ITEM_TYPE = ImageWithTagsType;
const ITEM_COLLECTION = "images";
const ITEM_NAME = "image";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const taggingDialog = ref<InstanceType<typeof TaggingDialog> | null>(null);

onMounted(loadItem);
watch(route, loadItem);
async function loadItem() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).getOne(itemId, {
      expand: "camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag",
    });
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const taggingDialogVisible = ref(false);

onHotkey({ key: "t", modifierKeys: [] }, showTaggingDialogViaHotkey);
emitter.on("show-tagging-dialog", showTaggingDialog); // from sidebar button
function showTaggingDialogViaHotkey(event: HotkeyEvent) {
  if (!taggingDialogVisible.value) {
    event.event.preventDefault();
  }
  showTaggingDialog();
}
function showTaggingDialog() {
  if (!taggingDialogVisible.value) {
    taggingDialogVisible.value = true;
    taggingDialog.value?.focusSearchText();
    taggingDialog.value?.clearSearchText();
    debug("show tag dialog");
  }
}
onHotkey({ key: "Escape", modifierKeys: [] }, hideTaggingDialog);
function hideTaggingDialog() {
  if (taggingDialogVisible.value) {
    taggingDialogVisible.value = false;
    debug("hide tag dialog");
  }
}

async function addImageTag(tag: ImageTagsResponse) {
  if (!item.value) {
    return;
  }
  const imageId = item.value.id;

  try {
    const result = await pb.collection("image_tag_assignments").create<ImageTagAssignmentType>({
      image: imageId,
      imageTag: tag.id,
      type: "manual",
    });
    result.expand = { imageTag: tag };
    item.value.expand.image_tag_assignments_via_image.push(result);
    taggingDialog.value?.focusSearchText();
    taggingDialog.value?.clearSearchText();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
