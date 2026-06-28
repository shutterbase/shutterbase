<template>
  <div>
    <button
      id="multiLevelDropdownButton"
      data-dropdown-toggle="multi-dropdown"
      class="text-white bg-accent-600 hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 font-medium rounded-lg text-sm px-5 py-2.5 text-center inline-flex items-center"
      type="button"
    >
      Dropdown button
      <svg class="w-2.5 h-2.5 ms-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 10 6">
        <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 1 4 4 4-4" />
      </svg>
    </button>

    <!-- Dropdown menu -->
    <div id="multi-dropdown" class="z-10 hidden bg-surface divide-y divide-primary-100 rounded-lg shadow-panel border border-primary-200 w-44 dark:bg-surface-dark dark:divide-primary-800 dark:border-primary-800 dark:shadow-panel-dark">
      <ul class="py-2 text-sm text-primary-700 dark:text-primary-200" aria-labelledby="multiLevelDropdownButton">
        <li>
          <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Dashboard</a>
        </li>
        <li>
          <button
            id="doubleDropdownButton"
            data-dropdown-toggle="doubleDropdown"
            data-dropdown-placement="right-start"
            type="button"
            class="flex items-center justify-between w-full px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white"
          >
            Dropdown<svg class="w-2.5 h-2.5 ms-3 rtl:rotate-180" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 6 10">
              <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 9 4-4-4-4" />
            </svg>
          </button>
          <div id="doubleDropdown" class="z-10 hidden bg-surface divide-y divide-primary-100 rounded-lg shadow-panel border border-primary-200 w-44 dark:bg-surface-dark dark:divide-primary-800 dark:border-primary-800 dark:shadow-panel-dark">
            <ul class="py-2 text-sm text-primary-700 dark:text-primary-200" aria-labelledby="doubleDropdownButton">
              <li>
                <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Overview</a>
              </li>
              <li>
                <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">My downloads</a>
              </li>
              <li>
                <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Billing</a>
              </li>
              <li>
                <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Rewards</a>
              </li>
            </ul>
          </div>
        </li>
        <li>
          <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Earnings</a>
        </li>
        <li>
          <a href="#" class="block px-4 py-2 text-primary-700 transition-colors hover:bg-primary-100 hover:text-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 dark:hover:text-white">Sign out</a>
        </li>
      </ul>
    </div>
  </div>
</template>
<script setup>
import { onMounted, ref } from "vue";
import { initFlowbite } from "flowbite";
import ModalMessage from "src/components/ModalMessage.vue";

const showModal = ref(false);

// initialize components based on data attribute selectors
onMounted(() => {
  initFlowbite();
});
</script>
