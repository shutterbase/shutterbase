<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div v-if="camera" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <button
          @click="router.push({ name: 'cameras', params: { userid: camera.user.id } })"
          class="mb-6 inline-flex cursor-pointer items-center gap-1.5 text-sm font-medium text-primary-500 transition-colors hover:text-primary-900 dark:text-primary-400 dark:hover:text-white"
        >
          <ArrowLeftIcon class="h-4 w-4" />Back to cameras
        </button>
        <p class="label-mono text-accent-600 dark:text-accent-400">Time sync</p>
        <h2 class="display mt-2 text-2xl text-primary-900 dark:text-white">
          Creating a new time offset for <span class="font-bold">{{ actingUserId === camera.user.id ? "Your" : fullNamePossessive(camera.user) }}</span> camera
          <span class="font-bold">{{ camera.name }}</span>
        </h2>
        <p class="mt-2 text-sm leading-6 text-primary-500 dark:text-primary-400">
          Photograph the QR code below with
          <span class="font-semibold text-primary-700 dark:text-primary-200"
            >{{ actingUserId === camera.user.id ? "Your" : fullNamePossessive(camera.user) }} {{ camera.name }}</span
          >
          as JPEG and upload the resulting image here.
        </p>
      </div>
      <div class="mt-12">
        <div class="mx-auto grid max-w-2xl grid-cols-1 items-start gap-x-8 lg:mx-0 lg:max-w-none lg:grid-cols-2">
          <QrTimeCode />
          <FileDropzone :multiple="false" @files="handleFiles" />
          <div v-if="timeOffsetResult" class="mt-12">
            <h2 class="display text-lg text-primary-900 dark:text-white">Time Offset</h2>
            <p class="mt-1 text-sm leading-6 text-primary-500 dark:text-primary-400">
              Your camera <b class="text-primary-700 dark:text-primary-200">{{ timeOffsetResult.model }}</b> is
              <span class="font-data">{{ Math.abs(timeOffsetResult.timeOffset) }}</span> seconds <span v-if="timeOffsetResult.timeOffset > 0">behind</span
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
                :disabled="pending"
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

    <!-- Friendly reminder: users shoot the QR in JPEG and have forgotten to switch
         the camera back to RAW afterwards, ruining subsequent uploads. -->
    <ModalMessage
      :show="showRawReminder"
      :type="MessageType.CONFIRM_INFO"
      headline="Before you keep shooting"
      message="Two things now that the offset is saved: switch your camera back to RAW image quality (easy to forget after shooting the QR code in JPEG), and don't change this camera's date or time for the rest of the project — every future offset depends on the clock staying put."
      confirmText="Yes, I've switched back to RAW"
      cancelText="Dismiss"
      @confirmed="onRawConfirmed"
      @closed="showRawReminder = false"
    />

    <!-- Shown the moment a large offset is identified while the project still has
         no uploads: better to fix the camera clock once than carry it all project.
         Once uploads exist, changing the clock would desync them, so it stays hidden. -->
    <ModalMessage
      :show="showBigOffsetWarning"
      :type="MessageType.CONFIRM_WARNING"
      headline="Large offset — fix the camera clock instead?"
      :message="bigOffsetMessage"
      confirmText="Discard time offset"
      cancelText="Accept time offset"
      @confirmed="onDiscardOffset"
      @closed="onAcceptOffset"
    />
  </div>
</template>

<script setup lang="ts">
import * as websocket from "src/util/websocket";
import { onMounted, onUnmounted, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
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
import { CheckCircleIcon, ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { TimeOffsetsResponse } from "src/types/pocketbase";

const router = useRouter();
const route = useRoute();

type ITEM_TYPE = CamerasResponse;

const cameraId = ref<string>(`${route.params.cameraid}`);
const camera = ref<ITEM_TYPE>();

const actingUserId = useUserStore().user?.id;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

type TimeOffsetMetadata = {
  timeOffset: number;
  serverTime: number;
  cameraTime: number;
  model: string;
  lensModel: string;
};

const timeOffsetResult = ref<TimeOffsetMetadata>();

const timeOffsetCreated = ref(false);
const pending = ref(false);
const showRawReminder = ref(false);
const showBigOffsetWarning = ref(false);

function onRawConfirmed() {
  showRawReminder.value = false;
  showNotificationToast({ headline: "Great — happy shooting!", type: "success" });
}

const bigOffsetMessage = computed(
  () =>
    `This camera is off by ${timeOffsetResult.value ? Math.abs(timeOffsetResult.value.timeOffset) : 0} seconds and there are no uploads in this project yet. ` +
    `It's better to set the camera's clock to the correct time once now, then scan the QR code again for a fresh, near-zero offset — rather than carrying a large offset for the whole project.`,
);

function onDiscardOffset() {
  showBigOffsetWarning.value = false;
  timeOffsetResult.value = undefined; // drop it; user re-scans after fixing the clock
  showNotificationToast({ headline: "Set the camera's clock to the correct time, then scan the QR code again", type: "info" });
}

function onAcceptOffset() {
  // Keep the offset as-is; the user saves it via the Save button when ready.
  showBigOffsetWarning.value = false;
}

// Scope to the active project when one is selected; otherwise check the owner's
// uploads across every project (this user-scoped page often has no active project).
async function cameraOwnerHasNoUploads(): Promise<boolean> {
  const ownerId = camera.value?.user?.id;
  if (!ownerId) return false; // can't identify the owner → don't warn
  try {
    const projectId = useUserStore().activeProjectId || undefined;
    const res = await api.uploads.list({ projectId, userId: ownerId, limit: 1 });
    return res.total === 0;
  } catch {
    return false; // don't warn on a check failure
  }
}

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

function showError(error: any) {
  unexpectedError.value = error;
  showUnexpectedErrorMessage.value = true;
}

async function handleFiles(files: File[]) {
  if (files.length !== 1) {
    showError(new Error(`upload exactly one photo of the QR code (got ${files.length} files)`));
    return;
  }
  // loadFile rejects on failure (it never resolves null) — catch, don't null-check.
  let data: ArrayBuffer;
  try {
    data = await fileUtil.loadFile(files[0]);
  } catch {
    showError(new Error(`could not read the file '${files[0].name}' — it may be corrupt or unreadable`));
    return;
  }
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
    // Warn immediately (not at save time) when the offset is large and the project
    // has no uploads yet — abs() so it triggers for positive or negative drift.
    if (Math.abs(timeOffsetResult.value.timeOffset) > 10 && (await cameraOwnerHasNoUploads())) {
      showBigOffsetWarning.value = true;
    }
  } catch (error: any) {
    // WASM errors (unreadable image, no/blurry QR, ECC failure, missing EXIF
    // time) arrive as js Errors whose message names the failure — show it.
    console.error(error);
    showError(error);
    return;
  }
}

async function saveTimeOffset() {
  if (timeOffsetResult.value == null) {
    console.log("No time offset to save");
    return;
  }
  pending.value = true;
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
    showRawReminder.value = true;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    pending.value = false;
  }
}

onMounted(getCamera);

onMounted(websocket.connect);
onUnmounted(websocket.disconnect);
</script>
