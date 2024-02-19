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
        <div class="fixed inset-0 bg-gray-500 dark:bg-gray-900 bg-opacity-75 dark:bg-opacity-75 transition-opacity" />
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
            <DialogPanel class="relative transform overflow-hidden rounded-lg bg-white dark:!bg-gray-800 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg">
              <div class="bg-white dark:!bg-gray-800 px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
                <div class="sm:flex sm:items-start">
                  <div class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-error-100 dark:bg-error-900 sm:mx-0 sm:h-10 sm:w-10">
                    <ExclamationTriangleIcon class="h-6 w-6 text-error-600" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                    <DialogTitle as="h3" class="text-base font-semibold leading-6 text-primary-900 dark:text-primary-300">{{ computedHeadline }}</DialogTitle>
                    <div class="mt-2">
                      <p class="text-sm text-primary-500">
                        {{ computedMessage }}
                      </p>
                    </div>
                    <button v-if="!showDetails" @click="showDetails = true" class="font-medium text-primary-600 hover:underline dark:text-primary-500">Show details</button>
                    <button v-if="showDetails" @click="showDetails = false" class="font-medium text-primary-600 hover:underline dark:text-primary-500">Hide details</button>
                    <button v-if="showDetails" @click="copyError" class="font-medium text-primary-600 hover:underline dark:text-primary-500 ml-4">{{ copyErrorText }}</button>
                  </div>
                </div>
                <pre v-if="showDetails" class="bg-gray-200 dark:bg-gray-700 dark:text-gray-300 border text-sm font-mono p-4 text-wrap">{{ detailText }}</pre>
              </div>

              <div class="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="bg-error-600 hover:bg-error-500 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto"
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
import { ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";

interface Props {
  show: boolean;
  buttonText?: string;
  headline?: string;
  message?: string;
  error?: any;
}

const props = withDefaults(defineProps<Props>(), {
  buttonText: () => "OK",
  headline: () => "",
  message: () => "",
  error: () => null,
});

const computedHeadline = computed(() => {
  return props.headline && props.headline !== "" ? props.headline : "Unexpected Error";
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
