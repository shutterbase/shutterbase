<template>
  <div
    v-if="item"
    class="max-2xl:hidden w-80 top-16 fixed inset-y-0 left-0 bg-gray-50 dark:bg-primary-900 text-gray-900 dark:text-gray-200 shadow-lg z-10 overflow-y-scroll no-scrollbar"
  >
    <div class="p-5">
      <h3 class="text-lg font-medium pb-6 border-b dark:border-primary-400">Image Details</h3>
      <div class="border-b py-6 dark:border-primary-400">
        <div class="pb-2">
          <p class="text-sm font-medium">Name</p>
          <p class="text-sm">
            {{ item.computedFileName }}
            <Clipboard class="h-4" :text="item.computedFileName" />
          </p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">ID</p>
          <p class="text-sm">
            {{ item.id }}
            <Clipboard class="h-4" :text="item.id" />
          </p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Original file name</p>
          <p class="text-sm">
            {{ item.fileName }}
            <Clipboard class="h-4" :text="item.fileName" />
          </p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Corrected capture time</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.capturedAtCorrected) }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Original capture time</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.capturedAt) }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Uploaded</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.created) }}</p>
          <p class="text-sm">by {{ item.expand.user.firstName }} {{ item.expand.user.lastName }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Updated</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.updated) }}</p>
        </div>
        <p v-if="imageCanBeDeleted()" @click="showDeleteImageDialog" class="text-sm text-bold underline cursor-pointer">delete</p>
      </div>

      <div class="border-b pb-6 dark:border-primary-400">
        <h3 class="text-lg font-medium py-6">Image Tags</h3>
        <div class="flex">
          <ImageTagBadge
            class="mr-2 mb-2"
            v-for="tagAssignment in tagAssignments"
            :key="tagAssignment.id"
            :tagAssignment="tagAssignment"
            :removable="removable(tagAssignment)"
            @remove="removeTag"
          />
        </div>
        <p v-if="tagsCanBeAdded()" @click="() => emitter.emit('show-tagging-dialog')" class="mt-4 p-2 text-sm text-bold underline cursor-pointer">add</p>
      </div>
      <div class="border-b pb-6 dark:border-primary-400">
        <h3 class="text-lg font-medium py-6">Download Links</h3>
        <div class="flex">
          <p v-for="resolution in ['original', '2048', '1024', '512', '256']">
            <span
              :class="[
                `inline-flex items-center rounded-md px-2 mr-2 py-1 text-xs font-medium ring-1 ring-inset cursor-pointer`,
                `bg-gray-200 dark:bg-gray-800 text-gray-900 dark:text-gray-100 ring-gray-200 dark:ring-gray-700`,
              ]"
              @click="() => downloadImage(item, resolution)"
              >{{ resolution }}</span
            >
          </p>
        </div>
      </div>
      <div class="border-b pb-6 dark:border-primary-400">
        <h3 class="text-lg font-medium py-6">Infos</h3>
        <div class="flex">
          <svg class="h-5 fill-gray-700 dark:fill-gray-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M480-260q75 0 127.5-52.5T660-440q0-75-52.5-127.5T480-620q-75 0-127.5 52.5T300-440q0 75 52.5 127.5T480-260Zm0-80q-42 0-71-29t-29-71q0-42 29-71t71-29q42 0 71 29t29 71q0 42-29 71t-71 29ZM160-120q-33 0-56.5-23.5T80-200v-480q0-33 23.5-56.5T160-760h126l74-80h240l74 80h126q33 0 56.5 23.5T880-680v480q0 33-23.5 56.5T800-120H160Zm0-80h640v-480H638l-73-80H395l-73 80H160v480Zm320-240Z"
            />
          </svg>
          <p class="ml-2">{{ item.exifData["Model"] }}</p>
        </div>
        <div class="flex">
          <svg class="h-5 fill-gray-700 dark:fill-gray-300" viewBox="0 0 14 14" role="img" focusable="false" aria-hidden="true" xmlns="http://www.w3.org/2000/svg">
            <path
              d="m 10.006823,5.68736 c -0.5360739,0 -0.8659762,0.46734 -0.8659762,1.27832 0,0.81098 0.3299023,1.31956 0.8659642,1.31956 0.536061,0 0.865963,-0.50858 0.865963,-1.31956 0,-0.81098 -0.329877,-1.27832 -0.865951,-1.27832 z m 2.742329,-2.26398 -11.4983043,0 C 1.112317,3.42337 1,3.53568 1,3.67422 l 0,6.65156 c 0,0.13855 0.112317,0.25085 0.2508477,0.25085 l 11.4983043,0 C 12.887696,10.57663 13,10.46433 13,10.32578 L 13,3.67422 C 13,3.53567 12.887696,3.42337 12.749152,3.42337 Z m -9.4912848,5.79655 -1.182107,0 0,-4.46726 1.182107,0 0,4.46726 z m 2.4054158,0.0825 c -0.5635669,0 -1.2095874,-0.20618 -1.6906756,-0.63916 L 4.6461209,7.8523 c 0.3161433,0.25427 0.7147653,0.43298 1.044655,0.43298 0.3573827,0 0.5085811,-0.11684 0.5085811,-0.31615 0,-0.21304 -0.2268039,-0.28177 -0.6048062,-0.43297 L 5.0378445,7.30242 C 4.5567689,7.10998 4.1375272,6.7045 4.1375272,6.05846 c 0,-0.76286 0.6872724,-1.38829 1.6631952,-1.38829 0.5085811,0 1.0721355,0.19245 1.4844914,0.59793 L 6.6941665,6.01036 C 6.3917697,5.79732 6.1305871,5.68735 5.8007224,5.68735 c -0.2886629,0 -0.4673542,0.10308 -0.4673542,0.30239 0,0.21303 0.2542842,0.28864 0.6597795,0.44671 L 6.5360823,6.6495 c 0.5567062,0.21992 0.8728495,0.60479 0.8728495,1.22334 2.51e-5,0.75603 -0.632274,1.42954 -1.7456488,1.42954 z m 4.34354,0 c -1.2371053,0 -2.0755511,-0.86595 -2.0755511,-2.33672 0,-1.47076 0.8384709,-2.29549 2.0755511,-2.29549 1.23708,0 2.075539,0.8316 2.075539,2.29549 2.5e-5,1.47077 -0.838446,2.33672 -2.075539,2.33672 z"
            />
          </svg>
          <p class="ml-2">{{ item.exifData["PhotographicSensitivity"] }}</p>
        </div>
        <div class="flex">
          <svg class="h-5 fill-gray-700 dark:fill-gray-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M360-840v-80h240v80H360ZM480-80q-75 0-140.5-28.5T225-186q-49-49-77-114.5T120-440q0-74 28.5-139.5T226-694q49-49 114.5-77.5T480-800q63 0 120 21t104 59l58-58 56 56-56 58q36 47 57 104t21 120q0 74-28 139.5T735-186q-49 49-114.5 77.5T480-80Zm0-360Zm0-80h268q-18-62-61.5-109T584-700L480-520Zm-70 40 134-232q-59-15-121.5-2.5T306-660l104 180Zm-206 80h206L276-632q-42 47-62.5 106.5T204-400Zm172 220 104-180H212q18 62 61.5 109T376-180Zm40 12q66 17 128 1.5T654-220L550-400 416-168Zm268-80q44-48 63.5-107.5T756-480H550l134 232Z"
            />
          </svg>
          <p class="ml-2">{{ item.exifData["ExposureTime"] }} s</p>
        </div>
        <div class="flex">
          <svg class="h-5 fill-gray-700 dark:fill-gray-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960">
            <path
              d="M456-600h320q-27-69-82.5-118.5T566-788L456-600Zm-92 80 160-276q-11-2-22-3t-22-1q-66 0-123 25t-101 67l108 188ZM170-400h218L228-676q-32 41-50 90.5T160-480q0 21 2.5 40.5T170-400Zm224 228 108-188H184q27 69 82.5 118.5T394-172Zm86 12q66 0 123-25t101-67L596-440 436-164q11 2 21.5 3t22.5 1Zm252-124q32-41 50-90.5T800-480q0-21-2.5-40.5T790-560H572l160 276ZM480-480Zm0 400q-82 0-155-31.5t-127.5-86Q143-252 111.5-325T80-480q0-83 31.5-155.5t86-127Q252-817 325-848.5T480-880q83 0 155.5 31.5t127 86q54.5 54.5 86 127T880-480q0 82-31.5 155t-86 127.5q-54.5 54.5-127 86T480-80Z"
            />
          </svg>
          <p class="ml-2">{{ item.exifData["FocalLength"] }}mm @ f{{ item.exifData["FNumber"] }}</p>
        </div>
        <div class="flex">
          <span class="h-5 w-5"></span>
          <p class="ml-2">{{ item.exifData["LensModel"] }}</p>
        </div>
        <div v-if="item.width && item.height" class="flex">
          <svg class="h-5 fill-gray-700 dark:fill-gray-300" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15"
            />
          </svg>

          <p class="ml-2">{{ item.width }}px x {{ item.height }}px</p>
        </div>
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
import { ImageTagAssignmentType, ImageWithTagsType } from "src/types/custom";
import { dateTimeFromBackend } from "src/util/dateTimeUtil";
import ImageTagBadge from "src/components/image/ImageTagBadge.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ref, computed } from "vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { downloadImage } from "src/util/download";
import pb from "src/boot/pocketbase";
import { dateTimeToBackendString } from "src/util/dateTimeUtil";
import { useUserStore } from "src/stores/user-store";
import { ImagesResponse } from "src/types/pocketbase";
import Clipboard from "src/components/Clipboard.vue";

