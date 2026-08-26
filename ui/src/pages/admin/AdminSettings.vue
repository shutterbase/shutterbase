<template>
  <main class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl space-y-12">
      <div>
        <h2 class="text-lg font-medium text-primary-900 dark:text-white">Platform Settings</h2>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">Global configuration for the Shutterbase instance</p>
      </div>

      <!-- MQTT / WLED Integration -->
      <div class="rounded-lg border border-primary-200 bg-surface p-6 dark:border-primary-700 dark:bg-surface-dark">
        <div class="flex items-center gap-3">
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
            {{ mqttConnected ? 'Connected' : 'Disconnected' }}
          </div>
          <h3 class="text-base font-medium text-primary-900 dark:text-white">MQTT / WLED Integration</h3>
        </div>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
          Publish upload events to an MQTT broker for WLED and other smart-home devices.
        </p>

        <form @submit.prevent="saveMqttSettings" class="mt-6 space-y-4">
          <div>
            <label for="mqtt-broker" class="block text-sm font-medium text-primary-700 dark:text-primary-300">Broker URL</label>
            <input
              id="mqtt-broker"
              v-model="mqttForm.broker"
              type="text"
              placeholder="tcp://localhost:1883"
              class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
            />
          </div>
          <div>
            <label for="mqtt-clientid" class="block text-sm font-medium text-primary-700 dark:text-primary-300">Client ID</label>
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
              <label for="mqtt-username" class="block text-sm font-medium text-primary-700 dark:text-primary-300">Username</label>
              <input
                id="mqtt-username"
                v-model="mqttForm.username"
                type="text"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
            <div>
              <label for="mqtt-password" class="block text-sm font-medium text-primary-700 dark:text-primary-300">Password</label>
              <input
                id="mqtt-password"
                v-model="mqttForm.password"
                type="password"
                class="mt-1 block w-full rounded-md border border-primary-300 bg-white px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
            </div>
          </div>
          <div>
            <label for="mqtt-topicprefix" class="block text-sm font-medium text-primary-700 dark:text-primary-300">Topic Prefix</label>
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
          <div class="flex justify-end">
            <button
              type="submit"
              :disabled="saving"
              class="inline-flex items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3.5 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
            >
              <ArrowPathIcon v-if="saving" class="h-4 w-4 animate-spin" />
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showError" :error="error" @closed="showError = false" />
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";

const mqttConnected = ref(false);
const saving = ref(false);
const showError = ref(false);
const error = ref(null);

const mqttForm = ref({
  broker: "",
  clientId: "",
  username: "",
  password: "",
  topicPrefix: "",
});

onMounted(async () => {
  try {
    const [settings, status] = await Promise.all([
      api.adminSettings.getMqttSettings(),
      api.mqtt.getStatus(),
    ]);
    mqttForm.value = settings;
    mqttConnected.value = status.connected;
  } catch (e: any) {
    error.value = e;
    showError.value = true;
  }
});

async function saveMqttSettings() {
  saving.value = true;
  try {
    await api.adminSettings.updateMqttSettings(mqttForm.value);
    const status = await api.mqtt.getStatus();
    mqttConnected.value = status.connected;
    showNotificationToast({ headline: "MQTT settings saved", type: "success" });
  } catch (e: any) {
    error.value = e;
    showError.value = true;
  } finally {
    saving.value = false;
  }
}
</script>
