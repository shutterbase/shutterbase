<template>
  <div>
    <div class="flex items-center justify-between gap-4">
      <div>
        <h2 class="display text-lg text-primary-900 dark:text-white">{{ item.name }}</h2>
      </div>
      <div class="flex items-center gap-3">
        <button
          v-if="!edit"
          type="button"
          :disabled="edit"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white disabled:opacity-50"
          @click="toTimeOffset"
        >
          Create Time Offset
        </button>
        <button
          v-if="edit"
          type="button"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
          @click="edit = false"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="edit && !hasEdits"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
          @click="() => (edit ? saveEdit() : startEdit())"
        >
          {{ edit ? "Save" : "Edit" }}
        </button>
      </div>
    </div>
    <div>
      <dl class="mt-6 space-y-6 divide-y divide-primary-100 dark:divide-primary-800 border-t border-primary-200 dark:border-primary-800 text-sm leading-6">
        <div v-for="field in fields" :key="field.key" class="pt-3 sm:flex">
          <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6">{{ field.label }}</dt>
          <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
            <div v-if="!edit">
              <div v-if="_item" class="py-1.5 text-primary-900 dark:text-primary-100">{{ _item[field.key] }}</div>
              <div v-else class="h-2.5 w-64 animate-pulse rounded-full bg-primary-200 dark:bg-primary-800"></div>
            </div>
            <div v-else class="w-full">
              <input
                v-if="field.type === FieldType.TEXT"
                v-model="editData[field.key]"
                type="text"
                class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
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
