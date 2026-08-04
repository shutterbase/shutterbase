<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <p class="label-mono text-accent-600 dark:text-accent-400">Upload</p>
      <h2 class="display mt-2 text-2xl text-primary-900 dark:text-white">New Upload</h2>
      <div v-if="!activeProject.id">
        <AlertBanner
          :type="AlertBannerType.ERROR"
          headline="No active project"
          message="Please select a project first"
          :actions="[{ text: 'Select project', onClick: () => router.push({ name: 'projects' }) }]"
        />
      </div>
      <div v-else-if="loading">
        <AlertBanner :type="AlertBannerType.INFO" headline="Loading required information" message="Please wait" />
      </div>
      <div v-else-if="cameras.length === 0">
        <AlertBanner
          :type="AlertBannerType.ERROR"
          headline="No cameras available"
          message="Please create a camera first"
          :actions="[{ text: 'Create camera', onClick: () => router.push({ name: 'camera-create', params: { userid: userId } }) }]"
        />
      </div>
      <div v-else class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
        <div>
          <div class="space-y-12">
            <div class="border-b border-primary-200 dark:border-primary-800 pb-12">
              <p class="mt-3 text-sm leading-6 text-primary-600 dark:text-primary-400">
                This will create a new upload from a <b class="text-primary-800 dark:text-primary-200">single camera</b> for the currently active project
                <b class="text-primary-800 dark:text-primary-200">{{ activeProject.name }}</b>
              </p>

              <AlertBanner
                :type="AlertBannerType.WARNING"
                v-if="outdatedTimeOffsetFound"
                headline="Outdated time offsets"
                message="Some of your cameras have outdated time offsets. Please create a new time offset to enable it for uploading."
                :actions="[{ text: 'Manage cameras', onClick: () => router.push({ name: 'cameras', params: { userid: userId } }) }]"
              />

              <div class="mt-10 grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
                <div class="sm:col-span-3">
                  <label for="upload-name" class="label-mono block text-primary-500 dark:text-primary-400">Upload name</label>
                  <div class="mt-2">
                    <div class="sm:max-w-md">
                      <input
                        type="text"
                        name="upload-name"
                        id="upload-name"
                        v-model="uploadNameOverride"
                        class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
                        :placeholder="uploadName"
                      />
                    </div>
                  </div>
                </div>

                <div class="sm:col-span-3">
                  <label for="camera" class="label-mono block text-primary-500 dark:text-primary-400">Camera</label>
                  <div class="mt-2">
                    <select
                      id="camera"
                      name="camera"
                      v-model="camera"
                      placeholder="Select a camera"
                      class="h-10 w-full cursor-pointer rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:hover:border-primary-600 sm:max-w-xs"
                    >
                      <option selected disabled>-- select one of your cameras --</option>
                      <option v-for="camera in cameras" :key="camera.id" :value="camera" :disabled="camera.disabled">{{ camera.name }}</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <button
            @click="createUpload"
            :disabled="!camera || uploadName === ''"
            class="mt-6 inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
          >
            Create
          </button>
        </div>
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import AlertBanner, { AlertBannerType } from "src/components/AlertBanner.vue";
import { UploadsResponse, CamerasResponse, TimeOffsetsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { dateTimeFromUnix, parseBackendTime, timeOffsetUpToDate } from "src/util/dateTimeUtil";

const router = useRouter();

const userStore = useUserStore();
const { activeProject } = storeToRefs(userStore);
const userId: string = userStore.user?.id || "";

const upload = ref<UploadsResponse | null>(null);

const loading = ref(true);

type CamerasType = CamerasResponse;
const cameras = ref<(CamerasType & { disabled: boolean })[]>([]);
const camera = ref<CamerasType | null>();

const uploadNameOverride = ref("");
const uploadName = computed(() => {
  if (uploadNameOverride.value !== "") {
    return uploadNameOverride.value;
  }
  const now = new Date();
  let result = dateTimeFromUnix(now.getTime() / 1000);
  if (camera.value) {
    result += ` - ${camera.value.name}`;
  }
  return result;
});

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const outdatedTimeOffsetFound = ref(false);

async function getCameras() {
  try {
    const resultList = await api.cameras.list({ userId, limit: 50 });
    cameras.value = await Promise.all(
      resultList.items.map(async (item) => {
        const timeOffsets = (await api.timeOffsets.list({ cameraId: item.id, limit: 50 })).items;
        let disabled = true;
        for (const timeOffset of timeOffsets) {
          if (timeOffsetUpToDate(timeOffset)) {
            disabled = false;
            break;
          }
        }
        if (disabled) {
          outdatedTimeOffsetFound.value = true;
        }
        return { ...item, disabled };
      }),
    );
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    loading.value = false;
  }
}

async function createUpload() {
  try {
    console.log(`Creating upload ${uploadName.value} for camera ${camera.value?.name} in project ${activeProject.value.name}`);
    const response = await api.uploads.create({
      name: uploadName.value,
      projectId: activeProject.value.id,
      cameraId: camera.value!.id,
      userId,
    });
    const itemId = response.id;
    console.log(`upload created with ID ${itemId}`);
    showNotificationToast({ headline: `Upload created`, type: "success" });
    router.push({ name: "upload-edit", params: { id: itemId } });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(getCameras);
</script>
