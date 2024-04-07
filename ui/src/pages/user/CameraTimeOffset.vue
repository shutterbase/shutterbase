<template>
  <div class="mx-auto max-w-7xl lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div v-if="camera" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">
          Creating a new time offset for <span class="font-bold">{{ actingUserId === camera.expand.user.id ? "Your" : fullNamePossessive(camera.expand.user) }}</span> camera
          <span class="font-bold">{{ camera.name }}</span>
        </h2>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-300">
          Photograph the QR code below with
          <span class="font-bold">{{ actingUserId === camera.expand.user.id ? "Your" : fullNamePossessive(camera.expand.user) }} {{ camera.name }}</span> as JPEG and upload the
          resulting image here.
        </p>
      </div>
      <div class="mt-12">
        <div class="mx-auto grid max-w-2xl grid-cols-1 items-start gap-x-8 lg:mx-0 lg:max-w-none lg:grid-cols-2">
          <QrTimeCode />
          <div class="w-64 h-64 bg-primary-400"></div>
        </div>
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import * as websocket from "src/util/websocket";
import { onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import QrTimeCode from "src/components/QrTimeCode.vue";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { CamerasResponse } from "src/types/pocketbase";
import { fullNamePossessive } from "src/util/userUtil";

const router = useRouter();
const route = useRoute();

const cameraId = ref<string>(`${route.params.cameraid}`);
const camera = ref<CamerasResponse>();

const actingUserId = pb.authStore.model?.id;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function getCamera() {
  try {
    console.log(`Getting camera ${cameraId.value}`);
    const response = await pb.collection<CamerasResponse>("cameras").getOne(cameraId.value, {
      expand: "user",
    });
    console.log(`Camera retrieved with ID ${cameraId.value}`);
    console.log(response);
    camera.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(getCamera);

onMounted(websocket.connect);
onUnmounted(websocket.disconnect);

// async function createItem() {
//   try {
//     console.log(`Creating ${ITEM_NAME} ${item.value.name}`);
//     const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).create({ ...item.value, user: userId.value });
//     const itemId = response.id;
//     console.log(`Project created with ID ${itemId}`);
//     showNotificationToast({ headline: `${capitalize(ITEM_NAME)} created`, type: "success" });
//     await router.push({ name: `cameras`, params: { cameraid: itemId, userid: userId.value } });
//   } catch (error: any) {
//     unexpectedError.value = error;
//     showUnexpectedErrorMessage.value = true;
//   }
// }
</script>
