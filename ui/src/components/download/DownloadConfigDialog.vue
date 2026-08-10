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
                  <span
                    class="mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-500/10 text-accent-600 dark:bg-accent-500/15 dark:text-accent-400"
                  >
                    <ArrowDownTrayIcon class="h-5 w-5" aria-hidden="true" />
                  </span>
                  <div>
                    <p class="label-mono text-accent-600 dark:text-accent-400">Download config</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">
                      {{ create ? "Add download config" : "Edit download config" }}
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
                <label class="block">
                  <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Name</span>
                  <input v-model="draft.name" type="text" required maxlength="100" class="input-field mt-1" placeholder="e.g. Full delivery, Social media picks" />
                </label>

                <div>
                  <p class="text-sm font-medium text-primary-700 dark:text-primary-200">Include tags (AND)</p>
                  <p class="text-xs text-primary-500 dark:text-primary-400">Only photos carrying every one of these tags are downloaded. Empty = all photos.</p>
                  <div class="mt-2">
                    <SearchSelect
                      id="whitelist-tag-search"
                      v-model="whitelistPick"
                      aria-label="Search include tags"
                      placeholder="Search tags…"
                      empty-text="No tag matches"
                      width-class="w-full"
                      :options="tagOptions(draft.whitelistTagIds)"
                    />
                  </div>
                  <div v-if="draft.whitelistTagIds.length" class="mt-2 flex flex-wrap gap-1.5">
                    <span
                      v-for="tagId in draft.whitelistTagIds"
                      :key="tagId"
                      class="inline-flex items-center gap-1 rounded-full border border-accent-500 bg-accent-500/15 py-0.5 pl-2.5 pr-1 text-xs font-medium text-accent-700 dark:text-accent-200"
                    >
                      {{ tagName(tagId) }}
                      <button
                        type="button"
                        class="flex h-4 w-4 cursor-pointer items-center justify-center rounded-full transition-colors hover:bg-accent-500/25"
                        :aria-label="`Remove tag ${tagName(tagId)}`"
                        @click="draft.whitelistTagIds = draft.whitelistTagIds.filter((t) => t !== tagId)"
                      >
                        <XMarkIcon class="h-3 w-3" />
                      </button>
                    </span>
                  </div>
                </div>

                <div>
                  <p class="text-sm font-medium text-primary-700 dark:text-primary-200">Exclude tags (OR)</p>
                  <p class="text-xs text-primary-500 dark:text-primary-400">Photos carrying any of these tags are skipped — e.g. internal or QA-rejected shots.</p>
                  <div class="mt-2">
                    <SearchSelect
                      id="blacklist-tag-search"
                      v-model="blacklistPick"
                      aria-label="Search exclude tags"
                      placeholder="Search tags…"
                      empty-text="No tag matches"
                      width-class="w-full"
                      :options="tagOptions(draft.blacklistTagIds)"
                    />
                  </div>
                  <div v-if="draft.blacklistTagIds.length" class="mt-2 flex flex-wrap gap-1.5">
                    <span
                      v-for="tagId in draft.blacklistTagIds"
                      :key="tagId"
                      class="inline-flex items-center gap-1 rounded-full border border-red-400 bg-red-500/10 py-0.5 pl-2.5 pr-1 text-xs font-medium text-red-700 dark:text-red-300"
                    >
                      {{ tagName(tagId) }}
                      <button
                        type="button"
                        class="flex h-4 w-4 cursor-pointer items-center justify-center rounded-full transition-colors hover:bg-red-500/25"
                        :aria-label="`Remove tag ${tagName(tagId)}`"
                        @click="draft.blacklistTagIds = draft.blacklistTagIds.filter((t) => t !== tagId)"
                      >
                        <XMarkIcon class="h-3 w-3" />
                      </button>
                    </span>
                  </div>
                </div>

                <label class="block">
                  <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Blocked images</span>
                  <span class="mt-0.5 block text-xs text-primary-500 dark:text-primary-400">One file name (or image id) per line — never downloaded by this config.</span>
                  <textarea v-model="blockedText" rows="3" class="input-field mt-1 font-mono text-xs" placeholder="FSG26_1234_max&#10;FSG26_1235_max"></textarea>
                </label>
                <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <label class="block">
                    <span class="text-sm font-medium text-primary-700 dark:text-primary-200">Folder strategy</span>
                    <span class="mt-0.5 block text-xs text-primary-500 dark:text-primary-400">How photos are placed into folders.</span>
                    <select v-model="folderChoice" class="input-field mt-1">
                      <option value="none">None</option>
                      <option value="date">Capture date</option>
                      <option value="weekday">Event day (from date tag)</option>
                    </select>
                  </label>
                  <label class="flex items-start gap-2.5">
                    <input v-model="draft.deltaSubfolder" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-primary-300 text-accent-600 focus:ring-accent-500" />
                    <span>
                      <span class="block text-sm font-medium text-primary-700 dark:text-primary-200">Delta subfolder</span>
                      <span class="block text-xs text-primary-500 dark:text-primary-400"
                        >Delta runs prefix day folders: new files into <code>new_…/</code>, changed files into <code>delta_…/</code>.</span
                      >
                    </span>
                  </label>
                </div>
              </div>
              <!-- footer -->
              <div class="flex flex-row-reverse flex-wrap gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button type="button" class="btn-primary" :disabled="draft.name.trim() === ''" @click="save">
                  {{ create ? "Add config" : "Save config" }}
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
import { ArrowDownTrayIcon, TrashIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { ref, watch } from "vue";
import SearchSelect, { SearchSelectOption } from "src/components/SearchSelect.vue";
import { DownloadConfig, ImageTag } from "src/types/api";
import { tagLabel } from "src/util/tagOrder";
import { DownloadConfigCreate, DownloadConfigUpdate } from "src/api/downloadConfigs";

interface Props {
  show: boolean;
  create: boolean;
  config?: DownloadConfig | null;
  projectTags: ImageTag[];
}

const props = withDefaults(defineProps<Props>(), { config: null });

const emit = defineEmits<{
  closed: [];
  save: [DownloadConfigCreate | DownloadConfigUpdate];
  deleted: [];
}>();

const draft = ref({ name: "", whitelistTagIds: [] as string[], blacklistTagIds: [] as string[], deltaSubfolder: false });
const blockedText = ref("");
// folderChoice folds folderStructure + groupByDate into one strategy select —
// the two backing fields are mutually exclusive by construction.
const folderChoice = ref<"none" | "date" | "weekday">("none");

watch(
  () => [props.show, props.config, props.create] as const,
  () => {
    if (!props.show) return;
    if (props.config && !props.create) {
      draft.value = {
        name: props.config.name,
        whitelistTagIds: [...props.config.whitelistTagIds],
        blacklistTagIds: [...props.config.blacklistTagIds],
        deltaSubfolder: props.config.deltaSubfolder,
      };
      blockedText.value = props.config.blockedImageIds.join("\n");
      if (props.config.folderStructure === "weekday") folderChoice.value = "weekday";
      else if (props.config.groupByDate) folderChoice.value = "date";
      else folderChoice.value = "none";
    } else {
      draft.value = { name: "", whitelistTagIds: [], blacklistTagIds: [], deltaSubfolder: false };
      blockedText.value = "";
      folderChoice.value = "none";
    }
  },
  { immediate: true },
);

// Any concrete tag type is a valid download filter (albums, custom "error",
// internal…). Template tags ($DATE, $WEEKDAY…) are resolved at upload time —
// no image ever carries the template itself, so offering them here would
// silently match nothing.
function tagOptions(exclude: string[]): SearchSelectOption[] {
  return props.projectTags
    .filter((t) => t.type !== "template" && !exclude.includes(t.id))
    .map((t) => ({ value: t.id, label: t.name, hint: t.type }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

const whitelistPick = ref("");
const blacklistPick = ref("");
watch(whitelistPick, (id) => {
  if (id && !draft.value.whitelistTagIds.includes(id)) draft.value.whitelistTagIds.push(id);
  if (id) whitelistPick.value = "";
});
watch(blacklistPick, (id) => {
  if (id && !draft.value.blacklistTagIds.includes(id)) draft.value.blacklistTagIds.push(id);
  if (id) blacklistPick.value = "";
});

function tagName(id: string): string {
  const tag = props.projectTags.find((t) => t.id === id);
  return tag ? tagLabel(tag) : id;
}

function save() {
  emit("save", {
    name: draft.value.name.trim(),
    whitelistTagIds: draft.value.whitelistTagIds,
    blacklistTagIds: draft.value.blacklistTagIds,
    blockedImageIds: blockedText.value
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line !== ""),
    deltaSubfolder: draft.value.deltaSubfolder,
    groupByDate: folderChoice.value === "date",
    // always sent explicitly — an omitted field would keep a stale "weekday"
    // on the server when switching back to date/none
    folderStructure: folderChoice.value === "weekday" ? "weekday" : "default",
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
