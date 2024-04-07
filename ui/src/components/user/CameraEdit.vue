<template>
  <div>
    <div class="flex justify-between">
      <div>
        <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">{{ item.name }}</h2>
      </div>
      <div>
        <button
          v-if="!edit"
          type="button"
          :disabled="edit"
          :class="[
            `inline-flex w-full justify-center rounded-md py-1 px-4 text-sm font-semibol shadow-sm sm:w-auto`,
            `text-white bg-secondary-600 hover:bg-secondary-500 dark:hover:bg-secondary-700`,
          ]"
          @click="toTimeOffset"
        >
          Create Time Offset
        </button>
        <button
          v-if="edit"
          type="button"
          :class="[
            `inline-flex rounded-md px-4 py-1 font-semibold shadow-sm ring-1 ring-inset sm:ml-3 text-sm`,
            `text-gray-900 bg-error-100 dark:bg-error-500 ring-error-300 dark:ring-error-700 hover:bg-error-50 hover:dark:bg-error-600 dark:text-gray-100`,
          ]"
          @click="edit = false"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="edit && !hasEdits"
          :class="[
            `inline-flex w-full justify-center rounded-md py-1 px-4 text-sm font-semibol shadow-sm sm:ml-3 sm:w-auto`,
            `text-white bg-secondary-600 hover:bg-secondary-500 dark:hover:bg-secondary-700`,
          ]"
          @click="() => (edit ? saveEdit() : startEdit())"
        >
          {{ edit ? "Save" : "Edit" }}
        </button>
      </div>
    </div>
    <div>
      <dl class="mt-6 space-y-6 divide-y divide-gray-100 dark:divide-gray-700 border-t border-gray-200 dark:border-gray-600 text-sm leading-6">
        <div v-for="field in fields" :key="field.key" class="pt-3 sm:flex">
          <dt class="font-medium text-gray-900 dark:text-primary-200 sm:w-64 sm:flex-none sm:pr-6">{{ field.label }}</dt>
          <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
            <div v-if="!edit">
              <div v-if="_item" class="py-1.5 text-gray-900 dark:text-primary-200">{{ _item[field.key] }}</div>
              <div v-else class="animate-pulse h-2.5 bg-gray-300 rounded-full dark:bg-gray-700 w-64"></div>
            </div>
            <div v-else class="w-full">
              <input
                v-if="field.type === FieldType.TEXT"
                v-model="editData[field.key]"
                type="text"
                :class="[
                  `block w-full rounded-md border-0 py-1.5 focus:ring-2 focus:ring-inset shadow-sm ring-1 ring-inset sm:text-sm sm:leading-6`,
                  `text-gray-900 placeholder:text-gray-400 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-700`,
                ]"
              />
            </div>
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>
<script setup lang="ts">
import { CamerasResponse } from "src/types/pocketbase";
import { Ref, computed, ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();

interface Props {
  item: CamerasResponse;
}

const props = withDefaults(defineProps<Props>(), {});

const editData: Ref<CameraEditData> = ref({} as CameraEditData);
const hasEdits = computed(() => checkEdits());

const emit = defineEmits<{
  editAbort: [];
  editSave: [CamerasResponse, CameraEditData];
  editStart: [];
}>();

const edit = ref(false);
const fields: Field[] = [{ key: "name", label: "Name", type: FieldType.TEXT }];

const _item = computed(() => props.item as CamerasResponse);

function startEdit() {
  if (!props.item) return;

  const data: CameraEditData = {
    name: props.item.name,
  };
  editData.value = data;
  edit.value = true;
}

function checkEdits() {
  if (!props.item) return false;
  if (props.item.name !== editData.value.name) {
    return true;
  }
  return false;
}

function saveEdit() {
  edit.value = false;
  emit("editSave", _item.value, editData.value);
}

function toTimeOffset() {
  router.push({ name: "camera-time-offset", params: { cameraid: props.item.id } });
}
</script>
<script lang="ts">
export type Field = {
  key: keyof CamerasResponse;
  label: string;
  type: FieldType;
};

export enum FieldType {
  TEXT = "text",
}

export type CameraEditData = {
  [key in keyof CamerasResponse]?: any;
};
</script>
