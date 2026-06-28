<template>
  <div>
    <div class="flex justify-between">
      <div>
        <h2 class="display text-xl text-primary-900 dark:text-white">{{ headline }}</h2>
        <p v-if="subtitle !== ''" class="mt-1 text-sm text-primary-500 dark:text-primary-400">{{ subtitle }}</p>
      </div>
    </div>
    <div>
      <dl class="mt-6 space-y-6 divide-y divide-primary-100 dark:divide-primary-800 border-t border-primary-200 dark:border-primary-800 text-sm leading-6">
        <div v-for="field in fields" :key="field.key" class="pt-3 sm:flex">
          <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-64 sm:flex-none sm:pr-6 sm:pt-2.5">{{ field.label }}</dt>
          <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
            <div class="w-full">
              <input
                v-if="field.type === FieldType.TEXT"
                v-model="createData[field.key]"
                type="text"
                :aria-label="field.label"
                class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              />
              <select
                v-else-if="field.type === FieldType.SELECT"
                v-model="createData[field.key]"
                :aria-label="field.label"
                class="h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              >
                <option v-for="option in field.options" :key="option" :value="option" :selected="option === field.optionsDefault">{{ option }}</option>
              </select>
            </div>
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>
<script setup lang="ts" generic="T extends Identifiable">
import { Ref, UnwrapNestedRefs, computed, onMounted, reactive, ref, watch } from "vue";

interface Props {
  headline: string;
  subtitle?: string;
  fields: Field<T>[];
}

const props = withDefaults(defineProps<Props>(), {
  subtitle: () => "",
});

const createData: UnwrapNestedRefs<CreateData<T>> = reactive({} as CreateData<T>);

onMounted(setSelectDefaults);
function setSelectDefaults() {
  props.fields.forEach((field) => {
    if (field.type === FieldType.SELECT && field.optionsDefault) {
      createData[field.key] = field.optionsDefault;
    }
  });
}

watch(createData, (newValue) => {
  emit("edit", newValue);
});

const emit = defineEmits<{
  edit: [CreateData<T>];
}>();
</script>

<script lang="ts">
export type Identifiable = { id: string };

export type Field<T> = {
  key: keyof T;
  label: string;
  type: FieldType;
  options?: string[];
  optionsDefault?: string;
};

export enum FieldType {
  TEXT = "text",
  SELECT = "select",
}

export type CreateData<T> = {
  [key in keyof T]?: any;
};
</script>
