<template>
  <div class="mx-auto max-w-7xl lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">New Upload</h2>
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
            <div class="border-b border-gray-900/10 dark:border-gray-100/10 pb-12">
              <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                This will create a new upload from a <b>single camera</b> for the currently active project <b>{{ activeProject.name }}</b>
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
                  <label for="upload-name" class="block text-sm font-medium leading-6 text-gray-900 dark:text-primary-200">Upload name</label>
                  <div class="mt-2">
                    <div
                      class="flex rounded-md shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-700 focus-within:ring-2 focus-within:ring-inset focus-within:ring-primary-600 dark:focus-within:ring-primary-400 sm:max-w-md"
                    >
                      <input
                        type="text"
                        name="upload-name"
                        id="upload-name"
                        v-model="uploadNameOverride"
                        :class="[
                          `block w-full rounded-md border-0 py-1.5 focus:ring-2 focus:ring-inset shadow-sm ring-1 ring-inset sm:text-sm sm:leading-6`,
                          `text-gray-900 placeholder:text-gray-900 dark:placeholder:text-gray-100 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-900`,
                        ]"
                        :placeholder="uploadName"
                      />
                    </div>
                  </div>
                </div>

                <div class="sm:col-span-3">
                  <label for="camera" class="block text-sm font-medium leading-6 text-gray-900 dark:text-primary-200">Camera</label>
                  <div class="mt-2">
                    <select
                      id="camera"
                      name="camera"
                      v-model="camera"
                      placeholder="Select a camera"
                      class="block w-full rounded-md border-0 py-1.5 text-gray-900 dark:text-gray-100 shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-primary-900 sm:max-w-xs sm:text-sm sm:leading-6"
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
            class="mt-6 block rounded-md bg-secondary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
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
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { dateTimeFromUnix, parseBackendTime, timeOffsetUpToDate } from "src/util/dateTimeUtil";

const router = useRouter();

const { activeProject } = storeToRefs(useUserStore());
const userId: string = pb.authStore.model?.id;

const upload = ref<UploadsResponse | null>(null);

const loading = ref(true);

type CamerasType = CamerasResponse & { expand?: { time_offsets_via_camera: TimeOffsetsResponse[] } };
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
    const resultList = await pb.collection<CamerasType>("cameras").getList(1, 50, {
      filter: `user='${userId}'`,
      expand: "time_offsets_via_camera",
    });
    cameras.value = resultList.items.map((item) => {
      const timeOffsets = item.expand?.time_offsets_via_camera || [];
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
    });
    loading.value = false;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function createUpload() {
  try {
    console.log(`Creating upload ${uploadName.value} for camera ${camera.value?.name} in project ${activeProject.value.name}`);
    const response = await pb.collection<UploadsResponse>("uploads").create({
      name: uploadName.value,
      project: activeProject.value.id,
      camera: camera.value?.id,
      user: userId,
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
