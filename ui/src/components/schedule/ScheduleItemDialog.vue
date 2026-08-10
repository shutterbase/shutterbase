<template>
  <!-- Admin-only create/edit dialog. Assignment (claim/drop, roster) happens in
       ScheduleItemPopover — this dialog is reached via the pen icon only. -->
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
                  <span
                    class="mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-500/10 text-accent-600 dark:bg-accent-500/15 dark:text-accent-400"
                  >
                    <CalendarDaysIcon class="h-5 w-5" aria-hidden="true" />
                  </span>
                  <div>
                    <p class="label-mono text-accent-600 dark:text-accent-400">Schedule item</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">
                      {{ create ? "Add schedule item" : "Edit schedule item" }}
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

                <!-- tag suggestions: searchable picker + removable chips —
                     projects end up with hundreds of tags, a chip cloud
                     doesn't scale. -->
                <div>
                  <p class="text-sm font-medium text-primary-700 dark:text-primary-200">Tag suggestions</p>
                  <p class="text-xs text-primary-500 dark:text-primary-400">Applied by the upload timeline to photos taken in this window.</p>
                  <div class="mt-2">
                    <SearchSelect
                      id="tag-suggestion-search"
                      v-model="tagPick"
                      aria-label="Search tags"
                      placeholder="Search tags…"
                      empty-text="No tag matches"
                      width-class="w-full"
                      :options="tagOptions"
                      :disabled="tagOptions.length === 0 && draft.tagIds.length === 0"
                    />
                  </div>
                  <div v-if="draft.tagIds.length" class="mt-2 flex flex-wrap gap-1.5">
                    <span
                      v-for="tagId in draft.tagIds"
                      :key="tagId"
                      class="inline-flex items-center gap-1 rounded-full border border-accent-500 bg-accent-500/15 py-0.5 pl-2.5 pr-1 text-xs font-medium text-accent-700 dark:text-accent-200"
                    >
                      {{ tagName(tagId) }}
                      <button
                        type="button"
                        class="flex h-4 w-4 cursor-pointer items-center justify-center rounded-full transition-colors hover:bg-accent-500/25"
                        :aria-label="`Remove tag ${tagName(tagId)}`"
                        @click="removeTag(tagId)"
                      >
                        <XMarkIcon class="h-3 w-3" />
                      </button>
                    </span>
                  </div>
                  <p v-else-if="selectableTags.length === 0" class="mt-2 text-xs text-primary-400">No tags in this project yet.</p>
                </div>
              </div>

              <!-- footer -->
              <div class="flex flex-row-reverse flex-wrap gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button type="button" class="btn-primary" :disabled="!valid" @click="save">
                  {{ create ? "Add item" : "Save item" }}
                </button>
                <button type="button" class="btn-secondary" @click="emit('closed')">Close</button>
                <button
                  v-if="!create"
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
import { CalendarDaysIcon, TrashIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { computed, ref, watch } from "vue";
import SearchSelect, { SearchSelectOption } from "src/components/SearchSelect.vue";
import { ImageTag, ScheduleItem } from "src/types/api";
import { tagLabel } from "src/util/tagOrder";
import { ScheduleItemCreate, ScheduleItemUpdate } from "src/api/scheduleItems";

interface Props {
  show: boolean;
  create: boolean;
  item?: ScheduleItem | null;
  projectTags: ImageTag[];
  // create mode: pre-filled window (ISO), e.g. from a drag on the calendar
  prefill?: { start: string; end: string } | null;
}

const props = withDefaults(defineProps<Props>(), { item: null, prefill: null });

const emit = defineEmits<{
  closed: [];
  save: [ScheduleItemCreate | ScheduleItemUpdate];
  deleted: [];
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
    } else if (props.prefill) {
      draft.value = { title: "", description: "", start: toLocal(props.prefill.start), end: toLocal(props.prefill.end), cardinality: 1, tagIds: [] };
    } else {
      const start = DateTime.now().startOf("hour").plus({ hours: 1 });
      draft.value = {
        title: "",
        description: "",
        start: start.toFormat("yyyy-MM-dd'T'HH:mm"),
        end: start.plus({ hours: 1 }).toFormat("yyyy-MM-dd'T'HH:mm"),
        cardinality: 1,
        tagIds: [],
      };
    }
  },
  { immediate: true },
);

// Suggestions may be any exported tag type; custom tags are excluded — they
// are never exported, so scheduling them makes no sense.
const selectableTags = computed(() => props.projectTags.filter((t) => t.type === "default" || t.type === "manual"));

const valid = computed(
  () => draft.value.title.trim() !== "" && draft.value.start !== "" && draft.value.end !== "" && toISO(draft.value.end) > toISO(draft.value.start) && draft.value.cardinality >= 1,
);

// Searchable tag picker: choosing an option adds a chip and resets the input.
const tagPick = ref("");
const tagOptions = computed<SearchSelectOption[]>(() =>
  selectableTags.value
    .filter((t) => !draft.value.tagIds.includes(t.id))
    .map((t) => ({ value: t.id, label: t.name, hint: t.type }))
    .sort((a, b) => a.label.localeCompare(b.label)),
);
watch(tagPick, (id) => {
  if (id && !draft.value.tagIds.includes(id)) draft.value.tagIds.push(id);
  if (id) tagPick.value = "";
});

// Chip labels: prefer the loaded project tags; an item may still reference a
// tag that fell out of the selectable set (e.g. type changed) — fall back to
// the item's own embedded tag names.
function tagName(id: string): string {
  const tag = props.projectTags.find((t) => t.id === id) ?? props.item?.tags.find((t) => t.id === id);
  return tag ? tagLabel(tag) : id;
}

function removeTag(id: string) {
  draft.value.tagIds = draft.value.tagIds.filter((t) => t !== id);
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
