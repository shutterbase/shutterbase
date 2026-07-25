<template>
  <!-- The assignment popover (claim tooltip): the NORMAL click target of a
       calendar item. Claim/drop for photographers; the admin additionally
       removes people per row ("x") and opens the assign modal. Editing fields
       lives behind the pen icon on the item, not here. -->
  <Teleport to="body">
    <div v-if="item" class="fixed inset-0 z-30" @click="emit('closed')" @keydown.esc="emit('closed')">
      <div
        data-testid="schedule-popover"
        class="fixed z-40 w-72 rounded-lg border border-primary-200 bg-surface p-4 shadow-panel dark:border-primary-700 dark:bg-surface-dark dark:shadow-panel-dark"
        :style="{ left: `${position.x}px`, top: `${position.y}px` }"
        @click.stop
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-primary-900 dark:text-white">{{ item.title }}</p>
            <p class="text-xs tabular-nums text-primary-500 dark:text-primary-400">{{ window }}</p>
          </div>
          <span :class="['flex-shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium', OCCUPANCY_CLASSES[status]]">
            {{ item.assignees.length }}/{{ item.cardinality }}
          </span>
        </div>

        <p :class="['mt-2 text-xs font-medium', statusTextClasses[status]]">{{ OCCUPANCY_LABEL[status] }}</p>

        <ul class="mt-3 max-h-44 space-y-1.5 overflow-y-auto">
          <li v-for="assignee in item.assignees" :key="assignee.id" class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-2 text-sm text-primary-700 dark:text-primary-200">
              <UserBubble :user="assignee" />
              <span class="truncate">{{ assignee.firstName }} {{ assignee.lastName }}</span>
              <span v-if="assignee.id === currentUserId" class="flex-shrink-0 text-xs text-primary-400">(you)</span>
            </span>
            <button
              v-if="canManage"
              type="button"
              class="flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-red-500/10 hover:text-red-600 dark:hover:text-red-400"
              :aria-label="`Remove ${assignee.firstName} ${assignee.lastName}`"
              @click="emit('unassign', assignee.id)"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </li>
          <li v-if="item.assignees.length === 0" class="text-sm text-primary-400">Nobody yet — be the first.</li>
        </ul>

        <div class="mt-4 flex flex-wrap gap-2">
          <button v-if="canJoin" type="button" :class="assigned ? 'pop-btn-secondary' : 'pop-btn-primary'" @click="emit(assigned ? 'drop' : 'claim')">
            <UserMinusIcon v-if="assigned" class="h-4 w-4" />
            <UserPlusIcon v-else class="h-4 w-4" />
            {{ assigned ? "Leave" : "Add to my schedule" }}
          </button>
          <button v-if="canManage" type="button" class="pop-btn-secondary" @click="emit('openAssign')">
            <UserGroupIcon class="h-4 w-4" />
            Assign
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { UserGroupIcon, UserMinusIcon, UserPlusIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { computed, onBeforeUnmount, onMounted } from "vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import { ScheduleItem } from "src/types/api";
import { OCCUPANCY_CLASSES, OCCUPANCY_LABEL, OccupancyStatus, isAssigned, occupancyStatus } from "src/util/schedule";

const props = defineProps<{
  item: ScheduleItem | null;
  position: { x: number; y: number };
  canManage: boolean;
  canJoin: boolean;
  currentUserId?: string;
}>();

const emit = defineEmits<{
  closed: [];
  claim: [];
  drop: [];
  unassign: [string];
  openAssign: [];
}>();

const status = computed(() => occupancyStatus(props.item?.assignees.length ?? 0, props.item?.cardinality ?? 1));
const assigned = computed(() => !!props.item && isAssigned(props.item, props.currentUserId));

const statusTextClasses: Record<OccupancyStatus, string> = {
  empty: "text-blue-600 dark:text-blue-300",
  partial: "text-yellow-700 dark:text-yellow-200",
  full: "text-green-700 dark:text-green-300",
  over: "text-violet-700 dark:text-violet-300",
};

const window = computed(() => {
  if (!props.item) return "";
  const s = DateTime.fromISO(props.item.start);
  const e = DateTime.fromISO(props.item.end);
  return s.hasSame(e, "day") ? `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("HH:mm")}` : `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("ccc dd.LL. HH:mm")}`;
});

function onKey(e: KeyboardEvent) {
  if (e.key === "Escape") emit("closed");
}
onMounted(() => document.addEventListener("keydown", onKey));
onBeforeUnmount(() => document.removeEventListener("keydown", onKey));
</script>

<style scoped>
.pop-btn-primary {
  @apply inline-flex cursor-pointer items-center gap-1.5 rounded-md bg-accent-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500;
}
.pop-btn-secondary {
  @apply inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3 py-1.5 text-xs font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white;
}
</style>
