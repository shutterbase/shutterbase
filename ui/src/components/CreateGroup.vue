<template>
  <div>
    <div class="flex justify-between">
      <div>
        <h2 class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">{{ headline }}</h2>
        <p v-if="subtitle !== ''" class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-300">{{ subtitle }}</p>
      </div>
    </div>
    <div>
      <dl class="mt-6 space-y-6 divide-y divide-gray-100 dark:divide-gray-700 border-t border-gray-200 dark:border-gray-600 text-sm leading-6">
        <div v-for="field in fields" :key="field.key" class="pt-3 sm:flex">
          <dt class="font-medium text-gray-900 dark:text-primary-200 sm:w-64 sm:flex-none sm:pr-6">{{ field.label }}</dt>
          <dd class="mt-1 flex justify-between gap-x-6 sm:mt-0 sm:flex-auto">
            <div class="w-full">
              <input
                v-if="field.type === FieldType.TEXT"
                v-model="createData[field.key]"
                type="text"
                :class="[
                  `block w-full rounded-md border-0 py-1.5 focus:ring-2 focus:ring-inset shadow-sm ring-1 ring-inset sm:text-sm sm:leading-6`,
                  `text-gray-900 placeholder:text-gray-400 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-900`,
                ]"
              />
              <select
                v-else-if="field.type === FieldType.SELECT"
                v-model="createData[field.key]"
                :class="[
                  `block w-full rounded-md border-0 py-1.5 focus:ring-2 focus:ring-inset shadow-sm ring-1 ring-inset sm:text-sm sm:leading-6`,
                  `text-gray-900 placeholder:text-gray-400 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-900`,
                ]"
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
