<template>
  <div class="px-4 sm:px-6 lg:px-8">
    <div class="sm:flex sm:items-end sm:justify-between">
      <div class="sm:flex-auto">
        <h1 class="display text-3xl text-primary-900 dark:text-white">{{ capitalize(pluralName) }}</h1>
        <p v-if="subtitle" class="mt-2 text-sm text-primary-500 dark:text-primary-400">{{ subtitle }}</p>
      </div>
      <div v-if="addCallback && allowAdd" class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
        <button
          @click="addCallback"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-md bg-accent-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950"
        >
          <PlusIcon class="h-4 w-4" />
          Add {{ name }}
        </button>
      </div>
    </div>

    <div class="mt-7 flow-root">
      <div class="-mx-4 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full align-middle sm:px-6 lg:px-8">
          <table class="min-w-full border-separate border-spacing-0">
            <thead>
              <tr>
                <th
                  v-for="column in columns"
                  :key="column.key"
                  scope="col"
                  class="label-mono sticky top-0 z-10 border-b border-primary-200 bg-surface/85 px-3 py-3.5 text-left text-primary-500 backdrop-blur first:pl-1 dark:border-primary-800 dark:bg-surface-dark/85 dark:text-primary-400"
                >
                  {{ column.label }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="items.length === 0">
                <td :colspan="columns.length" :class="[rowPadding, 'px-3 text-left text-sm text-primary-500 dark:text-primary-400']">No {{ pluralName }} found</td>
              </tr>

              <tr v-for="item in items" :key="item.id" class="group transition-colors hover:bg-primary-50 dark:hover:bg-primary-800/40">
                <td
                  v-for="(column, columnIndex) in columns"
                  :key="Array.isArray(column.key) ? column.key.join('.') : column.key"
                  :class="[
                    rowPadding,
                    'whitespace-nowrap border-b border-primary-100 px-3 text-sm first:pl-1 dark:border-primary-800/70',
                    columnIndex === 0 ? 'font-medium text-primary-900 dark:text-white' : 'text-primary-700 dark:text-primary-300',
                  ]"
                >
                  <span v-if="column.key !== 'actions'">{{ getValue(item, column) }}</span>
                  <span v-else class="flex items-center gap-2">
                    <button
                      v-for="(action, actionIndex) in column.actions?.filter((action) => (action.showCallback ? action.showCallback(item) : true))"
                      :key="actionIndex"
                      @click="() => action.callback(item)"
                      :class="[actionBase, actionVariant(action.type)]"
                    >
                      {{ action.label }}
                    </button>
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends Identifiable">
import { plural } from "pluralize";
import { computed } from "vue";
import { PlusIcon } from "@heroicons/vue/24/outline";

interface Props {
  title?: string;
  subtitle?: string;

  allowAdd?: boolean;

  columns?: TableColumn<T>[];

  name?: string;

  dense?: boolean;

  items?: T[];

  addCallback?: () => void;

  cancelText?: string;
  headline?: string;
  message?: string;
}

const props = withDefaults(defineProps<Props>(), {
  title: () => "",
  subtitle: () => "",
  name: () => "item",
  allowAdd: () => true,
  dense: () => false,
  columns: () => [],
  items: () => [],
});

const pluralName = plural(props.name);
const rowPadding = computed(() => (props.dense ? "py-2.5" : "py-3.5"));

// shared action-button styling, on the design tokens (the old `secondary-*` colour
// never existed in the palette, so these buttons used to render as invisible white text).
const actionBase = "inline-flex items-center rounded-md border px-2.5 py-1 text-xs font-medium shadow-sm transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-accent-500";
function actionVariant(type: string): string {
  if (type === "delete") {
    return "border-error-300 bg-error-50 text-error-700 hover:bg-error-100 dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300 dark:hover:bg-error-950/70";
  }
  if (type === "custom") {
    return "border-accent-400/50 bg-accent-500/10 text-accent-700 hover:bg-accent-500/20 dark:border-accent-400/40 dark:text-accent-200";
  }
  // edit / default — quiet bordered button
  return "border-primary-200 bg-surface text-primary-700 hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white";
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function getValue(obj: T, column: TableColumn<T>): any {
  if (column.key === "actions") {
    return null;
  }

  const getValueFromObject = (obj: T, key: keyof T | string[]) => {
    if (Array.isArray(key)) {
      const anyObj = obj as any;
      return key.reduce((acc, key) => acc[key], anyObj);
    } else {
      return obj[key];
    }
  };

  const plainValue = getValueFromObject(obj, column.key);
  if (column.formatter) {
    return column.formatter(plainValue);
  } else {
    return plainValue;
  }
}
</script>

<script lang="ts">
export type Identifiable = { id: string };

export type TableColumn<T> = {
  key: keyof T | "actions" | string[];
  label: string;
  actions?: TableRowAction<T>[];
  formatter?: (item: any) => string;
};
export type TableRowAction<T> = {
  key: string;
  label: string;
  type: TableRowActionType;
  showCallback?: (item: T) => boolean;
  callback: (item: T) => void;
};
export enum TableRowActionType {
  EDIT = "edit",
  DELETE = "delete",
  CUSTOM = "custom",
}
</script>
