<template>
  <TransitionRoot as="template" :show="show">
    <Dialog as="div" class="relative z-10" @close="emit('closed')">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity"></div>
      </TransitionChild>

      <div class="fixed inset-0 z-10 w-screen overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enter-to="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 translate-y-0 sm:scale-100"
            leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel
              class="relative w-full max-w-2xl transform overflow-hidden rounded-lg border border-primary-200 bg-surface text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark sm:my-8"
            >
              <!-- header -->
              <div class="flex items-start justify-between gap-4 border-b border-primary-100 px-6 py-5 dark:border-primary-800">
                <div class="flex items-start gap-3">
                  <span class="mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-500/10 text-accent-600 dark:bg-accent-500/15 dark:text-accent-400">
                    <CalendarDaysIcon class="h-5 w-5" aria-hidden="true" />
                  </span>
                  <div>
                    <p class="label-mono text-accent-600 dark:text-accent-400">Schedule item</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">
                      {{ create ? "Add schedule item" : canManage ? "Edit schedule item" : draft.title }}
                    </DialogTitle>
                  </div>
                </div>
                <button
                  type="button"
                  class="-mr-1 -mt-1 inline-flex h-8 w-8 flex-shrink-0 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:hover:bg-primary-800 dark:hover:text-primary-200"
                  @click="emit('closed')"
                >
                  <span class="sr-only">Close</span>
                  <XMarkIcon class="h-5 w-5" aria-hidden="true" />
                </button>
              </div>

              <!-- body -->
              <div class="space-y-5 px-6 py-5">
                <!-- occupancy banner (view + edit) -->
                <div v-if="!create && item" :class="['flex items-center gap-2 rounded-md border px-3 py-2 text-sm', OCCUPANCY_CLASSES[status]]">
                  <UsersIcon class="h-4 w-4 flex-shrink-0" />
                  <span class="font-medium">{{ OCCUPANCY_LABEL[status] }}</span>
                  <span class="opacity-75">{{ item.assignees.length }} / {{ item.cardinality }} covered</span>
                </div>

                <!-- fields: editable for managers, read-only otherwise -->
                <template v-if="canManage || create">
                  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <label class="block sm:col-span-2">
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Title</span>
                      <input v-model="draft.title" type="text" required class="input-field mt-1" placeholder="e.g. Endurance" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Start</span>
                      <input v-model="draft.start" type="datetime-local" required class="input-field mt-1" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-200">End</span>
                      <input v-model="draft.end" type="datetime-local" required class="input-field mt-1" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Cardinality</span>
                      <input v-model.number="draft.cardinality" type="number" min="1" class="input-field mt-1" />
                      <span class="mt-1 block text-xs text-primary-500 dark:text-primary-400">Target headcount — joining beyond it is allowed.</span>
                    </label>
                    <label class="block sm:col-span-2">
                      <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Description</span>
                      <textarea v-model="draft.description" rows="3" class="input-field mt-1" placeholder="Tasks, meeting point, gear…"></textarea>
                    </label>
                  </div>

                  <!-- tag suggestions -->
                  <div>
                    <p class="text-sm font-medium text-primary-700 dark:text-primary-200">Tag suggestions</p>
                    <p class="text-xs text-primary-500 dark:text-primary-400">Applied by the upload timeline to photos taken in this window.</p>
                    <div class="mt-2 flex max-h-40 flex-wrap gap-2 overflow-y-auto">
                      <button
                        v-for="tag in selectableTags"
                        :key="tag.id"
                        type="button"
                        @click="toggleTag(tag.id)"
                        :class="[
                          'cursor-pointer rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                          draft.tagIds.includes(tag.id)
                            ? 'border-accent-500 bg-accent-500/15 text-accent-700 dark:text-accent-200'
                            : 'border-primary-200 text-primary-500 hover:border-primary-300 hover:text-primary-700 dark:border-primary-700 dark:text-primary-400 dark:hover:text-primary-200',
                        ]"
                      >
                        {{ tag.name }}
                      </button>
                      <p v-if="selectableTags.length === 0" class="text-xs text-primary-400">No tags in this project yet.</p>
                    </div>
                  </div>
                </template>
                <template v-else-if="item">
                  <div class="space-y-2 text-sm text-primary-700 dark:text-primary-200">
                    <p class="flex items-center gap-2">
                      <ClockIcon class="h-4 w-4 text-primary-400" />
                      {{ formatWindow(item.start, item.end) }}
                    </p>
                    <p v-if="item.description" class="whitespace-pre-line text-primary-600 dark:text-primary-300">{{ item.description }}</p>
                    <div v-if="item.tags.length" class="flex flex-wrap items-center gap-1.5 pt-1">
                      <TagIcon class="h-4 w-4 text-primary-400" />
                      <span
                        v-for="tag in item.tags"
                        :key="tag.id"
                        class="rounded-full border border-primary-200 px-2 py-0.5 text-xs text-primary-600 dark:border-primary-700 dark:text-primary-300"
                      >
                        {{ tag.name }}
                      </span>
                    </div>
                  </div>
                </template>

                <!-- assignees -->
                <div v-if="!create && item">
                  <p class="text-sm font-medium text-primary-700 dark:text-primary-200">Covered by</p>
                  <ul class="mt-2 space-y-1.5">
                    <li v-for="assignee in item.assignees" :key="assignee.id" class="flex items-center justify-between gap-2">
                      <span class="flex items-center gap-2 text-sm text-primary-700 dark:text-primary-200">
                        <UserBubble :user="assignee" />
                        {{ assignee.firstName }} {{ assignee.lastName }}
                        <span v-if="assignee.id === currentUserId" class="text-xs text-primary-400">(you)</span>
                      </span>
                      <button
                        v-if="canManage && assignee.id !== currentUserId"
                        type="button"
                        class="cursor-pointer text-xs font-medium text-primary-400 transition-colors hover:text-red-600 dark:hover:text-red-400"
                        @click="emit('unassign', assignee.id)"
                      >
                        Remove
                      </button>
                    </li>
                    <li v-if="item.assignees.length === 0" class="text-sm text-primary-400">Nobody yet — be the first.</li>
                  </ul>

                  <!-- admin: pre-assign someone (transcript 05:58 "Teamfotos macht Axel") -->
                  <div v-if="canManage && assignableMembers.length" class="mt-3">
                    <SearchSelect
                      id="assign-member"
                      v-model="assignPick"
                      placeholder="Assign a member…"
                      empty-text="No member matches"
                      :options="assignableMembers"
                    />
                  </div>
                </div>
              </div>

              <!-- footer -->
              <div class="flex flex-row-reverse flex-wrap gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button v-if="canManage || create" type="button" class="btn-primary" :disabled="!valid" @click="save">
                  {{ create ? "Add item" : "Save item" }}
                </button>
                <button
                  v-if="!create && item && canJoin"
                  type="button"
                  class="btn-primary"
                  @click="emit(assigned ? 'leave' : 'join')"
                >
                  <UserPlusIcon v-if="!assigned" class="h-4 w-4" />
                  <UserMinusIcon v-else class="h-4 w-4" />
                  {{ assigned ? "Leave" : "Add to my schedule" }}
                </button>
                <button type="button" class="btn-secondary" @click="emit('closed')">Close</button>
                <button
                  v-if="canManage && !create"
                  type="button"
                  class="mr-auto inline-flex cursor-pointer items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-500/10 dark:text-red-400"
                  @click="emit('deleted')"
                >
                  <TrashIcon class="h-4 w-4" />
                  Delete
                </button>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { CalendarDaysIcon, ClockIcon, TagIcon, TrashIcon, UserMinusIcon, UserPlusIcon, UsersIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { computed, ref, watch } from "vue";
import SearchSelect, { SearchSelectOption } from "src/components/SearchSelect.vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import { EmbeddedUser, ImageTag, ScheduleItem } from "src/types/api";
import { OCCUPANCY_CLASSES, OCCUPANCY_LABEL, isAssigned, occupancyStatus } from "src/util/schedule";
import { ScheduleItemCreate, ScheduleItemUpdate } from "src/api/scheduleItems";

interface Props {
  show: boolean;
  create: boolean;
  item?: ScheduleItem | null;
  canManage: boolean;
  canJoin: boolean;
  currentUserId?: string;
  projectTags: ImageTag[];
  members: EmbeddedUser[];
  // Prefill for create mode (clicking an empty day slot).
  prefillStart?: string;
}

const props = withDefaults(defineProps<Props>(), { item: null, prefillStart: undefined });

const emit = defineEmits<{
  closed: [];
  save: [ScheduleItemCreate | ScheduleItemUpdate];
  deleted: [];
  join: [];
  leave: [];
  assign: [string];
  unassign: [string];
}>();

const draft = ref({ title: "", description: "", start: "", end: "", cardinality: 1, tagIds: [] as string[] });

// datetime-local speaks "yyyy-MM-dd'T'HH:mm" in LOCAL time; the API speaks ISO.
const toLocal = (iso: string) => DateTime.fromISO(iso).toFormat("yyyy-MM-dd'T'HH:mm");
const toISO = (local: string) => DateTime.fromISO(local).toUTC().toISO() ?? "";

watch(
  () => [props.show, props.item, props.create] as const,
  () => {
    if (!props.show) return;
    if (props.item && !props.create) {
      draft.value = {
        title: props.item.title,
        description: props.item.description ?? "",
        start: toLocal(props.item.start),
        end: toLocal(props.item.end),
        cardinality: props.item.cardinality,
        tagIds: props.item.tags.map((t) => t.id),
      };
    } else {
      const start = props.prefillStart ? DateTime.fromISO(props.prefillStart) : DateTime.now().startOf("hour").plus({ hours: 1 });
      draft.value = {
        title: "",
        description: "",
        start: start.toFormat("yyyy-MM-dd'T'HH:mm"),
        end: start.plus({ hours: 1 }).toFormat("yyyy-MM-dd'T'HH:mm"),
        cardinality: 1,
        tagIds: [],
      };
    }
    assignPick.value = "";
  },
  { immediate: true },
);

const status = computed(() => occupancyStatus(props.item?.assignees.length ?? 0, props.item?.cardinality ?? 1));
const assigned = computed(() => !!props.item && isAssigned(props.item, props.currentUserId));

// Suggestions may be any non-template tag; custom tags are excluded — they are
// never exported, so scheduling them makes no sense.
const selectableTags = computed(() => props.projectTags.filter((t) => t.type === "default" || t.type === "manual"));

const assignableMembers = computed<SearchSelectOption[]>(() => {
  const assignedIds = new Set(props.item?.assignees.map((a) => a.id));
  return props.members
    .filter((m) => !assignedIds.has(m.id))
    .map((m) => ({ value: m.id, label: `${m.firstName} ${m.lastName}` }))
    .sort((a, b) => a.label.localeCompare(b.label));
});

const assignPick = ref("");
watch(assignPick, (userId) => {
  if (userId) {
    emit("assign", userId);
    assignPick.value = "";
  }
});

const valid = computed(() => draft.value.title.trim() !== "" && draft.value.start !== "" && draft.value.end !== "" && toISO(draft.value.end) > toISO(draft.value.start) && draft.value.cardinality >= 1);

function toggleTag(id: string) {
  const idx = draft.value.tagIds.indexOf(id);
  if (idx >= 0) draft.value.tagIds.splice(idx, 1);
  else draft.value.tagIds.push(id);
}

function save() {
  emit("save", {
    title: draft.value.title.trim(),
    description: draft.value.description,
    start: toISO(draft.value.start),
    end: toISO(draft.value.end),
    cardinality: draft.value.cardinality,
    tagIds: draft.value.tagIds,
  });
}

function formatWindow(start: string, end: string): string {
  const s = DateTime.fromISO(start);
  const e = DateTime.fromISO(end);
  const sameDay = s.hasSame(e, "day");
  return sameDay ? `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("HH:mm")}` : `${s.toFormat("ccc dd.LL. HH:mm")} – ${e.toFormat("ccc dd.LL. HH:mm")}`;
}
</script>

<style scoped>
.input-field {
  @apply block w-full rounded-md border border-primary-200 bg-surface px-3 py-2 text-sm text-primary-900 shadow-sm focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-white;
}
.btn-primary {
  @apply inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950;
}
.btn-secondary {
  @apply inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white;
}
</style>
