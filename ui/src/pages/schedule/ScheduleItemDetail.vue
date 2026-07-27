<template>
  <div class="mx-auto w-full max-w-4xl px-4 pb-16 pt-6 sm:px-6 lg:px-8">
    <router-link :to="{ name: 'schedule' }" class="inline-flex items-center gap-1.5 text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400">
      <ArrowLeftIcon class="h-4 w-4" />
      Schedule
    </router-link>

    <div v-if="item" class="mt-4">
      <!-- block header -->
      <div class="rounded-lg border border-primary-200 bg-surface p-5 dark:border-primary-800 dark:bg-surface-dark">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h1 class="display text-2xl text-primary-900 dark:text-white">{{ item.title }}</h1>
            <p class="mt-1 text-sm tabular-nums text-primary-500 dark:text-primary-400">{{ windowLabel(item) }}</p>
            <p v-if="item.description" class="mt-2 max-w-prose text-sm text-primary-600 dark:text-primary-300">{{ item.description }}</p>
          </div>
          <div class="flex items-center gap-2">
            <span :class="['rounded-full border px-2.5 py-1 text-xs font-medium', OCCUPANCY_CLASSES[status]]">{{ OCCUPANCY_LABEL[status] }}</span>
            <button v-if="canManage" type="button" class="btn-secondary" @click="editOpen = true">
              <PencilIcon class="h-4 w-4" />
              Edit
            </button>
          </div>
        </div>
      </div>

      <!-- shifts -->
      <div class="mt-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="text-lg font-semibold text-primary-900 dark:text-white">Shifts</h2>
          <div v-if="canManage" class="flex flex-wrap gap-2">
            <button
              v-for="minutes in [60, 90, 120]"
              :key="minutes"
              type="button"
              class="btn-secondary"
              :disabled="!nextShiftWindow(item, minutes)"
              @click="quickAdd(minutes, 'item')"
            >
              <PlusIcon class="h-4 w-4" />
              {{ minutes }} min
            </button>
            <button type="button" class="btn-secondary" :disabled="!nextShiftWindow(item, 60)" @click="quickAdd(60, 'break')">
              <PauseIcon class="h-4 w-4" />
              Break
            </button>
          </div>
        </div>

        <p v-if="!shifts.length" class="mt-4 text-sm text-primary-400">
          No shifts yet — the whole block is claimed as one piece.
          <template v-if="canManage">Add shifts to subdivide it.</template>
        </p>

        <ul class="mt-3 space-y-2">
          <li
            v-for="shift in shifts"
            :key="shift.id"
            :data-testid="`shift-${shift.id}`"
            :class="[
              'rounded-lg border p-4',
              shift.kind === 'break'
                ? 'border-dashed border-primary-300 bg-primary-100/50 dark:border-primary-700 dark:bg-primary-900/40'
                : 'border-primary-200 bg-surface dark:border-primary-800 dark:bg-surface-dark',
            ]"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
                <span v-if="shift.kind === 'break'" class="label-mono-sm inline-flex items-center gap-1 text-primary-500 dark:text-primary-400">
                  <PauseIcon class="h-4 w-4" /> Break
                </span>
                <template v-if="canManage">
                  <input :value="drafts[shift.id]?.start" type="datetime-local" class="input-field w-auto" @change="commitWindow(shift, 'start', $event)" />
                  <span class="text-primary-400">–</span>
                  <input :value="drafts[shift.id]?.end" type="datetime-local" class="input-field w-auto" @change="commitWindow(shift, 'end', $event)" />
                </template>
                <span v-else class="text-sm tabular-nums text-primary-700 dark:text-primary-200">{{ windowLabel(shift) }}</span>
                <span class="label-mono-sm text-primary-400">{{ durationLabel(shift) }}</span>
                <label v-if="canManage && shift.kind !== 'break'" class="inline-flex items-center gap-1.5 text-xs text-primary-500 dark:text-primary-400">
                  target
                  <input :value="shift.cardinality" type="number" min="1" class="input-field w-16" @change="commitCardinality(shift, $event)" />
                </label>
                <span v-if="shift.kind !== 'break'" :class="['rounded-full border px-2 py-0.5 text-[10px] font-medium', OCCUPANCY_CLASSES[shiftStatus(shift)]]">
                  {{ shift.assignees.length }}/{{ shift.cardinality }}
                </span>
              </div>
              <button
                v-if="canManage"
                type="button"
                class="flex h-7 w-7 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400"
                :aria-label="`Delete ${shift.kind === 'break' ? 'break' : 'shift'}`"
                @click="removeShift(shift)"
              >
                <TrashIcon class="h-4 w-4" />
              </button>
            </div>

            <div v-if="shift.kind !== 'break'" class="mt-3 flex flex-wrap items-center gap-3">
              <div class="flex items-center -space-x-1.5">
                <UserBubble v-for="assignee in shift.assignees" :key="assignee.id" :user="assignee" />
              </div>
              <span v-if="!shift.assignees.length" class="text-sm text-primary-400">Nobody yet — be the first.</span>
              <div class="ml-auto flex flex-wrap gap-2">
                <button v-if="canJoin" type="button" :class="isAssigned(shift, userStore.user?.id) ? 'btn-secondary' : 'btn-primary'" @click="toggleClaim(shift)">
                  <UserMinusIcon v-if="isAssigned(shift, userStore.user?.id)" class="h-4 w-4" />
                  <UserPlusIcon v-else class="h-4 w-4" />
                  {{ isAssigned(shift, userStore.user?.id) ? "Leave" : "Take this shift" }}
                </button>
                <button v-if="canManage" type="button" class="btn-secondary" @click="assignShiftId = shift.id">
                  <UserGroupIcon class="h-4 w-4" />
                  Assign
                </button>
              </div>
            </div>
            <ul v-if="canManage && shift.assignees.length" class="mt-2 space-y-1">
              <li v-for="assignee in shift.assignees" :key="assignee.id" class="flex items-center justify-between gap-2 text-sm text-primary-600 dark:text-primary-300">
                <span class="truncate">{{ assignee.firstName }} {{ assignee.lastName }}</span>
                <button
                  type="button"
                  class="flex h-6 w-6 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400"
                  :aria-label="`Remove ${assignee.firstName} ${assignee.lastName}`"
                  @click="unassign(shift, assignee.id)"
                >
                  <XMarkIcon class="h-4 w-4" />
                </button>
              </li>
            </ul>
          </li>
        </ul>
      </div>
    </div>

    <AssignPhotographerModal :show="assignShiftId !== null" :candidates="assignCandidates" @closed="assignShiftId = null" @assign="assignFromModal" />
    <ScheduleItemDialog :show="editOpen" :create="false" :item="item" :project-tags="projectTags" @closed="editOpen = false" @save="saveBlock" @deleted="deleteBlock" />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
