<template>
  <div class="mx-auto w-full max-w-screen-2xl">
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="min-w-0">
          <h1 class="display text-3xl text-primary-900 dark:text-white">Schedule</h1>
          <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
            Pull items into your schedule — photos you upload from that window get the item's tags suggested.
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <!-- mine/all toggle (transcript 12:29) -->
          <div class="flex rounded-lg border border-primary-200 bg-surface p-0.5 dark:border-primary-700 dark:bg-surface-dark">
            <button
              v-for="option in scopeOptions"
              :key="option.value"
              type="button"
              @click="scope = option.value"
              :class="[
                'inline-flex h-8 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors cursor-pointer',
                scope === option.value
                  ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
                  : 'text-primary-500 hover:bg-primary-100 hover:text-primary-700 dark:text-primary-400 dark:hover:bg-primary-800 dark:hover:text-primary-200',
              ]"
            >
              <component :is="option.icon" class="h-4 w-4" />
              {{ option.label }}
            </button>
          </div>

          <button
            v-if="canManage"
            type="button"
            @click="openCreate()"
            class="inline-flex h-9 cursor-pointer items-center gap-1.5 rounded-md bg-accent-600 px-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950"
          >
            <PlusIcon class="h-4 w-4" />
            Add item
          </button>
        </div>
      </div>

      <!-- legend -->
      <div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-primary-500 dark:text-primary-400">
        <span v-for="s in legend" :key="s.status" class="inline-flex items-center gap-1.5">
          <span :class="['h-2.5 w-2.5 rounded-full border', legendDot[s.status]]"></span>
          {{ s.label }}
        </span>
      </div>
    </div>

    <!-- calendar -->
    <div class="mt-4 px-4 pb-10 sm:px-6 lg:px-8">
      <div class="overflow-x-auto rounded-lg border border-primary-200 bg-surface dark:border-primary-800 dark:bg-surface-dark">
        <div class="flex min-w-max">
          <!-- hour gutter -->
          <div class="sticky left-0 z-[5] w-14 flex-shrink-0 border-r border-primary-100 bg-surface dark:border-primary-800 dark:bg-surface-dark">
            <div class="h-10 border-b border-primary-100 dark:border-primary-800"></div>
            <div class="relative" :style="{ height: `${bodyHeight}px` }">
              <span
                v-for="hour in hourMarks"
                :key="hour"
                class="absolute right-2 -translate-y-1/2 text-[10px] tabular-nums text-primary-400"
                :style="{ top: `${((hour - hourRange.startHour) / (hourRange.endHour - hourRange.startHour)) * 100}%` }"
              >
                {{ String(hour).padStart(2, "0") }}:00
              </span>
            </div>
          </div>

          <!-- day columns -->
          <div v-for="day in days" :key="day.getTime()" class="w-48 flex-shrink-0 border-r border-primary-100 last:border-r-0 dark:border-primary-800">
            <div class="flex h-10 items-center justify-center border-b border-primary-100 text-sm font-medium text-primary-700 dark:border-primary-800 dark:text-primary-200">
              {{ formatDay(day) }}
            </div>
            <div class="relative" :style="{ height: `${bodyHeight}px` }" :data-testid="`day-${dayKey(day)}`">
              <!-- hour grid lines -->
              <div
                v-for="hour in hourMarks"
                :key="hour"
                class="absolute inset-x-0 border-t border-primary-100/70 dark:border-primary-800/70"
                :style="{ top: `${((hour - hourRange.startHour) / (hourRange.endHour - hourRange.startHour)) * 100}%` }"
              ></div>

              <!-- items -->
              <button
                v-for="entry in dayEntries(day)"
                :key="entry.item.id"
                type="button"
                :data-testid="`schedule-item-${entry.item.id}`"
                :ref="(el) => registerItemEl(entry.item.id, el as HTMLElement | null)"
                @click="openItem(entry.item)"
                :class="[
                  'absolute cursor-pointer overflow-hidden rounded-md border-l-4 px-1.5 py-1 text-left text-xs shadow-sm transition-shadow hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500',
                  OCCUPANCY_CLASSES[entry.status],
                  isAssigned(entry.item, userStore.user?.id) ? 'ring-1 ring-accent-500/60' : '',
                ]"
                :style="entry.style"
              >
                <p class="truncate font-semibold">{{ entry.item.title }}</p>
                <p class="truncate tabular-nums opacity-75">{{ formatTime(entry.item.start) }}–{{ formatTime(entry.item.end) }}</p>
                <div v-if="entry.item.assignees.length" class="mt-1 flex -space-x-1.5">
                  <UserBubble v-for="assignee in entry.item.assignees.slice(0, 4)" :key="assignee.id" :user="assignee" />
                  <span
                    v-if="entry.item.assignees.length > 4"
                    class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-200 text-[10px] font-medium text-primary-700 ring-2 ring-surface dark:bg-primary-700 dark:text-primary-200 dark:ring-surface-dark"
                  >
                    +{{ entry.item.assignees.length - 4 }}
                  </span>
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>
      <p v-if="visibleItems.length === 0" class="mt-6 text-center text-sm text-primary-400">
        {{ scope === "mine" ? "Nothing on your schedule yet — switch to “Everything” and grab an item." : "No schedule items yet." }}
      </p>
    </div>

    <ScheduleItemDialog
      :show="dialogOpen"
      :create="dialogCreate"
      :item="dialogItem"
      :can-manage="canManage"
      :can-join="canJoin"
      :current-user-id="userStore.user?.id"
      :project-tags="projectTags"
      :members="members"
      @closed="dialogOpen = false"
      @save="saveItem"
      @deleted="deleteItem"
      @join="assign(userStore.user!.id)"
      @leave="unassign(userStore.user!.id)"
      @assign="assign"
      @unassign="unassign"
    />
    <PurpleWave ref="wave" />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import confetti from "canvas-confetti";
