<template>
  <!-- In-flow details panel: full width when stacked under the image on narrow
       viewports, a fixed 20rem column beside it from lg up. -->
  <!-- lg: capped to the viewport (app header + images toolbar + film strip
       clearance) and scrolls internally, so the panel never stretches the
       page. relative z-10 keeps it above the zoomed image stage (z-0). -->
  <div
    v-if="item"
    class="scrollbar-tool relative z-10 w-full shrink-0 self-start rounded-lg border border-primary-200 bg-surface text-primary-900 dark:border-primary-800 dark:bg-surface-dark dark:text-primary-200 lg:max-h-[calc(100vh-20rem)] lg:w-80 lg:overflow-y-auto"
  >
    <div class="p-4">
      <!-- sticky within the panel's own scroll; negative margins pull it flush
           against the container edges so content slides underneath, not above -->
      <h3
        class="display sticky top-0 z-10 -mx-4 -mt-4 bg-surface px-4 pt-4 text-lg text-primary-900 dark:bg-surface-dark dark:text-white pb-3 border-b border-primary-200 dark:border-primary-800"
      >
        Image Details
      </h3>
      <div class="border-b border-primary-200 dark:border-primary-800 py-4 space-y-2">
        <div>
          <p class="label-mono text-primary-500 dark:text-primary-400">Name</p>
          <p ref="nameRow" class="mt-0.5 flex items-center gap-1.5 font-data text-sm text-primary-800 dark:text-primary-100">
            <span class="truncate whitespace-nowrap" :style="nameStyle">{{ item.computedFileName }}</span>
            <!-- invisible measurer: full name width at base size, immune to the
                 shrink applied to the visible span (no feedback loop) -->
            <span ref="nameMeasure" aria-hidden="true" class="invisible absolute whitespace-pre font-data text-sm">{{ item.computedFileName }}</span>
            <Clipboard class="h-4 shrink-0" :text="item.computedFileName" />
          </p>
        </div>
        <div>
          <p class="label-mono text-primary-500 dark:text-primary-400">Corrected capture time</p>
          <p class="mt-0.5 font-data text-sm text-primary-800 dark:text-primary-100">{{ dateTimeFromBackend(item.capturedAtCorrected) }}</p>
        </div>
        <div>
          <p class="label-mono text-primary-500 dark:text-primary-400">Updated</p>
          <p class="mt-0.5 font-data text-sm text-primary-800 dark:text-primary-100">{{ dateTimeFromBackend(item.updatedAt) }}</p>
        </div>
        <template v-if="showAllDetails">
          <div>
            <p class="label-mono text-primary-500 dark:text-primary-400">Original capture time</p>
            <p class="mt-0.5 font-data text-sm text-primary-800 dark:text-primary-100">{{ dateTimeFromBackend(item.capturedAt) }}</p>
          </div>
          <div v-if="appliedOffset">
            <p class="label-mono text-primary-500 dark:text-primary-400">Applied time offset</p>
            <p class="mt-0.5 font-data text-sm text-primary-800 dark:text-primary-100">{{ appliedOffset }}</p>
          </div>
          <div>
            <p class="label-mono text-primary-500 dark:text-primary-400">ID</p>
            <p class="mt-0.5 flex items-center gap-1.5 font-data text-sm text-primary-800 dark:text-primary-100">
              <span class="truncate">{{ item.id }}</span>
              <Clipboard class="h-4 shrink-0" :text="item.id" />
            </p>
          </div>
          <div>
            <p class="label-mono text-primary-500 dark:text-primary-400">Original file name</p>
            <p class="mt-0.5 flex items-center gap-1.5 font-data text-sm text-primary-800 dark:text-primary-100">
              <span class="truncate">{{ item.fileName }}</span>
              <Clipboard class="h-4 shrink-0" :text="item.fileName" />
            </p>
          </div>
          <div>
            <p class="label-mono text-primary-500 dark:text-primary-400">Uploaded</p>
            <p class="mt-0.5 font-data text-sm text-primary-800 dark:text-primary-100">{{ dateTimeFromBackend(item.createdAt) }}</p>
            <p class="text-sm text-primary-500 dark:text-primary-400">by {{ item.user.firstName }} {{ item.user.lastName }}</p>
          </div>
          <p v-if="imageCanBeDeleted()" @click="showDeleteImageDialog" class="text-sm font-medium text-error-600 hover:text-error-500 dark:text-error-400 underline cursor-pointer">
            delete
          </p>
        </template>
        <div class="flex gap-4">
          <p @click="showAllDetails = !showAllDetails" class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer">
            {{ showAllDetails ? "less" : "more" }}
          </p>
          <p @click="copyPermalink" class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer">copy link</p>
        </div>
      </div>

      <div class="border-b border-primary-200 dark:border-primary-800 pb-4">
        <h3 class="label-mono text-primary-500 dark:text-primary-400 py-3">Image Tags</h3>
        <p
          v-if="officialTagsFrozen(item)"
          class="mb-3 flex items-start gap-1.5 rounded-md border border-warning-200 bg-warning-50 px-2.5 py-2 text-xs text-warning-800 dark:border-warning-800/70 dark:bg-warning-950/40 dark:text-warning-200"
        >
          <LockClosedIcon class="mt-px h-4 w-4 shrink-0" />
          <span>This upload is submitted for review — official tags are frozen. Custom tags can still be changed.</span>
        </p>
        <div class="space-y-2">
          <div v-for="group in tagGroups" :key="group.category">
            <p class="label-mono text-[0.6rem] text-primary-400 dark:text-primary-500">{{ group.category }}</p>
            <div class="mt-1 flex flex-wrap gap-2">
              <ImageTagBadge
                v-for="tagAssignment in group.assignments"
                :key="tagAssignment.id"
                :tagAssignment="tagAssignment"
                :removable="canRemoveTagAssignment(item, tagAssignment)"
                @remove="(ta) => removeTagAssignment(item, ta)"
              />
            </div>
          </div>
        </div>
        <p
          v-if="tagsCanBeAdded()"
          @click="() => emitter.emit('show-tagging-dialog')"
          class="mt-4 inline-block text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer"
        >
          add
        </p>
      </div>
      <div class="border-b border-primary-200 dark:border-primary-800 pb-4">
        <h3 class="label-mono text-primary-500 dark:text-primary-400 py-3">AI Detection</h3>
        <div class="flex items-center gap-2">
          <SparklesIcon v-if="item.aiStatus === 'done'" class="h-4 w-4 text-accent-500" />
          <ArrowPathIcon v-else-if="item.aiStatus === 'processing'" class="h-4 w-4 animate-spin text-accent-500" />
          <ClockIcon v-else-if="item.aiStatus === 'pending'" class="h-4 w-4 text-primary-400" />
          <ExclamationTriangleIcon v-else-if="item.aiStatus === 'error'" class="h-4 w-4 text-error-500" />
          <p class="font-data text-sm text-primary-800 dark:text-primary-100">{{ aiStatusText }}</p>
        </div>
        <p v-if="item.aiStatus === 'error' && item.aiError" class="mt-1 truncate text-xs text-error-600 dark:text-error-400" :title="item.aiError">{{ item.aiError }}</p>
        <div class="mt-3 flex flex-wrap gap-x-4 gap-y-1">
          <p @click="() => emitter.emit('ai-toggle-faces')" class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer">
            faces
          </p>
          <p @click="() => emitter.emit('ai-show-similar')" class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer">
            similar images
          </p>
          <p @click="showDetectionResult" class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer">detection result</p>
          <p
            v-if="userStore.isProjectEditorOrHigher()"
            @click="rerunAi"
            class="text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer"
          >
            rerun
          </p>
        </div>
      </div>

      <div class="border-b border-primary-200 dark:border-primary-800 pb-4">
        <h3 class="label-mono text-primary-500 dark:text-primary-400 py-3">Download Links</h3>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="resolution in ['original', '2048', '1024', '512', '256']"
            :key="resolution"
            class="inline-flex items-center rounded-md border border-primary-200 bg-surface px-2 py-0.5 font-data text-xs font-medium text-primary-700 transition-colors hover:border-accent-400 hover:text-accent-700 cursor-pointer dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-accent-400/60 dark:hover:text-accent-200"
            @click="() => downloadImage(item, resolution)"
            >{{ resolution }}</span
          >
        </div>
      </div>
      <div class="pb-1">
        <h3 class="label-mono text-primary-500 dark:text-primary-400 py-3">Infos</h3>
        <div class="flex items-center gap-2 pt-1">
          <svg class="h-5 fill-primary-500 dark:fill-primary-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M480-260q75 0 127.5-52.5T660-440q0-75-52.5-127.5T480-620q-75 0-127.5 52.5T300-440q0 75 52.5 127.5T480-260Zm0-80q-42 0-71-29t-29-71q0-42 29-71t71-29q42 0 71 29t29 71q0 42-29 71t-71 29ZM160-120q-33 0-56.5-23.5T80-200v-480q0-33 23.5-56.5T160-760h126l74-80h240l74 80h126q33 0 56.5 23.5T880-680v480q0 33-23.5 56.5T800-120H160Zm0-80h640v-480H638l-73-80H395l-73 80H160v480Zm320-240Z"
            />
          </svg>
          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.exifData["Model"] }}</p>
        </div>
        <div class="flex items-center gap-2 pt-1">
          <svg class="h-5 fill-primary-500 dark:fill-primary-400" viewBox="0 0 14 14" role="img" focusable="false" aria-hidden="true" xmlns="http://www.w3.org/2000/svg">
            <path
              d="m 10.006823,5.68736 c -0.5360739,0 -0.8659762,0.46734 -0.8659762,1.27832 0,0.81098 0.3299023,1.31956 0.8659642,1.31956 0.536061,0 0.865963,-0.50858 0.865963,-1.31956 0,-0.81098 -0.329877,-1.27832 -0.865951,-1.27832 z m 2.742329,-2.26398 -11.4983043,0 C 1.112317,3.42337 1,3.53568 1,3.67422 l 0,6.65156 c 0,0.13855 0.112317,0.25085 0.2508477,0.25085 l 11.4983043,0 C 12.887696,10.57663 13,10.46433 13,10.32578 L 13,3.67422 C 13,3.53567 12.887696,3.42337 12.749152,3.42337 Z m -9.4912848,5.79655 -1.182107,0 0,-4.46726 1.182107,0 0,4.46726 z m 2.4054158,0.0825 c -0.5635669,0 -1.2095874,-0.20618 -1.6906756,-0.63916 L 4.6461209,7.8523 c 0.3161433,0.25427 0.7147653,0.43298 1.044655,0.43298 0.3573827,0 0.5085811,-0.11684 0.5085811,-0.31615 0,-0.21304 -0.2268039,-0.28177 -0.6048062,-0.43297 L 5.0378445,7.30242 C 4.5567689,7.10998 4.1375272,6.7045 4.1375272,6.05846 c 0,-0.76286 0.6872724,-1.38829 1.6631952,-1.38829 0.5085811,0 1.0721355,0.19245 1.4844914,0.59793 L 6.6941665,6.01036 C 6.3917697,5.79732 6.1305871,5.68735 5.8007224,5.68735 c -0.2886629,0 -0.4673542,0.10308 -0.4673542,0.30239 0,0.21303 0.2542842,0.28864 0.6597795,0.44671 L 6.5360823,6.6495 c 0.5567062,0.21992 0.8728495,0.60479 0.8728495,1.22334 2.51e-5,0.75603 -0.632274,1.42954 -1.7456488,1.42954 z m 4.34354,0 c -1.2371053,0 -2.0755511,-0.86595 -2.0755511,-2.33672 0,-1.47076 0.8384709,-2.29549 2.0755511,-2.29549 1.23708,0 2.075539,0.8316 2.075539,2.29549 2.5e-5,1.47077 -0.838446,2.33672 -2.075539,2.33672 z"
            />
          </svg>
          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.exifData["PhotographicSensitivity"] }}</p>
        </div>
        <div class="flex items-center gap-2 pt-1">
          <svg class="h-5 fill-primary-500 dark:fill-primary-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M360-840v-80h240v80H360ZM480-80q-75 0-140.5-28.5T225-186q-49-49-77-114.5T120-440q0-74 28.5-139.5T226-694q49-49 114.5-77.5T480-800q63 0 120 21t104 59l58-58 56 56-56 58q36 47 57 104t21 120q0 74-28 139.5T735-186q-49 49-114.5 77.5T480-80Zm0-360Zm0-80h268q-18-62-61.5-109T584-700L480-520Zm-70 40 134-232q-59-15-121.5-2.5T306-660l104 180Zm-206 80h206L276-632q-42 47-62.5 106.5T204-400Zm172 220 104-180H212q18 62 61.5 109T376-180Zm40 12q66 17 128 1.5T654-220L550-400 416-168Zm268-80q44-48 63.5-107.5T756-480H550l134 232Z"
            />
          </svg>
          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.exifData["ExposureTime"] }} s</p>
        </div>
        <div class="flex items-center gap-2 pt-1">
          <svg class="h-5 fill-primary-500 dark:fill-primary-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M456-600h320q-27-69-82.5-118.5T566-788L456-600Zm-92 80 160-276q-11-2-22-3t-22-1q-66 0-123 25t-101 67l108 188ZM170-400h218L228-676q-32 41-50 90.5T160-480q0 21 2.5 40.5T170-400Zm224 228 108-188H184q27 69 82.5 118.5T394-172Zm86 12q66 0 123-25t101-67L596-440 436-164q11 2 21.5 3t22.5 1Zm252-124q32-41 50-90.5T800-480q0-21-2.5-40.5T790-560H572l160 276ZM480-480Zm0 400q-82 0-155-31.5t-127.5-86Q143-252 111.5-325T80-480q0-83 31.5-155.5t86-127Q252-817 325-848.5T480-880q83 0 155.5 31.5t127 86q54.5 54.5 86 127T880-480q0 82-31.5 155t-86 127.5q-54.5 54.5-127 86T480-80Z"
            />
          </svg>
          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.exifData["FocalLength"] }}mm @ f{{ item.exifData["FNumber"] }}</p>
        </div>
        <div class="flex items-center gap-2 pt-1">
          <svg class="h-5 w-5 stroke-primary-500 dark:stroke-primary-400" viewBox="0 0 24 24" fill="none" stroke-width="1.8">
            <circle cx="12" cy="12" r="9" />
            <circle cx="12" cy="12" r="4.5" />
          </svg>
          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.exifData["LensModel"] }}</p>
        </div>
        <div v-if="item.width && item.height" class="flex items-center gap-2 pt-1">
          <svg class="h-5 fill-primary-500 dark:fill-primary-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15"
            />
          </svg>

          <p class="ml-2 font-data text-sm text-primary-700 dark:text-primary-200">{{ item.width }}px x {{ item.height }}px</p>
        </div>
      </div>
    </div>
    <div v-if="showAiResult" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" @click.self="showAiResult = false">
      <div class="flex max-h-[80vh] w-full max-w-2xl flex-col rounded-lg border border-primary-200 bg-surface shadow-xl dark:border-primary-700 dark:bg-surface-dark">
        <div class="flex items-center justify-between border-b border-primary-200 px-4 py-3 dark:border-primary-800">
          <h3 class="label-mono text-primary-500 dark:text-primary-400">AI Detection Result</h3>
          <button type="button" class="cursor-pointer text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400" @click="showAiResult = false">close</button>
        </div>
        <pre class="overflow-auto whitespace-pre-wrap break-all px-4 py-3 font-data text-xs leading-relaxed text-primary-800 dark:text-primary-100">{{ aiResultText }}</pre>
      </div>
    </div>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <ModalMessage
      :show="showDeleteDialog"
      :type="MessageType.CONFIRM_WARNING"
      @closed="showDeleteDialog = false"
      headline="Delete Image"
      :message="`Are you sure you want to delete image '${deleteCandidate?.computedFileName}'?`"
      @confirmed="confirmDeleteImage"
    />
  </div>
