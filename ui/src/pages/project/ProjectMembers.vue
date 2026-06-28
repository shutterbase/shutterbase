<template>
  <main class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl">
      <p class="label-mono text-accent-600 dark:text-accent-400">Project</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Members</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">People with access to this project.</p>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import { api } from "src/api";
const route = useRoute();

const project: Ref<ProjectsResponse | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadData() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log("No project ID provided");
    return;
  }

  try {
    console.log(`Loading project ${itemId}`);
    const response = await api.projects.get(itemId);
    project.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(route, loadData);
onMounted(loadData);
</script>
