<template>
  <TransitionRoot as="template" :show="show">
    <Dialog as="div" class="relative z-10" @close="emit('closed')">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity"></div>
      </TransitionChild>

      <div class="fixed inset-0 z-10 w-screen overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enter-to="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 translate-y-0 sm:scale-100"
            leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel class="relative transform overflow-hidden rounded-lg border border-primary-200 bg-surface text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark sm:my-8 sm:w-full sm:max-w-3xl">
              <button
                type="button"
                @click="emit('closed')"
                class="absolute right-3 top-3 inline-flex rounded-md p-1.5 text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500"
              >
                <span class="sr-only">Close</span>
                <XMarkIcon class="h-5 w-5" aria-hidden="true" />
              </button>
              <div class="px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
                <div class="sm:flex sm:items-start">
                  <div class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-error-100 dark:bg-error-950/40 sm:mx-0 sm:h-10 sm:w-10">
                    <ExclamationTriangleIcon class="h-6 w-6 text-error-600 dark:text-error-400" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                    <p class="label-mono text-error-600 dark:text-error-400">Error</p>
                    <DialogTitle as="h3" class="display mt-1 text-lg text-primary-900 dark:text-white">{{ computedHeadline }}</DialogTitle>
                    <div class="mt-2">
                      <p class="text-sm text-primary-500 dark:text-primary-400">
                        {{ computedMessage }}
                      </p>
                    </div>
                    <button v-if="!showDetails" @click="showDetails = true" class="font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400">Show details</button>
                    <button v-if="showDetails" @click="showDetails = false" class="font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400">Hide details</button>
                    <button v-if="showDetails" @click="copyError" class="font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400 ml-4">{{ copyErrorText }}</button>
                  </div>
                </div>
                <pre v-if="showDetails" class="mt-3 rounded-md border border-primary-200 bg-surface-muted p-4 text-sm font-mono text-primary-700 dark:border-primary-800 dark:bg-surface-dark-muted dark:text-primary-300 text-wrap">{{ detailText }}</pre>
              </div>

              <div class="border-t border-primary-200 bg-surface-muted px-4 py-3 dark:border-primary-800 dark:bg-surface-dark-muted sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="inline-flex w-full justify-center rounded-md bg-accent-600 px-3 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 sm:ml-3 sm:w-auto"
                  @click="emit('closed')"
                >
                  {{ buttonText }}
                </button>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ExclamationTriangleIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";

interface Props {
  show: boolean;
  buttonText?: string;
  headline?: string;
  message?: string;
  error?: any;
}

interface ErrorResponse {
  status: number;
  response: {
    code: number;
    message: string;
    data: Record<
      string,
      {
        code: string;
        message: string;
      }
    >;
  };
  isAbort: boolean;
  originalError?: {
    url: string;
    status: number;
    data: any;
  };
}

const props = withDefaults(defineProps<Props>(), {
  buttonText: () => "OK",
  headline: () => "",
  message: () => "",
  error: () => null,
});

const computedHeadline = computed(() => {
  const e = props.error as ErrorResponse;
  if (props.headline && props.headline !== "") {
    return props.headline;
  }
  if (!e) {
    return "Unexpected Error";
  }
  if (e.response?.data) {
    if (Object.keys(e.response?.data).length === 1) {
      return `Error on field '${Object.keys(e.response.data)[0]}': ${Object.values(e.response.data)[0].message}`;
    }

    if (Object.keys(e.response.data).length >= 1) {
      let errorMessages = {} as Record<string, boolean>;
      for (const [key, value] of Object.entries(e.response.data)) {
        errorMessages[value.message] = true;
      }
      if (Object.keys(errorMessages).length === 1) {
        return `Error on ${Object.keys(e.response.data).length} fields: ${Object.keys(errorMessages)[0]}`;
      } else {
        return `Multiple errors on ${Object.keys(e.response.data).length} fields`;
      }
    }
  }
});

const computedMessage = computed(() => {
  return props.message && props.message !== "" ? props.message : "Something went wrong. More details can be found below";
});

const emit = defineEmits<{
  closed: [];
}>();

const detailText = computed(() => {
  return props.error ? JSON.stringify(props.error, null, 2).trim() : "No details available";
});
const showDetails = ref(false);

const copyErrorText = ref("Copy error details");
function copyError() {
  navigator.clipboard.writeText(detailText.value);
  copyErrorText.value = "Copied!";
  setTimeout(() => {
    copyErrorText.value = "Copy error details";
  }, 2000);
}
</script>
