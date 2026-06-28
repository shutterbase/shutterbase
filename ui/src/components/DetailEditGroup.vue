<template>
  <div>
    <div class="flex justify-between">
      <div>
        <h2 class="display text-xl text-primary-900 dark:text-white">{{ headline }}</h2>
        <p v-if="subtitle !== ''" class="mt-1 text-sm text-primary-500 dark:text-primary-400">{{ subtitle }}</p>
      </div>
      <div class="flex items-center gap-2" v-if="alwaysEdit === false && allowEdit">
        <button
          v-if="edit"
          type="button"
          class="inline-flex items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
          @click="edit = false"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="edit && !hasEdits"
          class="inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 disabled:opacity-50"
          @click="() => (edit ? saveEdit() : startEdit())"
        >
          {{ edit ? "Save" : "Edit" }}
        </button>
      </div>
    </div>
    <div>
      <dl class="mt-6 space-y-6 divide-y divide-primary-100 dark:divide-primary-800 text-sm leading-6">
        <div v-for="field in fields" :key="field.key" class="pt-3 sm:flex">
          <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6 sm:pt-2">{{ field.label }}</dt>
          <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
            <div v-if="!edit">
              <div v-if="_item" class="py-1.5 text-sm text-primary-800 dark:text-primary-100">{{ _item[field.key] }}</div>
              <div v-else class="animate-pulse h-2.5 bg-primary-200 rounded-full dark:bg-primary-800 w-64"></div>
            </div>
            <div v-else class="w-full">
              <input
                v-if="field.type === FieldType.TEXT"
                v-model="editData[field.key]"
                type="text"
                :aria-label="field.label"
                class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              />
              <select
                v-else-if="field.type === FieldType.SELECT"
                v-model="editData[field.key]"
                :aria-label="field.label"
                class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              >
                <option v-for="option in field.options" :key="option" :value="option" :selected="option === editData[field.key]">{{ option }}</option>
              </select>
            </div>
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends Identifiable">
import { Ref, UnwrapNestedRefs, computed, reactive, ref, watch } from "vue";

interface Props {
  headline: string;
  subtitle?: string;
  item: T | null;
  fields: Field<T>[];
  alwaysEdit?: boolean;
  allowEdit?: boolean;
}

const _item = computed(() => props.item as T);

const props = withDefaults(defineProps<Props>(), {
  subtitle: () => "",
  alwaysEdit: false,
  allowEdit: true,
});

const editData: UnwrapNestedRefs<EditData<T>> = reactive({} as EditData<T>);
const hasEdits = computed(() => checkEdits());

const emit = defineEmits<{
  editAbort: [];
  editSave: [EditData<T>];
  editStart: [];
  edit: [EditData<T>];
}>();

const edit = ref(props.alwaysEdit);

function startEdit() {
  if (!props.item) return;
  setEditData();
  edit.value = true;
}

watch(() => props.item, setEditData, { immediate: true });
function setEditData() {
  if (!props.item) return;

  for (const field of props.fields) {
    editData[field.key] = props.item[field.key];
  }

  if (props.alwaysEdit) {
    watch(editData, (newValue) => {
      emit("edit", newValue);
    });
  }
}

function checkEdits() {
  if (!props.item) return false;
  if (!editData) return false;
  for (const field of props.fields) {
    if (props.item[field.key] !== editData[field.key]) {
      return true;
    }
  }
  return false;
}

function saveEdit() {
  edit.value = false;
  emit("editSave", editData);
}
</script>

<script lang="ts">
export type Identifiable = { id: string };

export type Field<T> = {
  key: keyof T;
  label: string;
  type: FieldType;
  options?: string[];
};

export enum FieldType {
  TEXT = "text",
  SELECT = "select",
}

export type EditData<T> = {
  [key in keyof T]?: any;
};
</script>
