<template>
  <div class="mx-auto py-12">
    <div class="mx-auto">
      <div :class="['rounded-md p-4', backgroundColor]">
        <div class="flex">
          <div class="flex-shrink-0">
            <ExclamationCircleIcon v-if="type === AlertBannerType.ERROR" class="h-5 w-5 text-error-500 dark:text-error-400" />
            <InformationCircleIcon v-else-if="type === AlertBannerType.INFO" class="h-5 w-5 text-accent-500 dark:text-accent-400" />
            <CheckCircleIcon v-else-if="type === AlertBannerType.SUCCESS" class="h-5 w-5 text-success-500 dark:text-success-400" />
            <ExclamationTriangleIcon v-else-if="type === AlertBannerType.WARNING" class="h-5 w-5 text-warning-500 dark:text-warning-400" />
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
      return "text-error-800 dark:text-error-300";
    case AlertBannerType.INFO:
      return "text-accent-800 dark:text-accent-300";
    case AlertBannerType.SUCCESS:
      return "text-success-800 dark:text-success-300";
    case AlertBannerType.WARNING:
      return "text-warning-800 dark:text-warning-300";
  }
});

const messageColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "text-error-700 dark:text-error-200";
    case AlertBannerType.INFO:
      return "text-accent-700 dark:text-accent-200";
    case AlertBannerType.SUCCESS:
      return "text-success-700 dark:text-success-200";
    case AlertBannerType.WARNING:
      return "text-warning-700 dark:text-warning-200";
  }
});

const backgroundColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "border border-error-200 bg-error-50 dark:border-error-900 dark:bg-error-950/40";
    case AlertBannerType.INFO:
      return "border border-accent-200 bg-accent-50 dark:border-accent-900 dark:bg-accent-950/40";
    case AlertBannerType.SUCCESS:
      return "border border-success-200 bg-success-50 dark:border-success-900 dark:bg-success-950/40";
    case AlertBannerType.WARNING:
      return "border border-warning-200 bg-warning-50 dark:border-warning-900 dark:bg-warning-950/40";
  }
});

const buttonColor = computed(() => {
  switch (props.type) {
    case AlertBannerType.ERROR:
      return "bg-error-100 dark:bg-error-950/40 hover:bg-error-200 dark:hover:bg-error-950/70 text-error-800 dark:text-error-200 focus:ring-error-600 dark:focus:ring-error-400 focus:ring-offset-error-50 dark:focus:ring-offset-error-950";
    case AlertBannerType.INFO:
      return "bg-accent-100 dark:bg-accent-950/40 hover:bg-accent-200 dark:hover:bg-accent-950/70 text-accent-800 dark:text-accent-200 focus:ring-accent-600 dark:focus:ring-accent-400 focus:ring-offset-accent-50 dark:focus:ring-offset-accent-950";
    case AlertBannerType.SUCCESS:
      return "bg-success-100 dark:bg-success-950/40 hover:bg-success-200 dark:hover:bg-success-950/70 text-success-800 dark:text-success-200 focus:ring-success-600 dark:focus:ring-success-400 focus:ring-offset-success-50 dark:focus:ring-offset-success-950";
    case AlertBannerType.WARNING:
      return "bg-warning-100 dark:bg-warning-950/40 hover:bg-warning-200 dark:hover:bg-warning-950/70 text-warning-800 dark:text-warning-200 focus:ring-warning-600 dark:focus:ring-warning-400 focus:ring-offset-warning-50 dark:focus:ring-offset-warning-950";
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
