<template>
  <div class="mx-auto w-full max-w-screen-2xl">
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="min-w-0">
          <h1 class="display text-3xl text-primary-900 dark:text-white">Schedule</h1>
          <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Pull items into your schedule — photos you upload from that window get the item's tags suggested.</p>
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

          <!-- view range: day / 3 days / week / whole event -->
          <div class="flex rounded-lg border border-primary-200 bg-surface p-0.5 dark:border-primary-700 dark:bg-surface-dark">
            <button
              v-for="option in viewOptions"
              :key="option.value"
              type="button"
              @click="viewMode = option.value"
              :class="[
                'inline-flex h-8 cursor-pointer items-center rounded-md px-2.5 text-xs font-medium transition-colors',
                viewMode === option.value
                  ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
                  : 'text-primary-500 hover:bg-primary-100 hover:text-primary-700 dark:text-primary-400 dark:hover:bg-primary-800 dark:hover:text-primary-200',
              ]"
            >
              {{ option.label }}
            </button>
          </div>
          <div v-if="viewMode !== 'all'" class="flex items-center gap-1">
            <button
              type="button"
              class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md text-primary-500 transition-colors hover:bg-primary-100 hover:text-primary-700 disabled:cursor-default disabled:opacity-40 dark:hover:bg-primary-800 dark:hover:text-primary-200"
              :disabled="!canPrev"
              aria-label="Previous"
              @click="shiftAnchor(-1)"
            >
              <ChevronLeftIcon class="h-4 w-4" />
            </button>
            <span class="min-w-28 text-center text-xs tabular-nums text-primary-600 dark:text-primary-300">{{ rangeLabel }}</span>
            <button
              type="button"
              class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md text-primary-500 transition-colors hover:bg-primary-100 hover:text-primary-700 disabled:cursor-default disabled:opacity-40 dark:hover:bg-primary-800 dark:hover:text-primary-200"
              :disabled="!canNext"
              aria-label="Next"
              @click="shiftAnchor(1)"
            >
              <ChevronRightIcon class="h-4 w-4" />
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

          <!-- legend -->
          <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-primary-500 dark:text-primary-400">
            <span v-for="s in legend" :key="s.status" class="inline-flex items-center gap-1.5">
              <span :class="['h-2.5 w-2.5 rounded-full border', legendDot[s.status]]"></span>
              {{ s.label }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- calendar: fixed 24h axis, sized to the viewport — the page never
         scrolls vertically, only the day columns scroll horizontally. -->
    <div class="mt-4 px-4 pb-6 sm:px-6 lg:px-8">
      <div class="overflow-x-auto rounded-lg border border-primary-200 bg-surface dark:border-primary-800 dark:bg-surface-dark">
        <div :class="['flex', viewMode === 'all' ? 'min-w-max' : 'w-full']">
          <!-- hour gutter -->
          <div class="sticky left-0 z-[5] w-14 flex-shrink-0 border-r border-primary-100 bg-surface dark:border-primary-800 dark:bg-surface-dark">
            <div class="h-9 border-b border-primary-100 dark:border-primary-800"></div>
            <div class="relative" :class="bodyHeightClass">
              <span
                v-for="hour in hourMarks"
                :key="hour"
                class="absolute right-2 -translate-y-1/2 text-[10px] tabular-nums text-primary-400"
                :style="{ top: `${(hour / 24) * 100}%` }"
              >
                {{ String(hour).padStart(2, "0") }}:00
              </span>
            </div>
          </div>

          <!-- day columns -->
          <div
            v-for="day in days"
            :key="day.getTime()"
            :class="[viewMode === 'all' ? 'w-48 flex-shrink-0' : 'min-w-40 flex-1', 'border-r border-primary-100 last:border-r-0 dark:border-primary-800']"
          >
            <div class="flex h-9 items-center justify-center border-b border-primary-100 text-sm font-medium text-primary-700 dark:border-primary-800 dark:text-primary-200">
              {{ formatDay(day) }}
            </div>
            <div
              class="relative"
              :class="bodyHeightClass"
              :data-testid="`day-${dayKey(day)}`"
              @pointerdown="onColDown(day, $event)"
              @pointermove="onColMove"
              @pointerup="onColUp"
              @pointercancel="dragState = null"
            >
              <!-- drag-create ghost -->
              <div
                v-if="dragState && dragState.dayKey === dayKey(day)"
                class="pointer-events-none absolute inset-x-1 z-10 rounded-md border-2 border-dashed border-accent-500 bg-accent-500/10"
                :style="ghostStyle"
              ></div>
              <!-- hour grid lines -->
              <div
                v-for="hour in hourMarks"
                :key="hour"
                class="absolute inset-x-0 border-t border-primary-100/70 dark:border-primary-800/70"
                :style="{ top: `${(hour / 24) * 100}%` }"
              ></div>

              <!-- items: click opens the assignment popover; the pen (admins
                   only) opens the edit dialog -->
              <div
                v-for="entry in dayEntries(day)"
                :key="entry.item.id"
                :data-testid="`schedule-item-${entry.item.id}`"
                :ref="(el) => registerItemEl(entry.item.id, el as HTMLElement | null)"
                role="button"
                tabindex="0"
                @click="openItem(entry.item)"
                @keydown.enter="openItem(entry.item)"
                :class="[
                  'group absolute cursor-pointer overflow-hidden rounded-md border-l-4 px-1.5 py-0.5 text-left text-xs shadow-sm transition-shadow hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500',
                  OCCUPANCY_CLASSES[entry.status],
                  isAssignedInBlock(entry.item, userStore.user?.id) ? 'ring-1 ring-accent-500/60' : '',
                ]"
                :style="entry.style"
              >
                <p class="truncate pr-5 font-semibold">{{ entry.item.title }}</p>
                <p class="truncate tabular-nums opacity-75">
                  {{ formatTime(entry.item.start) }}–{{ formatTime(entry.item.end)
                  }}<span v-if="entry.item.shifts?.length"> · {{ entry.item.shifts.length }} shift{{ entry.item.shifts.length === 1 ? "" : "s" }}</span>
                </p>
                <div v-if="tileAssignees(entry.item).length" class="mt-0.5 flex -space-x-1.5">
                  <UserBubble v-for="assignee in tileAssignees(entry.item).slice(0, 3)" :key="assignee.id" :user="assignee" />
                  <span
                    v-if="tileAssignees(entry.item).length > 3"
                    class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-200 text-[10px] font-medium text-primary-700 ring-2 ring-surface dark:bg-primary-700 dark:text-primary-200 dark:ring-surface-dark"
                  >
                    +{{ tileAssignees(entry.item).length - 3 }}
                  </span>
                </div>

                <button
                  v-if="canManage"
                  type="button"
                  :aria-label="`Edit ${entry.item.title}`"
                  class="absolute right-0.5 top-0.5 flex h-5 w-5 cursor-pointer items-center justify-center rounded bg-surface/70 text-primary-500 opacity-70 transition-opacity hover:opacity-100 dark:bg-surface-dark/70 dark:text-primary-300"
                  @click.stop="openEdit(entry.item)"
                >
                  <PencilIcon class="h-3 w-3" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      <p v-if="visibleItems.length === 0" class="mt-3 text-center text-sm text-primary-400">
        {{ scope === "mine" ? "Nothing on your schedule yet — switch to “Everything” and grab an item." : "No schedule items yet." }}
      </p>
    </div>

    <ScheduleItemPopover
      :item="popoverItem"
      :position="popoverPosition"
      :can-manage="canManage"
      :can-join="canJoin"
      :current-user-id="userStore.user?.id"
      @closed="popoverItemId = null"
      @claim="assign(popoverItemId!, userStore.user!.id)"
      @drop="unassign(popoverItemId!, userStore.user!.id)"
      @claim-shift="(shiftId) => assign(shiftId, userStore.user!.id)"
      @drop-shift="(shiftId) => unassign(shiftId, userStore.user!.id)"
      @unassign="(userId) => unassign(popoverItemId!, userId)"
      @open-assign="assignModalOpen = true"
      @open-shifts="router.push({ name: 'schedule-item', params: { id: popoverItemId! } })"
    />

    <AssignPhotographerModal :show="assignModalOpen" :candidates="assignCandidates" @closed="assignModalOpen = false" @assign="assignFromModal" />

    <ScheduleItemDialog
      :show="dialogOpen"
      :create="dialogCreate"
      :item="dialogItem"
      :project-tags="projectTags"
      :prefill="dialogPrefill"
      @closed="((dialogOpen = false), (dialogPrefill = null))"
      @save="saveItem"
      @deleted="deleteItem"
    />
    <PurpleWave ref="wave" />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import confetti from "canvas-confetti";
