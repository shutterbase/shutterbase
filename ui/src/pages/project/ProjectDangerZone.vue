<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
      <div class="pb-2">
        <h2 class="text-2xl font-semibold leading-7 text-primary-900 dark:text-primary-200">Danger Zone</h2>
      </div>
      <div class="flex justify-between border p-4 rounded-md border-red-600 dark:border-red-800">
        <div>
          <h2 class="text-lg font-semibold leading-7 text-primary-900 dark:text-primary-200">Delete project '{{ project?.name }}'</h2>
          <p class="mt-1 text-sm leading-6 text-primary-500 dark:text-primary-300">Remove this project and all its associated images</p>
        </div>
        <div class="flex items-center">
          <button
            @click="showConfirmDialog = true"
            :class="[
              `ring-1 ring-inset  rounded-md px-4 py-2  text-sm text-bold `,
              `ring-gray-300 bg-gray-200 text-error-600 hover:bg-error-700 hover:text-white`,
              `dark:ring-primary-950 dark:bg-primary-900 dark:hover:bg-error-700`,
            ]"
          >
            Delete this project
          </button>
        </div>
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
import pb from "src/boot/pocketbase";
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
    const response = await pb.collection<ProjectsResponse>("projects").getOne(itemId);
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
    await pb.collection<ProjectsResponse>("projects").delete(project.value.id);
    await router.push({ name: "projects" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(route, loadData);
onMounted(loadData);
</script>
