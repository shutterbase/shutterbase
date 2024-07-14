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
                    <div v-if="create" class="mt-2">
                      <CreateGroup headline="Add Tag" @edit="updateData" :fields="createTagFields" />
                    </div>
                    <div v-else>
                      <DetailEditGroup headline="Edit Tag" :alwaysEdit="true" @edit="updateData" :fields="editTagFields" :item="item" />
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
                  @click="saveTag"
                >
                  Save Tag
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
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { TagIcon } from "@heroicons/vue/24/outline";
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