import { CalendarDaysIcon, PlusIcon, UserIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { storeToRefs } from "pinia";
import { computed, onBeforeUnmount, onMounted, ref, Ref } from "vue";
import { useStorage } from "@vueuse/core";
import { api } from "src/api";
import { ScheduleItemCreate, ScheduleItemUpdate } from "src/api/scheduleItems";
import PurpleWave from "src/components/schedule/PurpleWave.vue";
import ScheduleItemDialog from "src/components/schedule/ScheduleItemDialog.vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { EmbeddedUser, ImageTag, Project, ScheduleItem } from "src/types/api";
import {
  OCCUPANCY_CLASSES,
  OCCUPANCY_LABEL,
  OccupancyStatus,
  assignLanes,
  calendarDays,
  dayHourRange,
  dayPosition,
  isAssigned,
  itemsOnDay,
  occupancyStatus,
} from "src/util/schedule";
import * as websocket from "src/util/websocket";

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const canManage = computed(() => userStore.isProjectAdminOrHigher());
const canJoin = computed(() => userStore.isProjectEditorOrHigher());

const scope = useStorage<"mine" | "all">("schedule-scope", "all");
const scopeOptions = [
  { value: "all" as const, label: "Everything", icon: CalendarDaysIcon },
  { value: "mine" as const, label: "My schedule", icon: UserIcon },
];

const legend = (["empty", "partial", "full", "over"] as OccupancyStatus[]).map((status) => ({ status, label: OCCUPANCY_LABEL[status] }));
const legendDot: Record<OccupancyStatus, string> = {
  empty: "border-blue-400 bg-blue-500/40",
  partial: "border-yellow-400 bg-yellow-400/50",
  full: "border-green-500 bg-green-500/40",
  over: "border-violet-500 bg-violet-500/50",
};

const items: Ref<ScheduleItem[]> = ref([]);
const project: Ref<Project | null> = ref(null);
const projectTags: Ref<ImageTag[]> = ref([]);
const members: Ref<EmbeddedUser[]> = ref([]);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

// --- celebrations ---------------------------------------------------------
// Transitions are detected on every (re)load, so they fire for own actions AND
// for teammates' moves arriving via websocket. full -> confetti at the item,
// over -> purple rupture wave across the screen (transcript 07:36 / 08:07).
const wave = ref<InstanceType<typeof PurpleWave>>();
const itemEls = new Map<string, HTMLElement>();
const statusById = new Map<string, OccupancyStatus>();
let statusesPrimed = false;

function registerItemEl(id: string, el: HTMLElement | null) {
  if (el) itemEls.set(id, el);
  else itemEls.delete(id);
}

function originOf(id: string): { x: number; y: number } {
  const el = itemEls.get(id);
  if (!el) return { x: 0.5, y: 0.5 };
  const rect = el.getBoundingClientRect();
  return {
    x: (rect.left + rect.width / 2) / window.innerWidth,
    y: (rect.top + rect.height / 2) / window.innerHeight,
  };
}

function celebrate(next: ScheduleItem[]) {
  if (statusesPrimed) {
    for (const item of next) {
      const before = statusById.get(item.id);
      const after = occupancyStatus(item.assignees.length, item.cardinality);
      if (before === after || before === undefined) continue;
      if (after === "full") {
        const origin = originOf(item.id);
        confetti({ particleCount: 90, spread: 70, startVelocity: 35, origin, disableForReducedMotion: true });
      } else if (after === "over") {
        const origin = originOf(item.id);
        wave.value?.trigger(origin.x, origin.y);
      }
    }
  }
  statusById.clear();
  next.forEach((item) => statusById.set(item.id, occupancyStatus(item.assignees.length, item.cardinality)));
  statusesPrimed = true;
}

// --- data ------------------------------------------------------------------

async function requestItems() {
  try {
    const result = await api.scheduleItems.list({ projectId: activeProjectId.value, limit: 500, sort: "start", order: "asc" });
    celebrate(result.items);
    items.value = result.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function loadContext() {
  try {
    const [proj, tags, assignments] = await Promise.all([
      api.projects.get(activeProjectId.value),
      api.imageTags.list({ projectId: activeProjectId.value, limit: 500, sort: "name", order: "asc" }),
      api.projectAssignments.list({ projectId: activeProjectId.value, limit: 500 }),
    ]);
    project.value = proj;
    projectTags.value = tags.items;
    const seen = new Map<string, EmbeddedUser>();
    assignments.items.forEach((a) => seen.set(a.user.id, a.user));
    members.value = [...seen.values()];
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- calendar geometry -------------------------------------------------------

const PX_PER_HOUR = 56;

const visibleItems = computed(() => (scope.value === "mine" ? items.value.filter((i) => isAssigned(i, userStore.user?.id)) : items.value));
const days = computed(() => calendarDays(project.value ?? {}, visibleItems.value.length ? visibleItems.value : items.value));
const hourRange = computed(() => dayHourRange(items.value));
const bodyHeight = computed(() => (hourRange.value.endHour - hourRange.value.startHour) * PX_PER_HOUR);
const hourMarks = computed(() => {
  const marks: number[] = [];
  for (let h = hourRange.value.startHour + 1; h < hourRange.value.endHour; h++) marks.push(h);
  return marks;
});

interface DayEntry {
  item: ScheduleItem;
  status: OccupancyStatus;
  style: Record<string, string>;
}

function dayEntries(day: Date): DayEntry[] {
  const dayItems = itemsOnDay(visibleItems.value, day);
  const lanes = assignLanes(dayItems);
  return dayItems.map((item) => {
    const pos = dayPosition(item, day, hourRange.value.startHour, hourRange.value.endHour);
    const lane = lanes.get(item.id) ?? { lane: 0, lanes: 1 };
    const widthPct = 100 / lane.lanes;
    return {
      item,
      status: occupancyStatus(item.assignees.length, item.cardinality),
      style: {
        top: `${pos.topPct}%`,
        height: `${pos.heightPct}%`,
        left: `calc(${lane.lane * widthPct}% + 2px)`,
        width: `calc(${widthPct}% - 4px)`,
      },
    };
  });
}

const formatDay = (day: Date) => DateTime.fromJSDate(day).toFormat("ccc dd.LL.");
const formatTime = (iso: string) => DateTime.fromISO(iso).toFormat("HH:mm");
const dayKey = (day: Date) => DateTime.fromJSDate(day).toFormat("yyyy-LL-dd");

// --- dialog + actions --------------------------------------------------------

const dialogOpen = ref(false);
const dialogCreate = ref(false);
const dialogItem: Ref<ScheduleItem | null> = ref(null);

function openCreate() {
  dialogCreate.value = true;
  dialogItem.value = null;
  dialogOpen.value = true;
}

function openItem(item: ScheduleItem) {
  dialogCreate.value = false;
  dialogItem.value = item;
  dialogOpen.value = true;
}

// Keep the dialog's item reference fresh after any mutation/refetch.
function syncDialogItem() {
  if (dialogItem.value) dialogItem.value = items.value.find((i) => i.id === dialogItem.value?.id) ?? null;
}

async function saveItem(payload: ScheduleItemCreate | ScheduleItemUpdate) {
  try {
    if (dialogCreate.value) {
      await api.scheduleItems.create({ ...(payload as ScheduleItemCreate), projectId: activeProjectId.value });
      showNotificationToast({ headline: "Schedule item added", type: "success" });
    } else if (dialogItem.value) {
      await api.scheduleItems.update(dialogItem.value.id, payload);
      showNotificationToast({ headline: "Schedule item saved", type: "success" });
    }
    dialogOpen.value = false;
    await requestItems();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function deleteItem() {
  if (!dialogItem.value) return;
  try {
    await api.scheduleItems.remove(dialogItem.value.id);
    dialogOpen.value = false;
    showNotificationToast({ headline: "Schedule item deleted", type: "success" });
    await requestItems();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function assign(userId: string) {
  if (!dialogItem.value) return;
  try {
    await api.scheduleItems.assign(dialogItem.value.id, userId);
    await requestItems();
    syncDialogItem();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function unassign(userId: string) {
  if (!dialogItem.value) return;
  try {
    await api.scheduleItems.unassign(dialogItem.value.id, userId);
    await requestItems();
    syncDialogItem();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- live updates ------------------------------------------------------------

let wsListenerId = "";

onMounted(async () => {
  await Promise.all([loadContext(), requestItems()]);
  websocket.connect();
  wsListenerId = websocket.on({ object: "scheduleItem", action: "changed" }, (message) => {
    if (message.data?.projectId === activeProjectId.value) {
      requestItems().then(syncDialogItem);
    }
  });
});

onBeforeUnmount(() => {
  if (wsListenerId) websocket.off(wsListenerId);
});
</script>
