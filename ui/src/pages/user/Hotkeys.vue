<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="sm:flex sm:items-center mb-6">
      <div class="sm:flex-auto">
        <h1 class="text-base font-semibold leading-6 text-gray-900 dark:text-gray-100">
          <span v-if="isOwnProfile">Your Hotkeys</span>
          <span v-else>Hotkeys of {{ fullName() }}</span>
        </h1>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">View and customize hotkey mappings. Clearing a custom mapping restores the default.</p>
      </div>
      <div class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none"></div>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-300 dark:divide-gray-700 text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-gray-800">
            <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Description</th>
            <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Event</th>
            <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Default</th>
            <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Your Hotkey</th>
            <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300 w-40">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
          <tr v-for="row in rows" :key="row.eventId" class="hover:bg-gray-50 dark:hover:bg-gray-800">
            <td class="px-3 py-2">
              <div class="text-gray-900 dark:text-gray-100">{{ row.description }}</div>
            </td>
            <td class="px-3 py-2 font-mono text-xs text-gray-600 dark:text-gray-400">{{ row.event }}</td>
            <td class="px-3 py-2">
              <span v-if="row.defaultHotkey" class="inline-block rounded bg-gray-200 dark:bg-gray-700 px-2 py-0.5 text-xs font-mono">
                {{ row.defaultHotkey }}
              </span>
              <span v-else class="text-xs italic text-gray-400">—</span>
            </td>
            <td class="px-3 py-2">
              <span
                v-if="row.userHotkey"
                class="inline-block rounded bg-secondary-200 dark:bg-secondary-700 px-2 py-0.5 text-xs font-mono"
                :title="row.userHotkeyDiffers ? 'Overrides default' : ''"
              >
                {{ row.userHotkey }}
              </span>
              <span v-else class="text-xs italic text-gray-400">—</span>
            </td>
            <td class="px-3 py-2 space-x-2">
              <button
                type="button"
                class="rounded-md bg-secondary-600 hover:bg-secondary-500 dark:hover:bg-secondary-700 px-2 py-1 text-xs font-medium text-white shadow-sm"
                @click="openCapture(row)"
              >
                {{ row.userHotkey ? "Change" : "Set" }}
              </button>
              <button
                v-if="row.mappingId"
                type="button"
                class="rounded-md bg-error-600 hover:bg-error-500 px-2 py-1 text-xs font-medium text-white shadow-sm"
                @click="clearMapping(row)"
              >
                Clear
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="rows.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">No hotkey events found.</div>
    </div>
  </main>

  <HotkeyCaptureDialog
    :show="captureDialogVisible"
    :initialHotkey="activeRow?.userHotkey || activeRow?.defaultHotkey || ''"
    @closed="captureDialogVisible = false"
    @save="saveCapturedHotkey"
  />

  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import pb from "src/boot/pocketbase";
import { HotkeyEventsResponse, HotkeyMappingsResponse, Collections } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import HotkeyCaptureDialog from "src/components/user/HotkeyCaptureDialog.vue";
import { useRoute } from "vue-router";
import { showNotificationToast } from "src/boot/mitt";
import { fullName } from "src/util/userUtil";

const route = useRoute();
const userId: string = `${route.params.userid}`;
const isOwnProfile = computed(() => userId === pb.authStore.model?.id);

interface HotkeyRow {
  eventId: string;
  event: string;
  description: string;
  defaultHotkey: string;
  mappingId?: string;
  userHotkey?: string;
  userHotkeyDiffers: boolean;
}

const rows = ref<HotkeyRow[]>([]);
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref<any>(null);

const captureDialogVisible = ref(false);
const activeRow = ref<HotkeyRow | null>(null);

async function loadData() {
  try {
    const events = await pb.collection<HotkeyEventsResponse>(Collections.HotkeyEvents).getFullList({
      sort: "event",
    });
    const mappings = await pb.collection<HotkeyMappingsResponse>(Collections.HotkeyMappings).getFullList({
      filter: `user='${userId}'`,
    });

    const mappingByEvent: Record<string, HotkeyMappingsResponse> = {};
    for (const m of mappings) {
      if (m.event) {
        mappingByEvent[m.event] = m as HotkeyMappingsResponse;
      }
    }

    rows.value = events.map((e) => {
      const map = mappingByEvent[e.id];
      const userHotkey = map?.hotkey;
      return {
        eventId: e.id,
        event: e.event,
        description: e.description,
        defaultHotkey: e.defaultHotkey || "",
        mappingId: map?.id,
        userHotkey,
        userHotkeyDiffers: !!userHotkey && userHotkey !== (e.defaultHotkey || ""),
      } as HotkeyRow;
    });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function openCapture(row: HotkeyRow) {
  activeRow.value = row;
  captureDialogVisible.value = true;
}

async function saveCapturedHotkey(hotkey: string) {
  if (!activeRow.value) return;
  const row = activeRow.value;
  try {
    if (row.mappingId) {
      const updated = await pb.collection<HotkeyMappingsResponse>(Collections.HotkeyMappings).update(row.mappingId, {
        hotkey,
      });
      row.userHotkey = updated.hotkey;
    } else {
      const created = await pb.collection<HotkeyMappingsResponse>(Collections.HotkeyMappings).create({
        user: userId,
        event: row.eventId,
        hotkey,
      });
      row.mappingId = created.id;
      row.userHotkey = created.hotkey;
    }
    row.userHotkeyDiffers = !!row.userHotkey && row.userHotkey !== row.defaultHotkey;
    showNotificationToast({ headline: "Hotkey saved", type: "success" });
    captureDialogVisible.value = false;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function clearMapping(row: HotkeyRow) {
  if (!row.mappingId) return;
  const mappingId = row.mappingId;
  try {
    await pb.collection(Collections.HotkeyMappings).delete(mappingId);
    row.mappingId = undefined;
    row.userHotkey = undefined;
    row.userHotkeyDiffers = false;
    showNotificationToast({ headline: "Hotkey cleared", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(loadData);
</script>
