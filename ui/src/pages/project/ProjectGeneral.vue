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
      <!-- MQTT / WLED integration -->
      <div v-if="userStore.isProjectAdminOrHigher()">
        <div class="flex items-center gap-3">
          <h3 class="text-lg font-medium text-primary-900 dark:text-white">MQTT / WLED Integration</h3>
          <div
            :class="[
              'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
              mqttConfigured
                ? mqttConnected
                  ? 'border-success-400/60 bg-success-500/10 text-success-700 dark:text-success-300'
                  : 'border-warning-400/60 bg-warning-500/10 text-warning-700 dark:text-warning-300'
                : 'border-primary-300 bg-transparent text-primary-400 dark:border-primary-700 dark:text-primary-500',
            ]"
          >
            <span
              :class="[
                'h-1.5 w-1.5 rounded-full',
                mqttConfigured ? (mqttConnected ? 'bg-success-500' : 'bg-warning-500') : 'bg-primary-400',
              ]"
            ></span>
            {{ mqttConfigured ? (mqttConnected ? 'Connected' : 'Disconnected') : 'Not configured' }}
          </div>
        </div>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">Publish upload events to an MQTT broker for WLED and other smart-home devices</p>

        <form @submit.prevent="saveMqttSettings" class="mt-4 space-y-6">
          <!-- Broker Connection -->
          <div class="space-y-4">
            <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Broker Connection</h4>
            <div>
              <label for="mqtt-broker" class="block text-sm text-primary-600 dark:text-primary-400">Broker URL</label>
              <input
                id="mqtt-broker"
                v-model="mqttForm.broker"
                type="text"
                placeholder="tcp://localhost:1883"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
            <div>
              <label for="mqtt-clientid" class="block text-sm text-primary-600 dark:text-primary-400">Client ID</label>
              <input
                id="mqtt-clientid"
                v-model="mqttForm.clientId"
                type="text"
                placeholder="shutterbase"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="mqtt-username" class="block text-sm text-primary-600 dark:text-primary-400">Username</label>
                <input
                  id="mqtt-username"
                  v-model="mqttForm.username"
                  type="text"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
              </div>
              <div>
                <label for="mqtt-password" class="block text-sm text-primary-600 dark:text-primary-400">Password</label>
                <input
                  id="mqtt-password"
                  v-model="mqttForm.password"
                  type="password"
                  class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
              </div>
            </div>
            <div>
              <label for="mqtt-topicprefix" class="block text-sm text-primary-600 dark:text-primary-400">Topic Prefix</label>
              <input
                id="mqtt-topicprefix"
                v-model="mqttForm.topicPrefix"
                type="text"
                placeholder="shutterbase"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
              <p class="mt-1 text-xs text-primary-400 dark:text-primary-500">
                Topics: <code class="font-mono">{{ mqttForm.topicPrefix || 'shutterbase' }}/{projectId}/upload/{uploadId}/{event}</code>
              </p>
            </div>
            <div>
              <label for="mqtt-wledtopic" class="block text-sm text-primary-600 dark:text-primary-400">WLED Device Topic</label>
              <input
                id="mqtt-wledtopic"
                v-model="mqttForm.wledDeviceTopic"
                type="text"
                placeholder="wled/device1"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
              <p class="mt-1 text-xs text-primary-400 dark:text-primary-500">
                Optional. When set, preset commands are also published to <code class="font-mono">{{ mqttForm.wledDeviceTopic || 'wled/device1' }}/api</code> for direct WLED control.
              </p>
            </div>
          </div>

          <!-- Event Toggles -->
          <div class="space-y-4">
            <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Events</h4>
            <p class="text-xs text-primary-500 dark:text-primary-400">Select which events trigger MQTT messages. Set a WLED preset number (0 = no preset).</p>
            <div class="space-y-3">
              <div v-for="event in mqttEventList" :key="event.key" class="flex items-center gap-4">
                <label class="flex items-center gap-2 min-w-[200px]">
                  <input
                    type="checkbox"
                    v-model="mqttForm.events[event.key]"
                    class="h-4 w-4 rounded border-primary-300 text-accent-500 focus:ring-accent-500"
                  />
                  <span class="text-sm text-primary-700 dark:text-primary-300">{{ event.label }}</span>
                </label>
                <input
                  v-if="mqttForm.events[event.key]"
                  v-model.number="mqttForm.presets[event.key]"
                  type="number"
                  min="0"
                  placeholder="Preset #"
                  class="w-24 rounded-md border border-primary-300 bg-white px-2 py-1 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
                />
                <span v-if="mqttForm.events[event.key]" class="text-xs text-primary-400 dark:text-primary-500">WLED preset</span>
              </div>
            </div>
          </div>

          <!-- Tag Triggers -->
          <div class="space-y-4">
            <h4 class="text-sm font-medium text-primary-700 dark:text-primary-300">Tag Triggers</h4>
            <p class="text-xs text-primary-500 dark:text-primary-400">When "Tag assigned" is enabled above, specify which tag names trigger an MQTT message.</p>
            <input
              v-model="mqttTriggerTagsInput"
              type="text"
              placeholder="error, vip, highlight (comma-separated)"
              class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
            />
          </div>

          <div class="flex justify-end">
            <button
              type="submit"
              :disabled="savingMqtt"
              class="inline-flex items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            >
              <ArrowPathIcon v-if="savingMqtt" class="h-4 w-4 animate-spin" />
              Save MQTT Settings
            </button>
          </div>
        </form>
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
const mqttConfigured = ref(false);
const mqttConnected = ref(false);
const savingMqtt = ref(false);
const mqttTriggerTagsInput = ref("");

