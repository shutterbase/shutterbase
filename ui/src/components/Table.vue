<template>
  <div class="px-4 sm:px-6 lg:px-8">
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h1 class="text-base font-semibold leading-6 text-gray-900 dark:text-gray-100">{{ capitalize(pluralName) }}</h1>
        <p class="mt-2 text-sm text-gray-700 dark:text-gray-300">{{ subtitle }}</p>
      </div>
      <div v-if="addCallback" class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
        <button
          @click="addCallback"
          type="button"
          class="block rounded-md bg-secondary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
        >
          Add {{ name }}
        </button>
      </div>
    </div>
    <div class="mt-8 flow-root">
      <div class="-mx-4 -my-2 sm:-mx-6 lg:-mx-8">
        <div class="inline-block min-w-full py-2 align-middle">
          <table class="min-w-full border-separate border-spacing-0">
            <thead>
              <tr>
                <th
                  v-for="(column, columnIndex) in columns"
                  :key="column.key"
                  scope="col"
                  class="sticky top-0 z-10 border-b border-gray-300 dark:dark:border-primary-400 bg-gray-50 dark:bg-primary-900 bg-opacity-75 py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-gray-200 backdrop-blur backdrop-filter sm:pl-6 lg:pl-8"
                >
                  {{ column.label }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="items.length === 0">
                <td :colspan="columns.length" :class="[rowPadding, 'text-sm font-medium text-gray-900 dark:text-gray-200 text-left']">No {{ pluralName }} found</td>
              </tr>

              <tr v-for="item in items" :key="item.id" class="even:bg-gray-200 even:dark:bg-primary-700">
                <td
                  v-for="(column, columnIndex) in columns"
                  :key="column.key"
                  :class="[rowPadding, 'whitespace-nowrap pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-gray-200 sm:pl-6 lg:pl-8']"
                >
                  <span v-if="column.key !== 'actions'">{{ item[column.key] }}</span>
                  <span v-else class="flex">
                    <span v-for="(action, actionIndex) in column.actions" :key="actionIndex" @click="() => action.callback(item)">
                      <button
                        v-if="action.type === 'edit'"
                        :class="[
                          actionIndex === 0 ? '' : 'ml-2',
                          'block rounded-sm px-2 text-center text-sm font-semibold shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
                          'bg-secondary-600 text-white hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline-secondary-600',
                        ]"
                      >
                        {{ action.label }}
                      </button>
                      <button
                        v-if="action.type === 'delete'"
                        :class="[
                          actionIndex === 0 ? '' : 'ml-2',
                          'block rounded-sm px-2 text-center text-sm font-semibold shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
                          'bg-error-600 text-white hover:bg-error-500 dark:hover:bg-error-700 focus-visible:outline-error-600',
                        ]"
                      >
                        {{ action.label }}
                      </button>
                      <button
                        v-if="action.type === 'custom'"
                        :class="[
                          actionIndex === 0 ? '' : 'ml-2',
                          'block rounded-sm px-2 text-center text-sm font-semibold shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
                          'bg-primary-600 dark:bg-primary-900 text-white hover:bg-primary-500 dark:hover:bg-primary-950 focus-visible:outline-primary-600',
                        ]"
                      >
                        {{ action.label }}
                      </button>
                    </span>
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
  allowAdd: () => false,
  dense: () => false,
  columns: () => [],
  items: () => [],
});

const pluralName = plural(props.name);
const rowPadding = computed(() => (props.dense ? "py-2" : "py-4"));

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
</script>

<script lang="ts">
export type Identifiable = { id: string };

export type TableColumn<T> = {
  key: keyof T | "actions";
  label: string;
  actions?: TableRowAction<T>[];
};
export type TableRowAction<T> = {
  key: string;
  label: string;
  type: TableRowActionType;
  callback: (item: T) => void;
};
export enum TableRowActionType {
  EDIT = "edit",
  DELETE = "delete",
  CUSTOM = "custom",
}
</script>
