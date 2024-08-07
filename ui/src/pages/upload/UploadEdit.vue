<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div v-if="upload" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <div class="border-b border-gray-900/10 dark:border-gray-100/10 pb-12">
          <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-100">
            Upload <b>{{ upload.name }}</b>
          </h2>
          <p v-if="showUploadEdit(upload)" class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-400">Edit this upload</p>
          <FileDropzone v-if="showUploadEdit(upload)" :multiple="true" @files="handleFiles" />
        </div>
        <ImageUploadList :allow-edit="showUploadEdit(upload)" :upload="upload" :files="inputFiles" />
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import FileDropzone from "src/components/FileDropzone.vue";
import CreateGroup, { Field, FieldType, CreateData } from "src/components/CreateGroup.vue";
import { UploadsResponse, CamerasResponse, TimeOffsetsResponse, UsersResponse, ImagesResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { dateTimeFromUnix } from "src/util/dateTimeUtil";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { TimeOffsetResult } from "src/util/fileProcessor";
import ImageUploadList, { InputFile } from "src/components/upload/ImageUploadList.vue";
import { isUploadReadOnly, showUploadEdit } from "./uploadUtil";

const router = useRouter();
const route = useRoute();

const { activeProject } = storeToRefs(useUserStore());
const userId: string = pb.authStore.model?.id;
const id: string = `${route.params.id}`;

type CameraType = CamerasResponse & { expand?: { time_offsets_via_camera: TimeOffsetsResponse[] } };
type UploadType = UploadsResponse & { expand?: { camera: CameraType; user: UsersResponse; images_via_upload: ImagesResponse[] } };
const upload = ref<UploadType | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function getUpload() {
  try {
    const result = await pb.collection<UploadType>("uploads").getOne(id, {
      expand: "camera,camera.time_offsets_via_camera,user,images_via_upload",
    });
    upload.value = result;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const inputFiles = ref<File[]>([]);

async function handleFiles(fileInput: File[]) {
  inputFiles.value.push(...fileInput);
}

onMounted(getUpload);
</script>