const userStore = useUserStore();

const showDeleteDialog = ref(false);
const deleteCandidate = ref<ImageWithTagsType | null>(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

interface Props {
  item: ImageWithTagsType | null;
}
const props = withDefaults(defineProps<Props>(), {});

const tagAssignments = computed(() => {
  return props.item?.expand.image_tag_assignments_via_image || [];
});

function removable(tagAssignment: ImageTagAssignmentType): boolean {
  const isOwnImage = props.item?.user === userStore.user.id;
  const isProjectAdminOrHigher = userStore.isProjectAdminOrHigher();
  if (tagAssignment.expand.imageTag.type === "default") {
    return isProjectAdminOrHigher;
  } else {
    return isOwnImage || isProjectAdminOrHigher;
  }
}

function tagsCanBeAdded(): boolean {
  return userStore.isProjectAdminOrHigher() || props.item?.user === userStore.user.id;
}

function imageCanBeDeleted(): boolean {
  return userStore.isProjectAdminOrHigher() || props.item?.user === userStore.user.id;
}

async function removeTag(tagAssignment: ImageTagAssignmentType) {
  if (!removable(tagAssignment)) {
    return;
  }
  try {
    await pb.collection("image_tag_assignments").delete(tagAssignment.id);
    emitter.emit(`notification`, {
      headline: `Tag ${tagAssignment.expand.imageTag.name} removed`,
      type: "success",
    });
    if (props.item) {
      props.item.expand.image_tag_assignments_via_image.splice(
        props.item.expand.image_tag_assignments_via_image.findIndex((ta) => ta.id === tagAssignment.id),
        1
      );
      props.item.updated = dateTimeToBackendString(new Date());
    }
  } catch (error: any) {
    emitter.emit(`notification`, {
      headline: `Error removing tag ${tagAssignment.expand.imageTag.name}`,
      type: "error",
    });
  }
}

function showDeleteImageDialog() {
  showDeleteDialog.value = true;
  deleteCandidate.value = props.item;
}
function confirmDeleteImage() {
  showDeleteDialog.value = false;

  if (!deleteCandidate.value) {
    console.error("No image selected for deletion");
    return;
  }

  try {
    pb.collection<ImagesResponse>("images").delete(deleteCandidate.value.id);
    showNotificationToast({ headline: `Image deleted`, type: "success" });
    emitter.emit("current-image-deleted", deleteCandidate.value.id);
  } catch (error: any) {
    error("error deleting image", error);
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
<style>
/* Hide scrollbar for Chrome, Safari and Opera */
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

/* Hide scrollbar for IE, Edge and Firefox */
.no-scrollbar {
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
}
</style>
