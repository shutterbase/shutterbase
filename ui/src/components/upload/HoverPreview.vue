<template>
  <div class="fixed inset-0 z-10 w-screen overflow-y-auto bg-primary-950/60 backdrop-blur-sm">
    <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
      <div
        class="relative transform overflow-hidden rounded-lg border border-primary-200 bg-surface text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark sm:my-8 sm:w-full sm:max-w-lg"
      >
        <div class="px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
          <div class="sm:flex sm:items-start">
            <div class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-error-100 dark:bg-error-950/40 sm:mx-0 sm:h-10 sm:w-10">
              <svg class="h-6 w-6 text-error-600 dark:text-error-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
                />
              </svg>
            </div>
            <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
              <h3 class="display text-base text-primary-900 dark:text-white" id="modal-title">Deactivate account</h3>
              <div class="mt-2">
                <p class="text-sm text-primary-500 dark:text-primary-400">Are you sure you want to deactivate your account? All of your data will be permanently removed. This action cannot be undone.</p>
              </div>
            </div>
          </div>
        </div>
        <div class="border-t border-primary-200 bg-surface-muted px-4 py-3 dark:border-primary-800 dark:bg-surface-dark-muted sm:flex sm:flex-row-reverse sm:px-6">
          <button
            type="button"
            class="inline-flex w-full cursor-pointer justify-center rounded-md bg-error-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-error-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-error-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 sm:ml-3 sm:w-auto"
          >
            Deactivate
          </button>
          <button
            type="button"
            class="mt-3 inline-flex w-full cursor-pointer justify-center rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white sm:mt-0 sm:w-auto"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
interface Props {
  imageUrl: string;
}
const props = withDefaults(defineProps<Props>(), {});


</script>
