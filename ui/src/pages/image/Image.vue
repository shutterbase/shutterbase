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
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ImageWithTagsType } from "src/types/custom";
import Sidebar from "src/components/image/Sidebar.vue";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
const route = useRoute();

type ITEM_TYPE = ImageWithTagsType;
const ITEM_COLLECTION = "images";
const ITEM_NAME = "image";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

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
</script>