</template>

<script setup lang="ts">
import { ImageWithTagsType } from "src/types/custom";
import { dateTimeFromBackend } from "src/util/dateTimeUtil";
import ImageTagBadge from "src/components/image/ImageTagBadge.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { downloadImage } from "src/util/download";
import { api } from "src/api";
import { useUserStore } from "src/stores/user-store";
import { ImagesResponse } from "src/types/pocketbase";
import Clipboard from "src/components/Clipboard.vue";
import { LockClosedIcon } from "@heroicons/vue/24/outline";
import { ArrowPathIcon, ClockIcon, ExclamationTriangleIcon, SparklesIcon } from "@heroicons/vue/24/solid";
import { officialTagsFrozen } from "src/pages/upload/uploadUtil";
import { appliedTimeOffset } from "src/util/dateTimeUtil";
import { canRemoveTagAssignment, removeTagAssignment } from "src/util/imageTags";
import { groupTagAssignments } from "src/util/tagOrder";
import { aiPositions, aiQueueTotal } from "src/pages/image/imageQueryLogic";
import { useRouter } from "vue-router";
import { copyToClipboard } from "src/util/clipboard";

const userStore = useUserStore();

const showDeleteDialog = ref(false);
const deleteCandidate = ref<ImageWithTagsType | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

