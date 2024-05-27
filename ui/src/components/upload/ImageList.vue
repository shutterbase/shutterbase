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
                  <th scope="col" :class="[tableHeaderClasses]">Status</th>
                  <th scope="col" :class="[tableHeaderClasses]">Original Filename</th>
                  <th scope="col" :class="[tableHeaderClasses]">Size</th>
                  <th scope="col" :class="[tableHeaderClasses]">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="image in images">
                  <td :class="[tableCellClasses]">{{ image.status }}</td>
                  <td :class="[tableCellClasses]">{{ image.originalFileName }}</td>
                  <td :class="[tableCellClasses]">{{ image.size }}</td>
                  <td :class="[tableCellClasses]">
                    <a href="#" class="text-red-700 dark:text-red-300 hover:text-primary-900">Remove</a>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { UploadsResponse, CamerasResponse, TimeOffsetsResponse, UsersResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { dateTimeFromUnix } from "src/util/dateTimeUtil";
import * as fileUtil from "src/util/fileUtil";
import { Image, ImageStatus } from "src/util/uploadUtil";

const tableHeaderClasses =
  "sticky top-0 z-10 border-b border-gray-300 dark:dark:border-primary-400 bg-gray-50 dark:bg-primary-900 bg-opacity-75 py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-gray-200 backdrop-blur backdrop-filter sm:pl-6 lg:pl-8";

const tableCellClasses = "whitespace-nowrap border-b border-gray-200 px-2 py-2 pr-3 text-sm font-medium text-gray-900 dark:text-gray-100 sm:pl-6 lg:pl-8";

type UploadType = UploadsResponse & { expand?: { camera: CamerasResponse; user: UsersResponse } };

interface Props {
  upload: UploadType;
  files: File[];
}
const props = withDefaults(defineProps<Props>(), {});

const images = ref<Image[]>([]);

async function updateFiles(files: File[]) {
  for (const file of files) {
    images.value.push({ status: ImageStatus.PENDING, file, originalFileName: file.name, data: null, size: file.size });
  }
}

watch(props, (props) => {
  updateFiles(props.files);
});
</script>

<script lang="ts">
export type InputFile = { name: string; data: ArrayBuffer; size: number };
</script>
