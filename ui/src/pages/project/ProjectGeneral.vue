<template>
  <main class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl space-y-12">
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Project Information"
        subtitle="General information concerning this project"
        :fields="informationFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Event Period"
        subtitle="Frames the schedule calendar — from first to last event day"
        :fields="periodFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Copyright Data"
        subtitle="Copyright information to be embedded into the EXIF data"
        :fields="copyrightFields"
        :item="item"
      />
      <div>
        <DetailEditGroup
          :allow-edit="userStore.isProjectAdminOrHigher()"
          @edit-save="saveItem"
          headline="AI Options"
          subtitle="Options for AI image tagging"
          :fields="aiFields"
          :item="item"
        />
        <div v-if="userStore.isProjectAdminOrHigher()" class="mt-4 flex flex-wrap gap-3">
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="rerunFailed"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Re-queue failed images
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showRecomputeConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Recompute all images
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showRerunNumbersConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Re-read car numbers
          </button>
          <button
            type="button"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            :disabled="rerunningFailed || rerunningAll || rerunningNumbers"
            @click="showReclusterConfirm = true"
          >
            <ArrowPathIcon class="h-4 w-4" />
            Recluster faces
          </button>
        </div>
      </div>
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Upload Review"
        subtitle="Let photographers submit uploads for review before their tags are final"
        :fields="reviewFields"
        :item="item"
      />
      <!-- MQTT / WLED integration status -->
      <div>
        <h3 class="text-lg font-medium text-primary-900 dark:text-white">Integrations</h3>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">External service connections</p>
        <div class="mt-4 flex items-center gap-3">
          <div
            :class="[
              'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm font-medium',
              mqttConnected
                ? 'border-success-400/60 bg-success-500/10 text-success-700 dark:text-success-300'
                : 'border-primary-300 bg-transparent text-primary-400 dark:border-primary-700 dark:text-primary-500',
            ]"
          >
            <span
              :class="[
                'h-2 w-2 rounded-full',
                mqttConnected ? 'bg-success-500' : 'bg-primary-400',
              ]"
            ></span>
            WLED / MQTT
          </div>
          <span class="text-xs text-primary-400 dark:text-primary-500">
            {{ mqttConnected ? 'Connected' : 'Not configured' }}
          </span>
        </div>
      </div>
    </div>
  </main>
  <ModalMessage
    :show="showRecomputeConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Recompute all images?"
    message="Every image of this project is re-queued for AI detection. Existing AI tags, descriptions and face data are replaced, and the full run costs AI credits."
    confirmText="Recompute all"
    cancelText="Cancel"
    @confirmed="rerunAll"
    @closed="showRecomputeConfirm = false"
  />
  <ModalMessage
    :show="showRerunNumbersConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Re-read car numbers?"
    message="Every image of this project is re-queued for a car-number re-read with the currently configured AI model. Number and scene tags are recomputed; faces, similarity data and descriptions are kept. Cheaper than a full recompute, but the run still costs AI credits."
    confirmText="Re-read numbers"
    cancelText="Cancel"
    @confirmed="rerunNumbers"
    @closed="showRerunNumbersConfirm = false"
  />
  <ModalMessage
    :show="showReclusterConfirm"
    :type="MessageType.CONFIRM_WARNING"
    headline="Recluster faces?"
    message="Person clusters are rebuilt from the existing face data — no AI credits are used. All cluster merges and merge decisions are discarded, and the review queue starts fresh. This affects every project, since face clusters are shared."
    confirmText="Recluster"
    cancelText="Cancel"
    @confirmed="recluster"
    @closed="showReclusterConfirm = false"
  />
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
import { useUserStore } from "src/stores/user-store";
const route = useRoute();

const userStore = useUserStore();

type ITEM_TYPE = ProjectsResponse;
const ITEM_COLLECTION = "projects";
const ITEM_NAME = "project";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const mqttConnected = ref(false);

async function loadItem() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await api.projects.get(itemId);
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(editData: EditData<ITEM_TYPE>) {
  if (!item.value) {
    console.log(`No ${ITEM_NAME} to save`);
    return;
  }

  const rollbackData = { ...item.value };
  item.value = { ...item.value, ...editData };

  try {
    console.log(`Saving ${ITEM_NAME} ${item.value.id}`);
    const response = await api.projects.update(item.value.id, editData as Partial<ITEM_TYPE>);
    item.value = response;
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} saved`, type: "success" });
  } catch (error: any) {
    item.value = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const rerunningFailed = ref(false);
async function rerunFailed() {
  if (!item.value) return;
  rerunningFailed.value = true;
  try {
    const queued = await api.ai.rerunFailed(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No failed images to re-queue" : `AI detection re-queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningFailed.value = false;
  }
}

const rerunningAll = ref(false);
const showRecomputeConfirm = ref(false);
async function rerunAll() {
  if (!item.value) return;
  showRecomputeConfirm.value = false;
  rerunningAll.value = true;
  try {
    const queued = await api.ai.rerunAll(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No images to recompute" : `AI detection re-queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningAll.value = false;
  }
}

const rerunningNumbers = ref(false);
const showRerunNumbersConfirm = ref(false);
async function rerunNumbers() {
  if (!item.value) return;
  showRerunNumbersConfirm.value = false;
  rerunningNumbers.value = true;
  try {
    const queued = await api.ai.rerunNumbers(item.value.id);
    showNotificationToast({
      headline: queued === 0 ? "No images to re-read" : `Car-number re-read queued for ${queued} image${queued === 1 ? "" : "s"}`,
      type: "success",
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    rerunningNumbers.value = false;
  }
}

const showReclusterConfirm = ref(false);
async function recluster() {
  if (!item.value) return;
  showReclusterConfirm.value = false;
  try {
    await api.ai.recluster(item.value.id);
    showNotificationToast({ headline: "Recluster started — clusters repopulate on the Faces page", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ITEM_TYPE>[] = [
  { key: "name", label: "Name", type: FieldType.TEXT },
  { key: "description", label: "Description", type: FieldType.TEXT },
];

const aiFields: Field<ITEM_TYPE>[] = [{ key: "aiSystemMessage", label: "System Message", type: FieldType.TEXT }];

const periodFields: Field<ITEM_TYPE>[] = [
  { key: "startAt", label: "Starts", type: FieldType.DATETIME },
  { key: "endAt", label: "Ends", type: FieldType.DATETIME },
];

const reviewFields: Field<ITEM_TYPE>[] = [{ key: "uploadReviewEnabled", label: "Upload reviews", type: FieldType.BOOLEAN, hint: "Enable the open / ready / reviewed flow" }];

const copyrightFields: Field<ITEM_TYPE>[] = [
  { key: "copyright", label: "Copyright", type: FieldType.TEXT },
  { key: "copyrightReference", label: "Copyright reference", type: FieldType.TEXT },
  { key: "copyrightTagPrefix", label: "Copyright tag prefix", type: FieldType.TEXT, hint: "Prepended to the photographer's copyright tag in exported EXIF only, e.g. by_" },
  { key: "locationName", label: "Location name", type: FieldType.TEXT },
  { key: "locationCode", label: "Location code", type: FieldType.TEXT },
  { key: "locationCity", label: "Location city", type: FieldType.TEXT },
];

watch(route, loadItem);
onMounted(() => {
  loadItem();
  api.mqtt.getStatus().then((s) => (mqttConnected.value = s.connected)).catch(() => {});
});
</script>
