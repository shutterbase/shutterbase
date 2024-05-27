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
          <FileDropzone :multiple="false" @files="handleFiles" />
          <div v-if="timeOffsetResult" class="mt-12">
            <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">Time Offset</h2>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-300">
              Your camera <b>{{ timeOffsetResult.model }}</b> is {{ Math.abs(timeOffsetResult.timeOffset) }} seconds <span v-if="timeOffsetResult.timeOffset > 0">behind</span
              ><span v-else-if="timeOffsetResult.timeOffset < 0">ahead of</span> the server's time.
            </p>
            <div class="mt-6 space-y-6 divide-y divide-gray-100 dark:divide-gray-700 border-t border-gray-200 dark:border-gray-600 text-sm leading-6">
              <div class="pt-3 sm:flex">
                <dt class="font-medium text-gray-900 dark:text-primary-200 sm:w-64 sm:flex-none sm:pr-6">Time Offset</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="py-1.5 text-gray-900 dark:text-primary-200">{{ timeOffsetResult.timeOffset }} seconds</div>
                  </div>
                </dd>
              </div>
              <div class="pt-3 sm:flex">
                <dt class="font-medium text-gray-900 dark:text-primary-200 sm:w-64 sm:flex-none sm:pr-6">Server Time</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="py-1.5 text-gray-900 dark:text-primary-200">{{ dateTimeFromUnix(timeOffsetResult.serverTime / 1000) }}</div>
                  </div>
                </dd>
              </div>
              <div class="pt-3 sm:flex">
                <dt class="font-medium text-gray-900 dark:text-primary-200 sm:w-64 sm:flex-none sm:pr-6">Camera Time</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="py-1.5 text-gray-900 dark:text-primary-200">{{ dateTimeFromUnix(timeOffsetResult.cameraTime / 1000) }}</div>
                  </div>
                </dd>
              </div>
            </div>
            <div v-if="!timeOffsetCreated" class="mt-10 flex items-center justify-center gap-x-6">
              <button
                @click="saveTimeOffset"
                class="bg-secondary-600 hover:bg-secondary-700 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto"
              >
                <CheckCircleIcon class="mr-2 w-5 h-5 text-white" />Save time offset
              </button>
            </div>
          </div>
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
import FileDropzone from "src/components/FileDropzone.vue";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { CamerasResponse, TimeOffsetsRecord, UsersResponse } from "src/types/pocketbase";
import { fullNamePossessive } from "src/util/userUtil";
import init, { get_image_metadata, get_time_offset, parse_qr_code } from "image-wasm";
import * as fileUtil from "src/util/fileUtil";
import { dateTimeFromUnix } from "src/util/dateTimeUtil";
import { CheckCircleIcon } from "@heroicons/vue/24/outline";
import { TimeOffsetsResponse } from "src/types/pocketbase";

const router = useRouter();
const route = useRoute();

type ITEM_TYPE = CamerasResponse & { expand: { user: UsersResponse } };

const cameraId = ref<string>(`${route.params.cameraid}`);
const camera = ref<ITEM_TYPE>();

const actingUserId = pb.authStore.model?.id;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const timeOffsetCreated = ref(false);

type TimeOffsetResult = {
  timeOffset: number;
  serverTime: number;
  cameraTime: number;
  model: string;
  lensModel: string;
};

const timeOffsetResult = ref<TimeOffsetResult>();

async function getCamera() {
  try {
    console.log(`Getting camera ${cameraId.value}`);
    const response = await pb.collection<ITEM_TYPE>("cameras").getOne(cameraId.value, {
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

async function handleFiles(files: File[]) {
  console.log(files);
  if (files.length !== 1) {
    console.log("Only one file can be uploaded");
    return;
  }
  const data = await fileUtil.loadFile(files[0]);
  if (data == null) {
    console.log("Failed to load file");
    return;
  }
  console.log(data);
  try {
    await init();
    const imageMetadata = await get_image_metadata(data);
    console.log(imageMetadata);
    const timeOffset = await get_time_offset(data);
    console.log(timeOffset);
    timeOffsetResult.value = {
      timeOffset: timeOffset.time_offset,
      serverTime: timeOffset.server_time * 1000,
      cameraTime: timeOffset.camera_time * 1000,
      model: imageMetadata.tags.get("Model"),
      lensModel: imageMetadata.tags.get("LensModel"),
    };
  } catch (error: any) {
    console.log("Failed to get image metadata");
    console.log(error);
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
    return;
  }
}

async function saveTimeOffset() {
  if (timeOffsetResult.value == null) {
    console.log("No time offset to save");
    return;
  }
  try {
    const timeOffsetRecord: TimeOffsetsRecord = {
      timeOffset: timeOffsetResult.value.timeOffset,
      cameraTime: new Date(timeOffsetResult.value.cameraTime).toISOString(),
      serverTime: new Date(timeOffsetResult.value.serverTime).toISOString(),
      camera: cameraId.value,
    };

    const response = await pb.collection<TimeOffsetsResponse>("time_offsets").create(timeOffsetRecord);
    const itemId = response.id;
    console.log(`Time offset created with ID ${itemId}`);
    showNotificationToast({ headline: `Time offset created`, type: "success" });
    timeOffsetCreated.value = true;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(getCamera);

onMounted(websocket.connect);
onUnmounted(websocket.disconnect);
</script>
