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
        <div class="fixed inset-0 bg-gray-800 bg-opacity-75 transition-opacity" />
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
                  <div
                    v-if="type === MessageType.SUCCESS"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-secondary-100 dark:bg-secondary-900 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <CheckCircleIcon class="h-6 w-6 text-secondary-600 dark:text-secondary-200" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.WARNING"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-warning-100 dark:bg-warning-600 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationTriangleIcon class="h-6 w-6 text-warning-600 dark:text-warning-200" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.ERROR"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-error-100 dark:bg-error-600 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationCircleIcon class="h-6 w-6 text-error-600 dark:text-error-200" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.CONFIRM_INFO"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <InformationCircleIcon class="h-6 w-6 text-primary-500" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.CONFIRM_WARNING"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-warning-100 dark:bg-warning-600 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationCircleIcon class="h-6 w-6 text-warning-600 dark:text-warning-200" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                    <DialogTitle as="h3" class="text-base font-semibold leading-6 text-primary-900 dark:text-primary-300">{{ headline }}</DialogTitle>
                    <div class="mt-2">
                      <p class="text-sm text-primary-500 dark:text-primary-400">
                        {{ message }}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
              <div
                v-if="type === MessageType.CONFIRM_INFO || type === MessageType.CONFIRM_WARNING"
                class="bg-gray-50 dark:!bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6"
              >
                <div class="bg-gray-50 dark:!bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
                  <button
                    type="button"
                    :class="`${
                      type === MessageType.CONFIRM_INFO ? 'bg-secondary-600 hover:bg-secondary-500' : 'bg-error-600 hover:bg-error-500'
                    } inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto`"
                    @click="emit('confirmed')"
                  >
                    {{ confirmText }}
                  </button>
                  <button
                    type="button"
                    class="mt-3 inline-flex w-full justify-center rounded-md bg-white dark:!bg-gray-800 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:mt-0 sm:w-auto"
                    @click="emit('closed')"
                    ref="cancelButtonRef"
                  >
                    {{ cancelText }}
                  </button>
                </div>
              </div>
              <div v-else class="bg-gray-50 dark:!bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="inline-flex w-full justify-center rounded-md bg-secondary-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 sm:ml-3 sm:w-auto"
                  @click="emit('closed')"
                >
                  {{ closeText }}
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
import { ExclamationTriangleIcon, ExclamationCircleIcon, CheckCircleIcon, InformationCircleIcon } from "@heroicons/vue/24/outline";

interface Props {
  show: boolean;
  type?: MessageType;
  closeText?: string;
  confirmText?: string;
  cancelText?: string;
  headline?: string;
  message?: string;
}

const props = withDefaults(defineProps<Props>(), {
  type: () => MessageType.SUCCESS,
  closeText: () => "Close",
  confirmText: () => "OK",
  cancelText: () => "Cancel",
  headline: () => "Success",
  message: () => "This worked!",
});

const emit = defineEmits<{
  closed: [];
  confirmed: [];
}>();
</script>

<script lang="ts">
export enum MessageType {
  SUCCESS = "success",
  ERROR = "error",
  WARNING = "warning",
  CONFIRM_WARNING = "confirm_warning",
  CONFIRM_INFO = "confirm_info",
}
</script>
