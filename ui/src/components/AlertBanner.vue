<template>
  <div class="mx-auto py-12">
    <div class="mx-auto">
      <div :class="['rounded-md p-4', backgroundColor]">
        <div class="flex">
          <div class="flex-shrink-0">
            <ExclamationCircleIcon v-if="type === AlertBannerType.ERROR" class="h-5 w-5 text-red-400" />
            <InformationCircleIcon v-else-if="type === AlertBannerType.INFO" class="h-5 w-5 text-blue-400" />
            <CheckCircleIcon v-else-if="type === AlertBannerType.SUCCESS" class="h-5 w-5 text-green-400" />
            <ExclamationTriangleIcon v-else-if="type === AlertBannerType.WARNING" class="h-5 w-5 text-yellow-400" />
          </div>
          <div class="ml-3">
            <h3 :class="['text-sm font-medium', headlineColor]">{{ headline }}</h3>
            <div :class="['mt-2 text-sm', messageColor]">
              {{ message }}
            </div>
            <div v-if="actions.length !== 0" class="mt-4">
              <div class="-mx-2 -my-1.5 flex">
                <button
                  v-for="action in actions"
                  :key="action.text"
                  @click="action.onClick"
                  type="button"
                  :class="['rounded-md px-2 py-1.5 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 ', buttonColor]"
                >
                  {{ action.text }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ExclamationTriangleIcon, ExclamationCircleIcon, CheckCircleIcon, InformationCircleIcon } from "@heroicons/vue/24/outline";
import { computed } from "vue";
import { RouteLocationRaw } from "vue-router";

interface Props {
  show?: boolean;
  type: AlertBannerType;
  actions?: AlertBannerAction[];
  headline: string;
  message: string;
}

const props = withDefaults(defineProps<Props>(), {
  show: () => true,
  type: () => AlertBannerType.ERROR,
  headline: () => "Something went wrong",
  message: () => "Please try again later",
  linkText: () => "Details",
  actions: () => [],
});

const headlineColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "text-red-800 dark:text-red-300";
    case AlertBannerType.INFO:
      return "text-blue-800 dark:text-blue-300";
    case AlertBannerType.SUCCESS:
      return "text-green-800 dark:text-green-300";
    case AlertBannerType.WARNING:
      return "text-yellow-800 dark:text-yellow-300";
  }
});

const messageColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "text-red-700 dark:text-red-200";
    case AlertBannerType.INFO:
      return "text-blue-700 dark:text-blue-200";
    case AlertBannerType.SUCCESS:
      return "text-green-700 dark:text-green-200";
    case AlertBannerType.WARNING:
      return "text-yellow-700 dark:text-yellow-200";
  }
});

const backgroundColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "bg-red-50 dark:bg-red-900";
    case AlertBannerType.INFO:
      return "bg-blue-50 dark:bg-blue-900";
    case AlertBannerType.SUCCESS:
      return "bg-green-50 dark:bg-green-900";
    case AlertBannerType.WARNING:
      return "bg-yellow-50 dark:bg-yellow-900";
  }
});

const buttonColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "bg-red-100 dark:bg-red-900 hover:bg-red-200 dark:hover:bg-red-950 text-red-800 dark:text-red-200 focus:ring-red-600 dark:focus:ring-red-400 focus:ring-offset-red-50 dark:focus:ring-offset-red-950";
    case AlertBannerType.INFO:
      return "bg-blue-100 dark:bg-blue-900 hover:bg-blue-200 dark:hover:bg-blue-950 text-blue-800 dark:text-blue-200 focus:ring-blue-600 dark:focus:ring-blue-400 focus:ring-offset-blue-50 dark:focus:ring-offset-blue-950";
    case AlertBannerType.SUCCESS:
      return "bg-green-100 dark:bg-green-900 hover:bg-green-200 dark:hover:bg-green-950 text-green-800 dark:text-green-200 focus:ring-green-600 dark:focus:ring-green-400 focus:ring-offset-green-50 dark:focus:ring-offset-green-950";
    case AlertBannerType.WARNING:
      return "bg-yellow-100 dark:bg-yellow-900 hover:bg-yellow-200 dark:hover:bg-yellow-950 text-yellow-800 dark:text-yellow-200 focus:ring-yellow-600 dark:focus:ring-yellow-400 focus:ring-offset-yellow-50 dark:focus:ring-offset-yellow-950";
  }
});
</script>

<script lang="ts">
export enum AlertBannerType {
  SUCCESS = "success",
  INFO = "info",
  ERROR = "error",
  WARNING = "warning",
}

export type AlertBannerAction = {
  text: string;
  onClick: () => void;
};
</script>
