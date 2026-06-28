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
        <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity" />
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
            <DialogPanel class="relative transform overflow-hidden rounded-lg border border-primary-200 bg-surface text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark sm:my-8 sm:w-full sm:max-w-lg">
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
                  <div
                    v-if="type === MessageType.SUCCESS"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-success-100 dark:bg-success-950/40 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <CheckCircleIcon class="h-6 w-6 text-success-600 dark:text-success-300" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.WARNING"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-warning-100 dark:bg-warning-950/40 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationTriangleIcon class="h-6 w-6 text-warning-600 dark:text-warning-300" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.ERROR"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-error-100 dark:bg-error-950/40 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationCircleIcon class="h-6 w-6 text-error-600 dark:text-error-300" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.CONFIRM_INFO"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-accent-100 dark:bg-accent-950/40 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <InformationCircleIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" aria-hidden="true" />
                  </div>
                  <div
                    v-else-if="type === MessageType.CONFIRM_WARNING"
                    class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-warning-100 dark:bg-warning-950/40 sm:mx-0 sm:h-10 sm:w-10"
                  >
                    <ExclamationCircleIcon class="h-6 w-6 text-warning-600 dark:text-warning-300" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                    <p class="label-mono text-accent-600 dark:text-accent-400">{{ kicker }}</p>
                    <DialogTitle as="h3" class="display mt-1 text-lg text-primary-900 dark:text-white">{{ headline }}</DialogTitle>
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
                class="border-t border-primary-200 bg-surface-muted px-4 py-3 dark:border-primary-800 dark:bg-surface-dark-muted sm:flex sm:flex-row-reverse sm:px-6"
              >
                <div class="sm:flex sm:flex-row-reverse">
                  <button
                    type="button"
                    :class="`${
                      type === MessageType.CONFIRM_INFO
                        ? 'bg-accent-600 hover:bg-accent-500 active:bg-accent-700 focus-visible:ring-accent-500'
                        : 'bg-error-600 hover:bg-error-500 active:bg-error-700 focus-visible:ring-error-500'
                    } inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 sm:ml-3 sm:w-auto`"
                    @click="emit('confirmed')"
                  >
                    {{ confirmText }}
                  </button>
                  <button
                    type="button"
                    class="mt-3 inline-flex w-full justify-center rounded-md border border-primary-200 bg-surface px-3 py-2 text-sm font-medium text-primary-700 shadow-sm transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white sm:mt-0 sm:w-auto"
                    @click="emit('closed')"
                    ref="cancelButtonRef"
                  >
                    {{ cancelText }}
                  </button>
                </div>
              </div>
              <div v-else class="border-t border-primary-200 bg-surface-muted px-4 py-3 dark:border-primary-800 dark:bg-surface-dark-muted sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="inline-flex w-full justify-center rounded-md bg-accent-600 px-3 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 sm:ml-3 sm:w-auto"
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
import { ExclamationTriangleIcon, ExclamationCircleIcon, CheckCircleIcon, InformationCircleIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { computed } from "vue";

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

const kicker = computed(() => {
  switch (props.type) {
    case MessageType.SUCCESS:
      return "Success";
    case MessageType.ERROR:
      return "Error";
    case MessageType.WARNING:
      return "Warning";
    case MessageType.CONFIRM_INFO:
    case MessageType.CONFIRM_WARNING:
      return "Confirm";
    default:
      return "Notice";
  }
});
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