interface Props {
  item: ImageWithTagsType | null;
}
const props = withDefaults(defineProps<Props>(), {});

// details collapse to name + corrected capture time; "more" reveals the rest
const showAllDetails = ref(false);

// permalink: the canonical minimal deep link — no filter/sort context, so the
// recipient's resolver (jumpToImage) decides how to present it
const router = useRouter();
function copyPermalink() {
  if (!props.item) return;
  const href = router.resolve({ name: "images", query: { image: props.item.id } }).href;
  copyToClipboard(`${window.location.origin}${href}`);
  showNotificationToast({ headline: "Link copied", type: "success" });
}

// Auto-shrink the name to a single line: compare the hidden measurer's width
// (full name at base 14px) against the row minus the clipboard icon, scale the
// font down proportionally. Floor at 9px — below that truncation takes over.
const nameRow = ref<HTMLElement | null>(null);
const nameMeasure = ref<HTMLElement | null>(null);
const nameStyle = ref<{ fontSize?: string }>({});
function fitName() {
  const row = nameRow.value;
  const measure = nameMeasure.value;
  if (!row || !measure) return;
  const available = row.clientWidth - 24; // ponytail: clipboard icon + gap, static
  const full = measure.offsetWidth;
  nameStyle.value = full > available ? { fontSize: `${Math.max((available / full) * 14, 9)}px` } : {};
}
watch(() => props.item?.computedFileName, fitName, { flush: "post" });
let nameResizeObserver: ResizeObserver | null = null;
onMounted(() => {
  fitName();
  nameResizeObserver = new ResizeObserver(fitName);
  if (nameRow.value) nameResizeObserver.observe(nameRow.value);
});
onUnmounted(() => nameResizeObserver?.disconnect());
const appliedOffset = computed(() => {
  if (!props.item?.capturedAt || !props.item?.capturedAtCorrected) return null;
  return appliedTimeOffset(props.item.capturedAt, props.item.capturedAtCorrected);
});