// Detail page of one schedule block: edit the frame (admins), subdivide it
// into shifts and breaks (quick-add 60/90/120, freely editable windows), and
// claim individual shifts. Reached from the calendar (subdivided blocks open
// here; plain items via the popover's "Shifts" button).
import { ArrowLeftIcon, PauseIcon, PencilIcon, PlusIcon, TrashIcon, UserGroupIcon, UserMinusIcon, UserPlusIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { computed, onBeforeUnmount, onMounted, ref, Ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "src/api";
import { ScheduleItemCreate, ScheduleItemUpdate } from "src/api/scheduleItems";
import AssignPhotographerModal from "src/components/schedule/AssignPhotographerModal.vue";
import ScheduleItemDialog from "src/components/schedule/ScheduleItemDialog.vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { EmbeddedUser, ImageTag, ScheduleItem } from "src/types/api";
import { OCCUPANCY_CLASSES, OCCUPANCY_LABEL, blockStatus, isAssigned, nextShiftWindow, occupancyStatus } from "src/util/schedule";
import * as websocket from "src/util/websocket";

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();

const canManage = computed(() => userStore.isProjectAdminOrHigher());
const canJoin = computed(() => userStore.isProjectEditorOrHigher());

const item: Ref<ScheduleItem | null> = ref(null);
const projectTags: Ref<ImageTag[]> = ref([]);
const members: Ref<EmbeddedUser[]> = ref([]);
const editOpen = ref(false);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const shifts = computed(() => item.value?.shifts ?? []);
const status = computed(() => (item.value ? blockStatus(item.value) : "empty"));
const shiftStatus = (s: ScheduleItem) => occupancyStatus(s.assignees.length, s.cardinality);

// datetime-local drafts per shift, refreshed on every (re)load
const drafts = ref<Record<string, { start: string; end: string }>>({});
const toLocal = (iso: string) => DateTime.fromISO(iso).toFormat("yyyy-MM-dd'T'HH:mm");
const toISO = (local: string) => DateTime.fromISO(local).toUTC().toISO() ?? "";

function windowLabel(it: { start: string; end: string }): string {
  const s = DateTime.fromISO(it.start);
  const e = DateTime.fromISO(it.end);
  return s.hasSame(e, "day") ? `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("HH:mm")}` : `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("ccc dd.LL. HH:mm")}`;
}

function durationLabel(it: { start: string; end: string }): string {
  const minutes = DateTime.fromISO(it.end).diff(DateTime.fromISO(it.start), "minutes").minutes;
  return `${Math.round(minutes)} min`;
}

async function load() {
  try {
    item.value = await api.scheduleItems.get(route.params.id as string);
    drafts.value = Object.fromEntries((item.value.shifts ?? []).map((s) => [s.id, { start: toLocal(s.start), end: toLocal(s.end) }]));
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function loadContext() {
  if (!canManage.value || !item.value) return;
  try {
    const [tags, assignments] = await Promise.all([
      api.imageTags.list({ projectId: item.value.project.id, limit: 500, sort: "name", order: "asc" }),
      api.projectAssignments.list({ projectId: item.value.project.id, limit: 500 }),
    ]);
    projectTags.value = tags.items;
    const seen = new Map<string, EmbeddedUser>();
    assignments.items.forEach((a) => seen.set(a.user.id, a.user));
    members.value = [...seen.values()];
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// API errors here are mostly rule violations (shift_outside_block …) — show
// the server's message instead of the generic error modal.
function toastApiError(error: any) {
  const message = error?.response?.data?.message;
  if (message) {
    showNotificationToast({ headline: message, type: "warning" });
  } else {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- shift management (admins) ----------------------------------------------

async function quickAdd(minutes: number, kind: "item" | "break") {
  if (!item.value) return;
  const window = nextShiftWindow(item.value, minutes);
  if (!window) return;
  try {
    await api.scheduleItems.create({
      title: kind === "break" ? "Break" : `Shift ${shifts.value.filter((s) => s.kind !== "break").length + 1}`,
      start: window.start.toISOString(),
      end: window.end.toISOString(),
      projectId: item.value.project.id,
      parentId: item.value.id,
      kind,
    });
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

async function commitWindow(shift: ScheduleItem, field: "start" | "end", event: Event) {
  const local = (event.target as HTMLInputElement).value;
  if (!local) return;
  try {
    await api.scheduleItems.update(shift.id, { [field]: toISO(local) } as ScheduleItemUpdate);
  } catch (error: any) {
    toastApiError(error);
  }
  await load(); // reload either way — a rejected edit resets the input
}

async function commitCardinality(shift: ScheduleItem, event: Event) {
  const value = Number((event.target as HTMLInputElement).value);
  if (!Number.isInteger(value) || value < 1) return;
  try {
    await api.scheduleItems.update(shift.id, { cardinality: value });
  } catch (error: any) {
    toastApiError(error);
  }
  await load();
}

async function removeShift(shift: ScheduleItem) {
  try {
    await api.scheduleItems.remove(shift.id);
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

// --- claiming ---------------------------------------------------------------

async function toggleClaim(shift: ScheduleItem) {
  const userId = userStore.user?.id;
  if (!userId) return;
  try {
    if (isAssigned(shift, userId)) {
      await api.scheduleItems.unassign(shift.id, userId);
    } else {
      await api.scheduleItems.assign(shift.id, userId);
    }
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

async function unassign(shift: ScheduleItem, userId: string) {
  try {
    await api.scheduleItems.unassign(shift.id, userId);
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

const assignShiftId = ref<string | null>(null);
const assignCandidates = computed(() => {
  const shift = shifts.value.find((s) => s.id === assignShiftId.value);
  const assigned = new Set(shift?.assignees.map((a) => a.id));
  return members.value.filter((m) => !assigned.has(m.id));
});

async function assignFromModal(userId: string) {
  const shiftId = assignShiftId.value;
  assignShiftId.value = null;
  if (!shiftId) return;
  try {
    await api.scheduleItems.assign(shiftId, userId);
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

// --- block edit / delete ----------------------------------------------------

async function saveBlock(payload: ScheduleItemCreate | ScheduleItemUpdate) {
  if (!item.value) return;
  try {
    await api.scheduleItems.update(item.value.id, payload as ScheduleItemUpdate);
    editOpen.value = false;
    showNotificationToast({ headline: "Schedule item saved", type: "success" });
    await load();
  } catch (error: any) {
    toastApiError(error);
  }
}

async function deleteBlock() {
  if (!item.value) return;
  try {
    await api.scheduleItems.remove(item.value.id);
    showNotificationToast({ headline: "Schedule item deleted", type: "success" });
    router.push({ name: "schedule" });
  } catch (error: any) {
    toastApiError(error);
  }
}

// --- live updates -------------------------------------------------------------

let wsListenerId = "";

onMounted(async () => {
  await load();
  await loadContext();
  websocket.connect();
  wsListenerId = websocket.on({ object: "scheduleItem", action: "changed" }, (message) => {
    if (message.data?.projectId === item.value?.project.id) {
      load();
    }
  });
});

onBeforeUnmount(() => {
  if (wsListenerId) websocket.off(wsListenerId);
});
</script>

<style scoped>
.input-field {
  @apply block rounded-md border border-primary-200 bg-surface px-2.5 py-1.5 text-sm text-primary-900 shadow-sm focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-white;
}
.btn-primary {
  @apply inline-flex cursor-pointer items-center gap-1.5 rounded-md bg-accent-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50;
}
.btn-secondary {
  @apply inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3 py-1.5 text-xs font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white;
}
</style>
