<template>
  <button
    @click="showModal = true"
    type="button"
    class="text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 me-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800"
  >
    Open Success Message
  </button>
  <modal-message v-if="showModal" @closed="showModal = false" message="Success message" />
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