const tagAssignments = computed(() => {
  return props.item?.tags || [];
});

// One row per tag category (template, manual, custom, ai), each row ordered by
// the tags' rank — same ordering the EXIF export applies.
const tagGroups = computed(() => groupTagAssignments(tagAssignments.value));

const aiStatusText = computed(() => {
  const item = props.item;
  if (!item) return "";
  switch (item.aiStatus) {
    case "pending": {
      const position = aiPositions.value[item.id];
      return position ? `queued — position ${position} of ${aiQueueTotal.value}` : "queued";
    }
    case "processing":
      return "detecting…";
    case "done":
      return item.inferredAt ? `done (${dateTimeFromBackend(item.inferredAt)})` : "done";
    case "error":
      return "failed";
    default:
      return "not run";
  }
});

const showAiResult = ref(false);
const aiResultText = ref("");
async function showDetectionResult() {
  if (!props.item) return;
  try {
    const res = await api.ai.result(props.item.id);
    aiResultText.value = JSON.stringify(res.raw, null, 2);
    showAiResult.value = true;
  } catch (error: any) {
    if (error?.response?.status === 404) {
      showNotificationToast({ headline: "No detection result stored for this image", type: "info" });
      return;
    }
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function rerunAi() {
  if (!props.item) return;
  try {
    await api.ai.rerunImage(props.item.id);
    props.item.aiStatus = "pending";
    showNotificationToast({ headline: "AI detection queued", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// Custom tags stay editable on a submitted upload, so the add affordance stays —
// addImageTag refuses the individual frozen tags.
function tagsCanBeAdded(): boolean {
  return userStore.isProjectEditorOrHigher() && (userStore.isProjectAdminOrHigher() || props.item?.user.id === userStore.user?.id);
}

function imageCanBeDeleted(): boolean {
  return userStore.isProjectAdminOrHigher() || props.item?.user.id === userStore.user?.id;
}

function showDeleteImageDialog() {
  showDeleteDialog.value = true;
  deleteCandidate.value = props.item;
}
async function confirmDeleteImage() {
  showDeleteDialog.value = false;

  if (!deleteCandidate.value) {
    console.error("No image selected for deletion");
    return;
  }

  try {
    await api.images.remove(deleteCandidate.value.id);
    showNotificationToast({ headline: `Image deleted`, type: "success" });
    emitter.emit("current-image-deleted", deleteCandidate.value.id);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
