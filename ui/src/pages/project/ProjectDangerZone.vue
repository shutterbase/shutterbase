<template>
  <main class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl">
      <p class="label-mono text-error-600 dark:text-error-400">Caution</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Danger zone</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Irreversible actions for this project.</p>

      <div
        class="mt-6 flex flex-col gap-4 rounded-lg border border-error-300 bg-error-50/50 p-5 dark:border-error-800/70 dark:bg-error-950/30 sm:flex-row sm:items-center sm:justify-between"
      >
        <div>
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Delete project '{{ project?.name }}'</h2>
          <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">Remove this project and all its associated images. This cannot be undone.</p>
        </div>
        <button
          @click="showConfirmDialog = true"
          class="inline-flex shrink-0 cursor-pointer items-center justify-center gap-1.5 rounded-md border border-error-300 bg-error-50 px-4 py-2 text-sm font-medium text-error-700 transition-colors hover:bg-error-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-error-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300 dark:hover:bg-error-950/70 dark:focus-visible:ring-offset-primary-950"
        >
          Delete this project
        </button>
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  <ModalMessage
    :show="showConfirmDialog"
    @closed="showConfirmDialog = false"
    @confirmed="deleteItem"
    :headline="`Delete project ${project?.name}`"
    :message="`Are you sure you want to delete the project '${project?.name}' and all its images? THIS ACTION CANNOT BE UNDONE!`"
    :type="MessageType.CONFIRM_WARNING"
  />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
const route = useRoute();
const router = useRouter();

const project: Ref<ProjectsResponse | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const showConfirmDialog = ref(false);

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

async function deleteItem() {
  if (!project.value) {
    console.log("No project to delete");
    return;
  }

  try {
    await api.projects.remove(project.value.id);
    await router.push({ name: "projects" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(route, loadData);
onMounted(loadData);
</script>
