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
        <div class="fixed inset-0 bg-gray-500 dark:bg-gray-900 bg-opacity-75 dark:bg-opacity-75 transition-opacity"></div>
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
            <DialogPanel class="relative transform overflow-hidden rounded-lg bg-white dark:!bg-gray-800 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-3xl">
              <div class="bg-white dark:!bg-gray-800 px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
                <div class="sm:flex sm:items-start">
                  <div class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-green-100 dark:bg-green-900 sm:mx-0 sm:h-10 sm:w-10">
                    <TagIcon class="h-6 w-6 text-green-600" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                    <DialogTitle as="h3" class="text-base font-semibold leading-6 text-primary-900 dark:text-primary-300">Add Bulk Tags as CSV</DialogTitle>
                    <div class="mt-2">
                      <textarea v-model="bulkText" placeholder="<name>,<description>,<type>" class="w-[40rem]"></textarea>
                    </div>
                  </div>
                </div>
              </div>

              <div class="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="bg-error-600 hover:bg-error-500 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto"
                  @click="emit('closed')"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  class="bg-green-600 hover:bg-green-500 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto"
                  @click="saveTags"
                >
                  Create Bulk Tags
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
import { TagIcon } from "@heroicons/vue/24/outline";
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
