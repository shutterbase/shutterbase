<template>
  <div v-if="timeOffsets.length !== 0" class="px-4 py-6 sm:flex sm:px-0">
    <dt class="sm:w-64 sm:pr-6 text-sm font-medium leading-6 text-gray-900 dark:text-primary-200">Time Offsets</dt>
    <dd class="sm:flex-auto mt-2 text-sm text-gray-900 dark:text-primary-200 sm:mt-0">
      <ul role="list" class="divide-y divide-gray-100 dark:divide-gray-700 rounded-md border border-gray-200 dark:border-gray-600">
        <li v-for="timeOffset in timeOffsets" :key="timeOffset.id" class="flex items-center justify-between py-4 pl-4 pr-5 text-sm leading-6">
          <div class="flex w-0 flex-1 items-center">
            <ClockIcon class="w-6" />
            <div class="ml-4 flex min-w-0 flex-1 gap-2">
              <span class="truncate font-medium">{{ dateTimeFromBackend(timeOffset.serverTime) }}</span>
              <span class="flex-shrink-0 text-gray-400">{{ timeOffset.timeOffset }} seconds</span>
            </div>
          </div>
          <div class="ml-4 flex-shrink-0">
            <button @click="deleteTimeOffset(timeOffset)" class="font-medium text-red-600 hover:text-red-500">Delete</button>
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
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";

type CameraType = CamerasResponse & { expand?: { time_offsets_via_camera: TimeOffsetsResponse[] } };

interface Props {
  camera: CameraType;
}

const props = withDefaults(defineProps<Props>(), {});
const timeOffsets = ref<TimeOffsetsResponse[]>([]);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function setTimeOffsets() {
  timeOffsets.value = props.camera.expand?.time_offsets_via_camera || [];
}

async function deleteTimeOffset(timeOffset: TimeOffsetsResponse) {
  if (!timeOffset) {
    console.log("No time offset to delete");
    return;
  }

  try {
    await pb.collection<TimeOffsetsResponse>("time_offsets").delete(timeOffset.id);
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
