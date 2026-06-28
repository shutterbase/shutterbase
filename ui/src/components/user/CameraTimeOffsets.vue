<template>
  <div v-if="timeOffsets.length !== 0" class="py-6 sm:flex">
    <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:pr-6">Time Offsets</dt>
    <dd class="mt-2 text-sm text-primary-700 dark:text-primary-300 sm:mt-0 sm:flex-auto">
      <ul role="list" class="divide-y divide-primary-100 rounded-md border border-primary-200 bg-surface dark:divide-primary-800 dark:border-primary-800 dark:bg-surface-dark">
        <li v-for="timeOffset in timeOffsets" :key="timeOffset.id" class="flex items-center justify-between py-4 pl-4 pr-5 text-sm leading-6">
          <div class="flex w-0 flex-1 items-center">
            <ClockIcon class="w-5 text-primary-400 dark:text-primary-500" />
            <div class="ml-4 flex min-w-0 flex-1 items-baseline gap-2">
              <span class="font-data truncate font-medium text-primary-900 dark:text-primary-100">{{ dateTimeFromBackend(timeOffset.serverTime) }}</span>
              <span class="font-data flex-shrink-0 text-primary-500 dark:text-primary-400">{{ timeOffset.timeOffset }} seconds</span>
            </div>
          </div>
          <div class="ml-4 flex-shrink-0">
            <button @click="deleteTimeOffset(timeOffset)" class="cursor-pointer font-medium text-error-600 transition-colors hover:text-error-500 dark:text-error-400">Delete</button>
          </div>
        </li>
      </ul>
    </dd>
  </div>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { dateTimeFromBackend } from "src/util/dateTimeUtil";
import { ClockIcon } from "@heroicons/vue/24/outline";
import { CamerasResponse, TimeOffsetsResponse } from "src/types/pocketbase";
import { onMounted, ref, watch } from "vue";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";

type CameraType = CamerasResponse;

interface Props {
  camera: CameraType;
}

const props = withDefaults(defineProps<Props>(), {});
const timeOffsets = ref<TimeOffsetsResponse[]>([]);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function setTimeOffsets() {
  if (!props.camera?.id) {
    timeOffsets.value = [];
    return;
  }
  timeOffsets.value = (await api.timeOffsets.list({ cameraId: props.camera.id, limit: 100 })).items;
}

async function deleteTimeOffset(timeOffset: TimeOffsetsResponse) {
  if (!timeOffset) {
    console.log("No time offset to delete");
    return;
  }

  try {
    await api.timeOffsets.remove(timeOffset.id);
    timeOffsets.value = timeOffsets.value.filter((e) => e.id !== timeOffset.id);
    showNotificationToast({ headline: `Time offset deleted`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(setTimeOffsets);
watch(() => props.camera, setTimeOffsets);
</script>
