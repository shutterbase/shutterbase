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
                    <TagIcon class="h-5 w-5" aria-hidden="true" />
                  </span>
                  <div>
                    <p class="label-mono text-accent-600 dark:text-accent-400">Project tags</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">Bulk create tags</DialogTitle>
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
              <div class="px-6 py-5">
                <label for="bulk-tags" class="label-mono block text-primary-500 dark:text-primary-400">CSV input</label>
                <p class="mt-1.5 text-sm text-primary-500 dark:text-primary-400">One tag per line as <span class="font-data text-primary-700 dark:text-primary-200">name,description,type</span>.</p>
                <textarea
                  id="bulk-tags"
                  v-model="bulkText"
                  rows="8"
                  placeholder="<name>,<description>,<type>"
                  class="mt-3 block w-full rounded-md border border-primary-200 bg-surface px-3 py-2.5 font-data text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
                ></textarea>
              </div>

              <!-- footer -->
              <div class="flex flex-row-reverse gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button
                  type="button"
                  class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
                  @click="saveTags"
                >
                  Create tags
                </button>
                <button
                  type="button"
                  class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
                  @click="emit('closed')"
                >
                  Cancel
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
import { Dialog, DialogTitle, DialogPanel, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { TagIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import CreateGroup, { CreateData, Field as CreateField, FieldType as CreateFieldType } from "src/components/CreateGroup.vue";
import DetailEditGroup, { Field as EditField, FieldType as EditFieldType } from "src/components/DetailEditGroup.vue";
import { ImageTagsResponse } from "src/types/pocketbase";
import { computed, ref, Ref, watch } from "vue";

interface Props {
  show: boolean;
}
const props = withDefaults(defineProps<Props>(), {});

const bulkText = ref("");
watch(
  () => props.show,
  (show) => {
    if (show) {
      bulkText.value = "";
    }
  }
);

const emit = defineEmits<{
  add: [ImageTagsResponse[]];
  closed: [];
}>();

function saveTags() {
  const tags = bulkText.value.split("\n").flatMap((line) => {
    if (line.trim() === "") return [];
    const [name, description, type] = line.split(",");
    return { name, description, type } as ImageTagsResponse;
  });
  console.log(tags);
  emit("add", tags);
}
</script>
