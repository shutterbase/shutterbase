<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div v-if="camera" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <p class="label-mono text-accent-600 dark:text-accent-400">Time sync</p>
        <h2 class="display mt-2 text-2xl text-primary-900 dark:text-white">
          Creating a new time offset for <span class="font-bold">{{ actingUserId === camera.user.id ? "Your" : fullNamePossessive(camera.user) }}</span> camera
          <span class="font-bold">{{ camera.name }}</span>
        </h2>
        <p class="mt-2 text-sm leading-6 text-primary-500 dark:text-primary-400">
          Photograph the QR code below with
          <span class="font-semibold text-primary-700 dark:text-primary-200">{{ actingUserId === camera.user.id ? "Your" : fullNamePossessive(camera.user) }} {{ camera.name }}</span> as JPEG and upload the
          resulting image here.
        </p>
      </div>
      <div class="mt-12">
        <div class="mx-auto grid max-w-2xl grid-cols-1 items-start gap-x-8 lg:mx-0 lg:max-w-none lg:grid-cols-2">
          <QrTimeCode />
          <FileDropzone :multiple="false" @files="handleFiles" />
          <div v-if="timeOffsetResult" class="mt-12">
            <h2 class="display text-lg text-primary-900 dark:text-white">Time Offset</h2>
            <p class="mt-1 text-sm leading-6 text-primary-500 dark:text-primary-400">
              Your camera <b class="text-primary-700 dark:text-primary-200">{{ timeOffsetResult.model }}</b> is <span class="font-data">{{ Math.abs(timeOffsetResult.timeOffset) }}</span> seconds <span v-if="timeOffsetResult.timeOffset > 0">behind</span
              ><span v-else-if="timeOffsetResult.timeOffset < 0">ahead of</span> the server's time.
            </p>
            <div class="mt-6 space-y-6 divide-y divide-primary-100 dark:divide-primary-800 border-t border-primary-200 dark:border-primary-800 text-sm leading-6">
              <div class="pt-3 sm:flex">
                <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6">Time Offset</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="font-data py-1.5 text-primary-900 dark:text-primary-100">{{ timeOffsetResult.timeOffset }} seconds</div>
                  </div>
                </dd>
              </div>
              <div class="pt-3 sm:flex">
                <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6">Server Time</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="font-data py-1.5 text-primary-900 dark:text-primary-100">{{ dateTimeFromUnix(timeOffsetResult.serverTime / 1000) }}</div>
                  </div>
                </dd>
              </div>
              <div class="pt-3 sm:flex">
                <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6">Camera Time</dt>
                <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
                  <div>
                    <div class="font-data py-1.5 text-primary-900 dark:text-primary-100">{{ dateTimeFromUnix(timeOffsetResult.cameraTime / 1000) }}</div>
                  </div>
                </dd>
              </div>
            </div>
            <div v-if="!timeOffsetCreated" class="mt-10 flex items-center justify-center gap-x-6">
              <button
                @click="saveTimeOffset"
                class="inline-flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950 sm:w-auto"
              >
                <CheckCircleIcon class="h-5 w-5" />Save time offset
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
import { api } from "src/api";
import { useUserStore } from "src/stores/user-store";
import { showNotificationToast } from "src/boot/mitt";
import { CamerasResponse } from "src/types/pocketbase";
import { fullNamePossessive } from "src/util/userUtil";
import init, { get_image_metadata, get_time_offset, parse_qr_code, TimeOffsetResult } from "image-wasm";
import * as fileUtil from "src/util/fileUtil";
import { dateTimeFromUnix } from "src/util/dateTimeUtil";
import { CheckCircleIcon } from "@heroicons/vue/24/outline";
import { TimeOffsetsResponse } from "src/types/pocketbase";

const router = useRouter();
const route = useRoute();

type ITEM_TYPE = CamerasResponse;

const cameraId = ref<string>(`${route.params.cameraid}`);
const camera = ref<ITEM_TYPE>();

const actingUserId = useUserStore().user?.id;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const timeOffsetCreated = ref(false);

type TimeOffsetMetadata = {
  timeOffset: number;
  serverTime: number;
  cameraTime: number;
  model: string;
  lensModel: string;
};

const timeOffsetResult = ref<TimeOffsetMetadata>();

async function getCamera() {
  try {
    console.log(`Getting camera ${cameraId.value}`);
    const response = await api.cameras.get(cameraId.value);
    console.log(`Camera retrieved with ID ${cameraId.value}`);
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
    // server computes timeOffset = serverTime - cameraTime (§4.10)
    const response = await api.timeOffsets.create({
      cameraId: cameraId.value,
      cameraTime: new Date(timeOffsetResult.value.cameraTime).toISOString(),
      serverTime: new Date(timeOffsetResult.value.serverTime).toISOString(),
    });
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
