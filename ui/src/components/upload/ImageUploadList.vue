<template>
  <div class="mt-12">
    <div class="px-4 sm:px-6 lg:px-8">
      <div class="sm:flex sm:items-center">
        <div class="sm:flex-auto">
          <h1 class="text-lg font-semibold tracking-tight text-primary-900 dark:text-white">Uploaded images</h1>
          <p class="mt-1.5 text-sm text-primary-500 dark:text-primary-400">
            These images have been added to the upload and are either waiting for processing or have been processed.
          </p>
        </div>
        <!-- <div class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
          <button
            type="button"
            class="block rounded-md bg-primary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-primary-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
          >
            Add user
          </button>
        </div> -->
      </div>
      <div class="mt-8 flow-root">
        <div class="overflow-x-auto">
          <div class="inline-block min-w-full align-middle">
            <table class="min-w-full border-separate border-spacing-0">
              <thead>
                <tr>
                  <th scope="col" :class="[tableHeaderClasses]">Preview</th>
                  <th scope="col" :class="[tableHeaderClasses]">Status</th>
                  <th scope="col" :class="[tableHeaderClasses]">Filename</th>
                  <th scope="col" :class="[tableHeaderClasses]">Time</th>
                  <th scope="col" :class="[tableHeaderClasses]">Size</th>
                  <th v-if="allowEdit" scope="col" :class="[tableHeaderClasses]">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="image in displayedImages" :key="image.id || image.originalFileName">
                  <td :class="[tableCellClasses]" class="relative">
                    <img v-if="image.thumbnail" :src="`data:image/jpeg;base64, ${image.thumbnail}`" alt="Thumbnail" class="thumbnail" />
                    <div v-else-if="image.downloadUrls">
                      <img :src="image.downloadUrls[`256`]" alt="Thumbnail" class="thumbnail" />
                    </div>
                    <div class="tooltip">
                      <img v-if="image.thumbnail" :src="`data:image/jpeg;base64, ${image.thumbnail}`" alt="Preview" class="preview" />
                      <div v-else-if="image.downloadUrls">
                        <img :src="image.downloadUrls[`512`]" alt="Preview" class="preview" />
                      </div>
                      <div class="arrow"></div>
                    </div>
                  </td>
                  <td :class="[tableCellClasses]">{{ image.status }} {{ image.progress !== 0.0 && image.status !== ImageStatus.DONE ? `(${image.progress.toFixed(0)}%)` : `` }}</td>
                  <td :class="[tableCellClasses]">{{ fileNameTableEntry(image) }}</td>
                  <td :class="[tableCellClasses]">{{ timeTableEntry(image) }}</td>
                  <td :class="[tableCellClasses]">{{ fileSize(image.size) }}</td>
                  <td v-if="allowEdit" :class="[tableCellClasses]">
                    <button
                      v-if="image.status === ImageStatus.DONE"
                      @click="deleteItem(image)"
                      class="inline-flex items-center rounded-md border border-error-300 bg-error-50 px-2.5 py-1 text-xs font-medium text-error-700 shadow-sm transition-colors hover:bg-error-100 dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300 dark:hover:bg-error-950/70"
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { UploadsResponse } from "src/types/pocketbase";
import { TimeOffset } from "src/types/api";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { Image, ImageStatus, FileProcessor, newImage, newImageFromBackendImage, TimeOffsetResult } from "src/util/fileProcessor";
import { error } from "src/util/logger";
import { fileSize } from "src/util/fileUtil";

const tableHeaderClasses =
  "label-mono sticky top-0 z-10 border-b border-primary-200 dark:border-primary-800 bg-surface/85 dark:bg-surface-dark/85 px-4 py-3.5 text-left text-primary-500 dark:text-primary-400 backdrop-blur";

const tableCellClasses = "whitespace-nowrap border-b border-primary-100 dark:border-primary-800/70 px-4 py-3 text-sm text-primary-700 dark:text-primary-300";

type UploadType = UploadsResponse;

interface Props {
  upload: UploadType;
  files: File[];
  allowEdit?: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  allowEdit: false,
});

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const upload = computed(() => props.upload);
const images = ref<Image[]>([]);
const uploadedImages = ref<Image[]>([]);

const displayedImages = computed(() => {
  return [...uploadedImages.value, ...images.value];
});

const cameraTimeOffsets = ref<TimeOffset[]>([]);
const timeOffsets = computed(() => {
  return cameraTimeOffsets.value.map((timeOffset) => ({
    free: (): void => {},
    time_offset: BigInt(timeOffset.timeOffset),
    server_time: BigInt(dateTimeUtil.parseBackendTime(timeOffset.serverTime).getTime() / 1000),
    camera_time: BigInt(dateTimeUtil.parseBackendTime(timeOffset.cameraTime).getTime() / 1000),
  }));
});

onMounted(async () => {
  if (props.upload.camera?.id) {
    cameraTimeOffsets.value = (await api.timeOffsets.list({ cameraId: props.upload.camera.id, limit: 50 })).items;
  }
});

const fileProcessor = new FileProcessor(upload, images, timeOffsets);
onUnmounted(() => fileProcessor.stop());

watch(props, (props) => {
  updateFiles(props.files);
});
async function updateFiles(files: File[]) {
  for (const file of files) {
    if (displayedImages.value.find((image) => image.originalFileName === file.name)) {
      continue;
    }
    images.value.push(newImage({ file }));
  }
  fileProcessor.start();
}

onMounted(requestImages);
async function requestImages() {
  try {
    const resultList = await api.images.list({ projectId: upload.value.project.id, uploadId: upload.value.id, limit: 1000 });
    uploadedImages.value = resultList.items.map(newImageFromBackendImage);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function fileNameTableEntry(image: Image): string {
  return `${image.originalFileName}${image.computedFileName ? ` => ${image.computedFileName}` : ""}`;
}

function timeTableEntry(image: Image): string {
  if (image.cameraTime && image.correctedTime) {
    const cameraTimeString = dateTimeUtil.dateTimeFromUnix(image.cameraTime.toUnixInteger());
    const correctedTimeString = dateTimeUtil.dateTimeFromUnix(image.correctedTime.toUnixInteger());
    return `${cameraTimeString} => ${correctedTimeString}`;
  }
  return "-";
}

async function deleteItem(item: Image): Promise<void> {
  if (!item.id) {
    error("image cannot be deleted without an id");
    return;
  }

  try {
    await api.images.remove(item.id);
    uploadedImages.value = uploadedImages.value.filter((i) => i.id !== item.id);
    images.value = images.value.filter((i) => i.id !== item.id);
    showNotificationToast({ headline: `Image deleted`, type: "success" });
  } catch (err: any) {
    error("error deleting image", err);
    unexpectedError.value = err;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>

<script lang="ts">
export type InputFile = { name: string; data: ArrayBuffer; size: number };
</script>

<style scoped>
.thumbnail {
  position: relative;
  z-index: 1;
}

.tooltip {
  display: none;
  position: absolute;
  top: 0;
  left: 100%;
  margin-left: 10px;
  z-index: 10;
}

.thumbnail:hover + .tooltip {
  display: block;
}

.preview {
  max-width: 300px;
  max-height: 300px;
}

.arrow {
  position: absolute;
  top: 50%;
  left: -10px;
  margin-top: -5px;
  width: 0;
  height: 0;
  border-left: 10px solid transparent;
  border-right: 10px solid transparent;
  border-bottom: 10px solid #000;
}
</style>