import { CalendarDaysIcon, ChevronLeftIcon, ChevronRightIcon, PencilIcon, PlusIcon, UserIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { storeToRefs } from "pinia";
import { computed, onBeforeUnmount, onMounted, ref, Ref } from "vue";
import { useStorage } from "@vueuse/core";
import { api } from "src/api";
import { ScheduleItemCreate, ScheduleItemUpdate } from "src/api/scheduleItems";
import AssignPhotographerModal from "src/components/schedule/AssignPhotographerModal.vue";
import PurpleWave from "src/components/schedule/PurpleWave.vue";
import ScheduleItemDialog from "src/components/schedule/ScheduleItemDialog.vue";
import ScheduleItemPopover from "src/components/schedule/ScheduleItemPopover.vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { EmbeddedUser, ImageTag, Project, ScheduleItem } from "src/types/api";
import {
  CalendarView,
  OCCUPANCY_CLASSES,
  OCCUPANCY_LABEL,
  OccupancyStatus,
  VIEW_STEP_DAYS,
  addDays,
  assignLanes,
  blockStatus,
  calendarDays,
  dayPosition,
  isAssignedInBlock,
  itemsOnDay,
  pctToTime,
  startOfDay,
  viewDays,
} from "src/util/schedule";
import * as websocket from "src/util/websocket";
import { useRouter } from "vue-router";

const router = useRouter();
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
// for teammates' moves arriving via websocket. Confetti only when full is
// reached by JOINING (empty/partial -> full) — dropping from overbooked back
// to full is nothing to celebrate. Purple rupture wave on newly overbooked.
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
      const after = blockStatus(item);
      if (before === after || before === undefined) continue;
      if (after === "full" && (before === "empty" || before === "partial")) {
        const origin = originOf(item.id);
        confetti({ particleCount: 90, spread: 70, startVelocity: 35, origin, disableForReducedMotion: true });
      } else if (after === "over") {
        const origin = originOf(item.id);
        wave.value?.trigger(origin.x, origin.y);
      }
    }
  }
  statusById.clear();
  next.forEach((item) => statusById.set(item.id, blockStatus(item)));
  statusesPrimed = true;
}

