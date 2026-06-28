<template>
  <main class="py-8">
    <div class="sm:flex sm:items-end sm:justify-between">
      <div class="sm:flex-auto">
        <p class="label-mono text-accent-600 dark:text-accent-400">Cameras</p>
        <h1 class="display mt-2 text-2xl text-primary-900 dark:text-white">
          <span v-if="userId === userStore.user?.id">Your cameras</span>
          <span v-else>Cameras of {{ fullName() }}</span>
        </h1>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Add and manage cameras here</p>
      </div>
      <div class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
        <button
          @click="addItem"
          type="button"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
        >
          Add Camera
        </button>
      </div>
    </div>
    <div class="my-8 border-t border-primary-200 dark:border-primary-800"></div>
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
import { api } from "src/api";
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

type ITEM_TYPE = CamerasResponse;
const ITEM_NAME = "camera";

const userId: string = `${route.params.userid}`;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<ITEM_TYPE[]> = ref([]);

async function requestItems() {
  try {
    const resultList = await api.cameras.list({ userId, limit: 50 });
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
    const response = await api.cameras.update(data.id, { name: data.name });
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
