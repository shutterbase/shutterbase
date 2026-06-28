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
                    <p class="label-mono text-accent-600 dark:text-accent-400">Project tag</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">{{ create ? "Add tag" : "Edit tag" }}</DialogTitle>
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
                <div v-if="create">
                  <CreateGroup headline="Tag details" @edit="updateData" :fields="createTagFields" />
                </div>
                <div v-else>
                  <DetailEditGroup headline="Tag details" :alwaysEdit="true" @edit="updateData" :fields="editTagFields" :item="item" />
                </div>
                <button
                  type="button"
                  @click="emit('bulk')"
                  class="mt-5 cursor-pointer text-sm font-medium text-accent-600 underline-offset-2 transition-colors hover:text-accent-500 hover:underline dark:text-accent-400"
                >
                  Create tags in bulk instead
                </button>
              </div>

              <!-- footer -->
              <div class="flex flex-row-reverse gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button
                  type="button"
                  class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
                  @click="saveTag"
                >
                  Save tag
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
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { TagIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import CreateGroup, { CreateData, Field as CreateField, FieldType as CreateFieldType } from "src/components/CreateGroup.vue";
import DetailEditGroup, { Field as EditField, FieldType as EditFieldType } from "src/components/DetailEditGroup.vue";
import { ImageTagsResponse } from "src/types/pocketbase";
import { computed, ref, Ref, watch } from "vue";

interface Props {
  show: boolean;
  create: boolean;
  tag?: ImageTagsResponse;
}

const props = withDefaults(defineProps<Props>(), {
  create: false,
});

const item = ref<ImageTagsResponse>({} as ImageTagsResponse);
watch(
  () => props.tag,
  (value) => {
    item.value = value ?? ({} as ImageTagsResponse);
  }
);

const emit = defineEmits<{
  add: [ImageTagsResponse];
  edit: [ImageTagsResponse];
  closed: [];
  bulk: [];
}>();

async function updateData(data: CreateData<ImageTagsResponse>) {
  item.value = { ...item.value, ...data };
}

function saveTag() {
  if (props.create) {
    if (item.value.name === "") {
      console.log("Name is required");
      return;
    }
    if (item.value.description === "") {
      console.log("Description is required");
      return;
    }

    emit("add", item.value);
  } else {
    emit("edit", item.value);
  }
}

const createTagFields: CreateField<ImageTagsResponse>[] = [
  { key: "name", label: "Name", type: CreateFieldType.TEXT },
  { key: "description", label: "Description", type: CreateFieldType.TEXT },
  { key: "type", label: "Type", type: CreateFieldType.SELECT, options: ["template", "default", "manual", "custom"], optionsDefault: "manual" },
];

const editTagFields: EditField<ImageTagsResponse>[] = [
  { key: "name", label: "Name", type: EditFieldType.TEXT },
  { key: "description", label: "Description", type: EditFieldType.TEXT },
  { key: "type", label: "Type", type: EditFieldType.SELECT, options: ["template", "default", "manual", "custom"] },
];
</script>