// --- data ------------------------------------------------------------------

async function requestItems() {
  try {
    const result = await api.scheduleItems.list({ projectId: activeProjectId.value, limit: 500, sort: "start", order: "asc" });
    celebrate(result.items);
    items.value = result.items;
    syncOpenItem();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function loadContext() {
  try {
    project.value = await api.projects.get(activeProjectId.value);
    // Tags + member roster only exist for managers (edit dialog / assign
    // modal), and the project-assignments list is projectAdmin-gated anyway —
    // a photographer requesting it would 403 straight into the error modal.
    if (canManage.value) {
      const [tags, assignments] = await Promise.all([
        api.imageTags.list({ projectId: activeProjectId.value, limit: 500, sort: "name", order: "asc" }),
        api.projectAssignments.list({ projectId: activeProjectId.value, limit: 500 }),
      ]);
      projectTags.value = tags.items;
      const seen = new Map<string, EmbeddedUser>();
      assignments.items.forEach((a) => seen.set(a.user.id, a.user));
      members.value = [...seen.values()];
    }
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

// --- calendar geometry -------------------------------------------------------

// Fixed 24h axis that fits the viewport: header (~64px) + toolbar/legend +
// day-header row leave roughly 21rem of chrome above the fold.
const bodyHeightClass = "h-[calc(100dvh-21rem)] min-h-[420px]";
const hourMarks = [2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22];

const visibleItems = computed(() => (scope.value === "mine" ? items.value.filter((i) => isAssignedInBlock(i, userStore.user?.id)) : items.value));

// --- view range (day / 3day / week / all) ------------------------------------

const viewMode = useStorage<CalendarView>("schedule-view", "all");
const viewOptions: { value: CalendarView; label: string }[] = [
  { value: "day", label: "Day" },
  { value: "3day", label: "3 days" },
  { value: "week", label: "Week" },
  { value: "all", label: "Event" },
];

// the project-wide span bounds the anchor so prev/next cannot wander into
// empty months; anchor defaults to today (clamped into the span)
const allDays = computed(() => calendarDays(project.value ?? {}, visibleItems.value.length ? visibleItems.value : items.value));
const anchorOverride = ref<Date | null>(null);
const anchor = computed(() => {
  const bounds = allDays.value;
  const t = startOfDay(anchorOverride.value ?? new Date()).getTime();
  const first = bounds[0].getTime();
  const last = bounds[bounds.length - 1].getTime();
  return new Date(Math.min(Math.max(t, first), last));
});

const days = computed(() => (viewMode.value === "all" ? allDays.value : viewDays(viewMode.value, anchor.value, project.value ?? {}, visibleItems.value)));

const canPrev = computed(() => days.value[0].getTime() > allDays.value[0].getTime());
const canNext = computed(() => days.value[days.value.length - 1].getTime() < allDays.value[allDays.value.length - 1].getTime());

function shiftAnchor(direction: 1 | -1) {
  anchorOverride.value = addDays(anchor.value, direction * VIEW_STEP_DAYS[viewMode.value]);
}

const rangeLabel = computed(() => {
  const first = days.value[0];
  const last = days.value[days.value.length - 1];
  return days.value.length === 1 ? formatDay(first) : `${formatDay(first)} – ${formatDay(last)}`;
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
    const pos = dayPosition(item, day);
    const lane = lanes.get(item.id) ?? { lane: 0, lanes: 1 };
    const widthPct = 100 / lane.lanes;
    return {
      item,
      status: blockStatus(item),
      style: {
        top: `${pos.topPct}%`,
        height: `${pos.heightPct}%`,
        left: `calc(${lane.lane * widthPct}% + 2px)`,
        width: `calc(${widthPct}% - 4px)`,
      },
    };
  });
}

// tile bubbles: a subdivided block's people live on its shifts — show the
// distinct union of both claim surfaces.
function tileAssignees(item: ScheduleItem): EmbeddedUser[] {
  if (!item.shifts?.length) return item.assignees;
  const seen = new Map<string, EmbeddedUser>();
  item.assignees.forEach((a) => seen.set(a.id, a));
  item.shifts.forEach((s) => s.assignees.forEach((a) => seen.set(a.id, a)));
  return [...seen.values()];
}

const formatDay = (day: Date) => DateTime.fromJSDate(day).toFormat("ccc dd.LL.");
const formatTime = (iso: string) => DateTime.fromISO(iso).toFormat("HH:mm");
const dayKey = (day: Date) => DateTime.fromJSDate(day).toFormat("yyyy-LL-dd");

// --- popover (claim tooltip) --------------------------------------------------

const popoverItemId = ref<string | null>(null);
const popoverPosition = ref({ x: 0, y: 0 });
const popoverItem = computed(() => items.value.find((i) => i.id === popoverItemId.value) ?? null);

// Every item opens the claim popover — subdivided blocks list their shifts
// there (claim per shift, same pull principle); the detail page stays the
// admin surface for editing shifts.
function openItem(item: ScheduleItem) {
  openPopover(item);
}

function openPopover(item: ScheduleItem) {
  const el = itemEls.get(item.id);
  if (el) {
    const rect = el.getBoundingClientRect();
    popoverPosition.value = {
      // 344 = widest popover (w-80 for blocks) + margin
      x: Math.max(8, Math.min(rect.right + 8, window.innerWidth - 344)),
      y: Math.max(8, Math.min(rect.top, window.innerHeight - 360)),
    };
  }
  popoverItemId.value = item.id;
}

// --- dialog (admin create/edit) ------------------------------------------------

const dialogOpen = ref(false);
const dialogCreate = ref(false);
const dialogItem: Ref<ScheduleItem | null> = ref(null);
const dialogPrefill: Ref<{ start: string; end: string } | null> = ref(null);

function openCreate(prefill: { start: string; end: string } | null = null) {
  dialogPrefill.value = prefill;
  dialogCreate.value = true;
  dialogItem.value = null;
  dialogOpen.value = true;
}

// --- drag-create: pull a time span open on a day column (admins) -------------

const dragState = ref<{ dayKey: string; day: Date; fromPct: number; toPct: number } | null>(null);

function colPct(e: PointerEvent): number {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
  return Math.min(100, Math.max(0, ((e.clientY - rect.top) / rect.height) * 100));
}

function onColDown(day: Date, e: PointerEvent) {
  if (!canManage.value) return;
  if ((e.target as HTMLElement).closest("[data-testid^='schedule-item-']")) return; // item clicks stay item clicks
  (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  const pct = colPct(e);
  dragState.value = { dayKey: dayKey(day), day, fromPct: pct, toPct: pct };
}

function onColMove(e: PointerEvent) {
  if (!dragState.value) return;
  dragState.value.toPct = colPct(e);
}

function onColUp() {
  const d = dragState.value;
  dragState.value = null;
  if (!d) return;
  const [a, b] = [Math.min(d.fromPct, d.toPct), Math.max(d.fromPct, d.toPct)];
  if (b - a < 2) return; // a click, not a drag
  const start = pctToTime(d.day, a);
  const end = pctToTime(d.day, b);
  if (end.getTime() <= start.getTime()) return;
  openCreate({ start: start.toISOString(), end: end.toISOString() });
}

const ghostStyle = computed(() => {
  const d = dragState.value;
  if (!d) return {};
  const top = Math.min(d.fromPct, d.toPct);
  return { top: `${top}%`, height: `${Math.abs(d.toPct - d.fromPct)}%` };
});

function openEdit(item: ScheduleItem) {
  popoverItemId.value = null;
  dialogCreate.value = false;
  dialogItem.value = item;
  dialogOpen.value = true;
}

// Keep the dialog's item reference fresh after any mutation/refetch; the
// popover follows items.value by id automatically.
function syncOpenItem() {
  if (dialogItem.value) dialogItem.value = items.value.find((i) => i.id === dialogItem.value?.id) ?? null;
}

async function saveItem(payload: ScheduleItemCreate | ScheduleItemUpdate) {
  try {
    if (dialogCreate.value) {
      const created = await api.scheduleItems.create({ ...(payload as ScheduleItemCreate), projectId: activeProjectId.value });
      showNotificationToast({ headline: "Schedule item added", type: "success" });
      if (dialogPrefill.value) {
        // drag-created blocks flow straight into the detail page to add shifts
        dialogOpen.value = false;
        dialogPrefill.value = null;
        router.push({ name: "schedule-item", params: { id: created.id } });
        return;
      }
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

// --- assignment actions ---------------------------------------------------------

const assignModalOpen = ref(false);
const assignCandidates = computed(() => {
  const assigned = new Set(popoverItem.value?.assignees.map((a) => a.id));
  return members.value.filter((m) => !assigned.has(m.id));
});

async function assign(itemId: string, userId: string) {
  try {
    await api.scheduleItems.assign(itemId, userId);
    await requestItems();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function unassign(itemId: string, userId: string) {
  try {
    await api.scheduleItems.unassign(itemId, userId);
    await requestItems();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function assignFromModal(userId: string) {
  assignModalOpen.value = false;
  if (popoverItemId.value) await assign(popoverItemId.value, userId);
}

// --- live updates ------------------------------------------------------------

let wsListenerId = "";

onMounted(async () => {
  await Promise.all([loadContext(), requestItems()]);
  websocket.connect();
  wsListenerId = websocket.on({ object: "scheduleItem", action: "changed" }, (message) => {
    if (message.data?.projectId === activeProjectId.value) {
      requestItems();
    }
  });
});

onBeforeUnmount(() => {
  if (wsListenerId) websocket.off(wsListenerId);
});
</script>
