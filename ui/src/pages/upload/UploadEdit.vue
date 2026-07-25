<template>
  <div class="mx-auto max-w-7xl w-full lg:px-8">
    <main class="px-4 py-4 sm:px-6 lg:px-0">
      <div v-if="upload" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <h2 class="display text-2xl text-primary-900 dark:text-white">
          Upload <b class="font-semibold">{{ upload.name }}</b>
        </h2>

        <!-- Review state: the same flow the uploads kanban drives, on the page
             where the photographer actually works through the upload. -->
        <div v-if="reviewEnabled" class="mt-4 flex flex-wrap items-center gap-3">
          <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium', stateBadgeClasses]">
            <component :is="stateIcon" class="h-4 w-4" />
            {{ UPLOAD_STATE_LABEL[upload.state ?? "open"] }}
          </span>
          <span class="text-xs text-primary-500 dark:text-primary-400">{{ UPLOAD_STATE_HINT[upload.state ?? "open"] }}</span>
          <button
            v-for="next in transitions"
            :key="next"
            type="button"
            @click="moveTo(next)"
            :disabled="movingTo !== null"
            :class="[transitionBase, next === 'open' ? transitionQuiet : transitionAccent]"
          >
            {{ TRANSITION_LABEL[next] }}
          </button>
        </div>

        <p v-if="showUploadEdit(upload) && !canAddMoreImages" class="mt-4 flex items-start gap-1.5 rounded-md border border-warning-200 bg-warning-50 px-2.5 py-2 text-xs text-warning-800 dark:border-warning-800/70 dark:bg-warning-950/40 dark:text-warning-200">
          <LockClosedIcon class="mt-px h-4 w-4 shrink-0" />
          <span>This upload is submitted for review — official tags are frozen and it takes no further images until a project admin sends it back.</span>
        </p>

        <!-- 1. dropzone -->
        <FileDropzone v-if="showUploadEdit(upload) && canAddMoreImages" class="mt-6" :multiple="true" @files="handleFiles" />

        <!-- 2. tagging timeline -->
        <UploadTimeline :upload="upload" :images="displayedImages" :readonly="timelineReadonly" @applied="upload = $event" />

        <!-- 3. tile grid with per-tile progress -->
        <ImageUploadGrid :images="displayedImages" :allow-edit="showUploadEdit(upload)" @remove="deleteImage" />
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import FileDropzone from "src/components/FileDropzone.vue";
import ImageUploadGrid from "src/components/upload/ImageUploadGrid.vue";
import UploadTimeline from "src/components/upload/UploadTimeline.vue";
import { UploadsResponse } from "src/types/pocketbase";
import { TimeOffset, UploadState } from "src/types/api";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { FileProcessor, Image, newImage, newImageFromBackendImage } from "src/util/fileProcessor";
import { error } from "src/util/logger";
import { showUploadEdit } from "./uploadUtil";
import { CheckCircleIcon, ClockIcon, LockClosedIcon, PencilSquareIcon } from "@heroicons/vue/24/outline";
import { UPLOAD_STATE_LABEL, UPLOAD_STATE_HINT, TRANSITION_LABEL, allowedTransitions, canAddImages } from "src/util/uploadReview";

const route = useRoute();

const userStore = useUserStore();
const { activeProject } = storeToRefs(userStore);
const userId: string = userStore.user?.id || "";
const id: string = `${route.params.id}`;

const upload = ref<UploadsResponse | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function getUpload() {
  try {
    upload.value = await api.uploads.get(id);
    await Promise.all([requestImages(), requestTimeOffsets()]);
  } catch (err: any) {
    unexpectedError.value = err;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- image pipeline (was ImageUploadList's; lifted so the timeline sees the
// same image set as the grid) ---

const images = ref<Image[]>([]);
const uploadedImages = ref<Image[]>([]);
const displayedImages = computed(() => [...uploadedImages.value, ...images.value]);

const cameraTimeOffsets = ref<TimeOffset[]>([]);
const timeOffsets = computed(() => dateTimeUtil.toWasmTimeOffsets(cameraTimeOffsets.value));

const uploadForProcessor = computed(() => upload.value as UploadsResponse);
const fileProcessor = new FileProcessor(uploadForProcessor, images, timeOffsets);
onUnmounted(() => fileProcessor.stop());

async function handleFiles(fileInput: File[]) {
  for (const file of fileInput) {
    if (displayedImages.value.find((image) => image.originalFileName === file.name)) {
      continue;
    }
    images.value.push(newImage({ file }));
  }
  fileProcessor.start();
}

async function requestImages() {
  if (!upload.value) return;
  const resultList = await api.images.list({ projectId: upload.value.project.id, uploadId: upload.value.id, limit: 1000 });
  uploadedImages.value = resultList.items.map(newImageFromBackendImage);
}

async function requestTimeOffsets() {
  if (!upload.value?.camera?.id) return;
  cameraTimeOffsets.value = (await api.timeOffsets.list({ cameraId: upload.value.camera.id, limit: 50 })).items;
}

async function deleteImage(item: Image): Promise<void> {
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

// --- review state ---

const reviewEnabled = computed(() => !!activeProject.value?.uploadReviewEnabled);
const isReviewer = computed(() => userStore.isProjectAdminOrHigher());

const transitions = computed(() =>
  !upload.value || !reviewEnabled.value
    ? []
    : allowedTransitions(upload.value.state ?? "open", { isReviewer: isReviewer.value, isOwner: upload.value.user?.id === userId }),
);

const canAddMoreImages = computed(() =>
  canAddImages({ reviewEnabled: reviewEnabled.value, uploadState: upload.value?.state, isReviewer: isReviewer.value }),
);

// The timeline writes OFFICIAL (scheduled) tags, so it freezes under exactly
// the same review rule as adding images (server: CanApplyUploadTimeline).
const timelineReadonly = computed(() => !upload.value || !showUploadEdit(upload.value) || !canAddMoreImages.value);

const stateIcon = computed(() => {
  switch (upload.value?.state) {
    case "ready":
      return ClockIcon;
    case "reviewed":
      return CheckCircleIcon;
    default:
      return PencilSquareIcon;
  }
});

const stateBadgeClasses = computed(() => {
  switch (upload.value?.state) {
    case "ready":
      return "border-warning-300 bg-warning-50 text-warning-800 dark:border-warning-800/70 dark:bg-warning-950/40 dark:text-warning-200";
    case "reviewed":
      return "border-success-300 bg-success-50 text-success-800 dark:border-success-800/70 dark:bg-success-950/40 dark:text-success-200";
    default:
      return "border-primary-200 bg-surface text-primary-700 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200";
  }
});

const transitionBase =
  "inline-flex items-center justify-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50";
const transitionAccent = "bg-accent-600 text-white hover:bg-accent-500 active:bg-accent-700";
const transitionQuiet =
  "border border-primary-200 bg-surface text-primary-700 hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white";

const movingTo = ref<UploadState | null>(null);

async function moveTo(state: UploadState) {
  if (!upload.value) return;
  movingTo.value = state;
  try {
    upload.value = await api.uploads.update(upload.value.id, { state });
    showNotificationToast({ headline: `Upload moved to '${UPLOAD_STATE_LABEL[state]}'`, type: "success" });
  } catch (err: any) {
    unexpectedError.value = err;
    showUnexpectedErrorMessage.value = true;
  } finally {
    movingTo.value = null;
  }
}

onMounted(getUpload);
</script>
