<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
      <div class="pb-2">
        <h2 class="text-2xl font-semibold leading-7 text-primary-900 dark:text-primary-200">Project Tags</h2>
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
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
    const response = await pb.collection<ProjectsResponse>("projects").getOne(itemId);
    project.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(route, loadData);
onMounted(loadData);
</script>
