<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h1 class="text-base font-semibold leading-6 text-gray-900 dark:text-gray-100">
          <span v-if="userId === pb.authStore.model?.id">Your cameras</span>
          <span v-else>Cameras of {{ fullName() }}</span>
        </h1>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">Add and manage cameras here</p>
      </div>
      <div class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
        <button
          @click="addItem"
          type="button"
          class="block rounded-md bg-secondary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
        >
          Add Camera
        </button>
      </div>
    </div>
    <div class="my-10 space-y-6 divide-y divide-gray-100 dark:divide-gray-700 border-t border-gray-200 dark:border-gray-600"></div>
    <div class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
      <div v-for="camera in items" :key="camera.id">
        <CameraEdit :item="camera" @edit-save="saveItem" />
        <CameraTimeOffsets :camera="camera" />
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import pb from "src/boot/pocketbase";
import { CamerasResponse, TimeOffsetsResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import CameraEdit, { CameraEditData } from "src/components/user/CameraEdit.vue";
import CameraTimeOffsets from "src/components/user/CameraTimeOffsets.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter, useRoute } from "vue-router";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
import { fullName } from "src/util/userUtil";

const router = useRouter();
const route = useRoute();

const userStore = useUserStore();

type ITEM_TYPE = CamerasResponse & { expand?: { time_offsets_via_camera: TimeOffsetsResponse[] } };
const ITEM_COLLECTION = "cameras";
const ITEM_NAME = "camera";

const userId: string = `${route.params.userid}`;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<ITEM_TYPE[]> = ref([]);

async function requestItems() {
  try {
    const resultList = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).getList(1, 50, {
      filter: `user='${userId}'`,
      expand: "time_offsets_via_camera",
    });
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(item: CamerasResponse, editData: CameraEditData) {
  if (!item) {
    console.log("No item to save");
    return;
  }
  const rollbackData = { ...item };
  const data = { ...item, ...editData };

  try {
    console.log(`Saving item ${item.id}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).update(data.id, data);
    const index = items.value.findIndex((i) => i.id === item.id);
    items.value[index] = response;
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} saved`, type: "success" });
  } catch (error: any) {
    item = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function addItem() {
  router.push({ name: `camera-create`, params: { userid: userId } });
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
