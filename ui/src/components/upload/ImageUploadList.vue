<template>
  <div class="mt-12">
    <div class="px-4 sm:px-6 lg:px-8">
      <div class="sm:flex sm:items-center">
        <div class="sm:flex-auto">
          <h1 class="text-base font-semibold leading-6 text-gray-900 dark:text-primary-200">Uploaded images</h1>
          <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">These images have been added to the upload and are either waiting for processing or have been processed.</p>
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
        <div class="-mx-4 -my-2 sm:-mx-6 lg:-mx-8">
          <div class="inline-block min-w-full py-2 align-middle">
            <table class="min-w-full border-separate border-spacing-0">
              <thead>
                <tr>
                  <th scope="col" :class="[tableHeaderClasses]">Preview</th>
                  <th scope="col" :class="[tableHeaderClasses]">Status</th>
                  <th scope="col" :class="[tableHeaderClasses]">Filename</th>
                  <th scope="col" :class="[tableHeaderClasses]">Time</th>
                  <th scope="col" :class="[tableHeaderClasses]">Size</th>
                  <th scope="col" :class="[tableHeaderClasses]">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="image in displayedImages">
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
                  <td :class="[tableCellClasses]">
                    <button v-if="image.status === ImageStatus.DONE" @click="deleteItem(image)" class="text-red-700 dark:text-red-300 hover:text-primary-900">Remove</button>
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
import { UploadsResponse, CamerasResponse, TimeOffsetsResponse, UsersResponse, ImagesResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { Image, ImageStatus, FileProcessor, newImage, newImageFromBackendImage, TimeOffsetResult } from "src/util/fileProcessor";
import { error } from "src/util/logger";
import { fileSize } from "src/util/fileUtil";

const tableHeaderClasses =
  "sticky top-0 z-10 border-b border-gray-300 dark:dark:border-primary-400 bg-gray-50 dark:bg-primary-900 bg-opacity-75 py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-gray-200 backdrop-blur backdrop-filter sm:pl-6 lg:pl-8";

const tableCellClasses = "whitespace-nowrap border-b border-gray-200 px-2 py-2 pr-3 text-sm font-medium text-gray-900 dark:text-gray-100 sm:pl-6 lg:pl-8";

type CameraType = CamerasResponse & { expand?: { time_offsets_via_camera: TimeOffsetsResponse[] } };
type UploadType = UploadsResponse & { expand?: { camera: CameraType; user: UsersResponse; images_via_upload: ImagesResponse[] } };

interface Props {
  upload: UploadType;
  files: File[];
}
const props = withDefaults(defineProps<Props>(), {});

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const upload = computed(() => props.upload);
const images = ref<Image[]>([]);
const uploadedImages = ref<Image[]>([]);

const displayedImages = computed(() => {
  return [...uploadedImages.value, ...images.value];
});

const timeOffsets = computed(() => {
  const cameraTimeOffsets = props.upload.expand?.camera.expand?.time_offsets_via_camera || [];
  return cameraTimeOffsets.map((timeOffset) => ({
    free: (): void => {},
    time_offset: BigInt(timeOffset.timeOffset),
    server_time: BigInt(dateTimeUtil.parseBackendTime(timeOffset.serverTime).getTime() / 1000),
    camera_time: BigInt(dateTimeUtil.parseBackendTime(timeOffset.cameraTime).getTime() / 1000),
  }));
});

const fileProcessor = new FileProcessor(upload, images, timeOffsets);
onUnmounted(() => fileProcessor.stop);

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
    const resultList = await pb.collection<ImagesResponse>("images").getList(1, 1000, {
      filter: `(upload='${upload.value.id}')`,
    });
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

function deleteItem(item: Image): void {
  if (!item.id) {
    error("image cannot be deleted without an id");
    return;
  }

  try {
    pb.collection<ImagesResponse>("images").delete(item.id);
    uploadedImages.value = uploadedImages.value.filter((i) => i.id !== item.id);
    images.value = images.value.filter((i) => i.id !== item.id);
    showNotificationToast({ headline: `Image deleted`, type: "success" });
  } catch (error: any) {
    error("error deleting image", error);
    unexpectedError.value = error;
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
