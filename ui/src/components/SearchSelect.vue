<!--
  Searchable single-select. Headless UI's Combobox (already a dependency) does the
  a11y and keyboard work — roles, active-option tracking, type-ahead, Esc/blur —
  so this file is styling plus the filter predicate.

  The empty string is a legitimate value: give it an option ("All photographers")
  and selecting it clears the selection.
-->
<template>
  <Combobox :model-value="modelValue" :disabled="disabled" @update:model-value="select">
    <div class="relative">
      <div class="relative">
        <ComboboxInput
          :id="id"
          :aria-label="ariaLabel"
          :class="[
            'h-9 w-full cursor-pointer rounded-md border border-primary-200 bg-surface pl-3 pr-9 text-sm text-primary-900 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:hover:border-primary-600',
            widthClass,
          ]"
          :display-value="displayValue"
          :placeholder="placeholder"
          autocomplete="off"
          @change="query = $event.target.value"
          @blur="query = ''"
        />
        <ComboboxButton class="absolute inset-y-0 right-0 flex items-center px-2 text-primary-400 hover:text-primary-600 dark:hover:text-primary-200">
          <ChevronUpDownIcon class="h-5 w-5" aria-hidden="true" />
          <span class="sr-only">Toggle options</span>
        </ComboboxButton>
      </div>

      <transition leave-active-class="transition ease-in duration-100" leave-from-class="opacity-100" leave-to-class="opacity-0" @after-leave="query = ''">
        <ComboboxOptions
          class="absolute z-20 mt-1 max-h-72 w-full min-w-max overflow-auto rounded-md border border-primary-200 bg-surface py-1 text-sm shadow-panel focus:outline-none dark:border-primary-700 dark:bg-surface-dark dark:shadow-panel-dark"
        >
          <p v-if="filtered.length === 0" class="px-3 py-2 text-sm text-primary-500 dark:text-primary-400">{{ emptyText }}</p>
          <ComboboxOption v-for="option in filtered" :key="option.value" v-slot="{ active, selected }" :value="option.value" as="template">
            <li
              :class="[
                'flex cursor-pointer items-center justify-between gap-3 px-3 py-2',
                active ? 'bg-accent-500/10 text-accent-700 dark:bg-accent-500/15 dark:text-accent-200' : 'text-primary-800 dark:text-primary-100',
              ]"
            >
              <span class="flex min-w-0 flex-col">
                <span :class="['truncate', selected ? 'font-semibold' : '']">{{ option.label }}</span>
                <span v-if="option.hint" class="truncate font-data text-xs text-primary-500 dark:text-primary-400">{{ option.hint }}</span>
              </span>
              <CheckIcon v-if="selected" class="h-4 w-4 shrink-0 text-accent-600 dark:text-accent-400" aria-hidden="true" />
            </li>
          </ComboboxOption>
        </ComboboxOptions>
      </transition>
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Combobox, ComboboxButton, ComboboxInput, ComboboxOption, ComboboxOptions } from "@headlessui/vue";
import { CheckIcon, ChevronUpDownIcon } from "@heroicons/vue/24/outline";

export interface SearchSelectOption {
  value: string;
  label: string;
  hint?: string;
}

const props = withDefaults(
  defineProps<{
    modelValue: string;
    options: SearchSelectOption[];
    id?: string;
    ariaLabel?: string;
    placeholder?: string;
    emptyText?: string;
    disabled?: boolean;
    widthClass?: string;
  }>(),
  { placeholder: "Select…", emptyText: "No match", disabled: false, widthClass: "sm:w-64" },
);

const emit = defineEmits<{ "update:modelValue": [value: string] }>();

const query = ref("");

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return props.options;
  return props.options.filter((o) => `${o.label} ${o.hint ?? ""}`.toLowerCase().includes(q));
});

function displayValue(value: unknown): string {
  return props.options.find((o) => o.value === value)?.label ?? "";
}

function select(value: string | null) {
  query.value = "";
  emit("update:modelValue", value ?? "");
}
</script>