const mqttEventList = [
  { key: "uploadCreated", label: "Upload created" },
  { key: "imageUploaded", label: "Image uploaded" },
  { key: "ready", label: "Ready for review" },
  { key: "approved", label: "Approved" },
  { key: "rejected", label: "Rejected / sent back" },
  { key: "imageRejected", label: "Image rejected (tag)" },
  { key: "tagAssigned", label: "Tag assigned" },
];

const mqttForm = ref({
  broker: "",
  clientId: "",
  username: "",
  password: "",
  topicPrefix: "",
  wledDeviceTopic: "",
  events: {
    uploadCreated: false,
    imageUploaded: false,
    ready: false,
    approved: false,
    rejected: false,
    imageRejected: false,
    tagAssigned: false,
  },
  presets: {
    uploadCreated: 0,
    imageUploaded: 0,
    ready: 0,
    approved: 0,
    rejected: 0,
    imageRejected: 0,
    tagAssigned: 0,
  },
});

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
  loadMqttSettings();
});

async function loadMqttSettings() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  try {
    const [settings, status] = await Promise.all([
      api.adminSettings.getProjectMqttSettings(projectId),
      api.mqtt.getProjectMqttStatus(projectId),
    ]);
    mqttForm.value.broker = settings.broker;
    mqttForm.value.clientId = settings.clientId;
    mqttForm.value.username = settings.username;
    mqttForm.value.password = settings.password;
    mqttForm.value.topicPrefix = settings.topicPrefix;
    mqttForm.value.wledDeviceTopic = settings.wledDeviceTopic;
    mqttForm.value.events = settings.events;
    mqttForm.value.presets = settings.presets;
    mqttTriggerTagsInput.value = settings.triggerTags?.join(", ") || "";
    mqttConfigured.value = status.configured;
    mqttConnected.value = status.connected;
  } catch {
    // MQTT not configured for this project
  }
}

async function saveMqttSettings() {
  const projectId = `${route.params.id}`;
  if (!projectId) return;
  savingMqtt.value = true;
  try {
    const triggerTags = mqttTriggerTagsInput.value
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t.length > 0);
    await api.adminSettings.updateProjectMqttSettings(projectId, {
      ...mqttForm.value,
      triggerTags,
    });
    const status = await api.mqtt.getProjectMqttStatus(projectId);
    mqttConfigured.value = status.configured;
    mqttConnected.value = status.connected;
    showNotificationToast({ headline: "MQTT settings saved", type: "success" });
  } catch (e: any) {
    unexpectedError.value = e;
    showUnexpectedErrorMessage.value = true;
  } finally {
    savingMqtt.value = false;
  }
}
</script>
