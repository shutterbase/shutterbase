<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div v-if="upload" class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
        <div class="border-b border-primary-200 dark:border-primary-800 pb-12">
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

          <p v-if="showUploadEdit(upload) && canAddMoreImages" class="mt-2 text-sm leading-6 text-primary-500 dark:text-primary-400">Edit this upload</p>
          <p v-else-if="showUploadEdit(upload)" class="mt-4 flex items-start gap-1.5 rounded-md border border-warning-200 bg-warning-50 px-2.5 py-2 text-xs text-warning-800 dark:border-warning-800/70 dark:bg-warning-950/40 dark:text-warning-200">
            <LockClosedIcon class="mt-px h-4 w-4 shrink-0" />
            <span>This upload is submitted for review — official tags are frozen and it takes no further images until a project admin sends it back.</span>
          </p>
          <FileDropzone v-if="showUploadEdit(upload) && canAddMoreImages" :multiple="true" @files="handleFiles" />
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
import { UploadsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { dateTimeFromUnix } from "src/util/dateTimeUtil";
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { TimeOffsetResult } from "src/util/fileProcessor";
import ImageUploadList, { InputFile } from "src/components/upload/ImageUploadList.vue";
import { isUploadReadOnly, showUploadEdit } from "./uploadUtil";
import { UploadState } from "src/types/api";
import { CheckCircleIcon, ClockIcon, LockClosedIcon, PencilSquareIcon } from "@heroicons/vue/24/outline";
import {
  UPLOAD_STATE_LABEL,
  UPLOAD_STATE_HINT,
  TRANSITION_LABEL,
  allowedTransitions,
  canAddImages,
} from "src/util/uploadReview";

const router = useRouter();
const route = useRoute();

const userStore = useUserStore();
const { activeProject } = storeToRefs(userStore);
const userId: string = userStore.user?.id || "";
const id: string = `${route.params.id}`;

type UploadType = UploadsResponse;
const upload = ref<UploadType | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function getUpload() {
  try {
    const result = await api.uploads.get(id);
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
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    movingTo.value = null;
  }
}

onMounted(getUpload);
</script>
