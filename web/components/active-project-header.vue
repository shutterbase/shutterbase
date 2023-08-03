<template>
  <div v-if="project" class="btn btn-ghost normal-case text-xl" @click="navigateToProject">{{ project.name }}</div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { Project } from "~/api/project";
import { requestSingle } from "~/api/common";
const store = useStore();
const { activeProjectId } = storeToRefs(store);

const project = ref<Project | null>(null);

watch(activeProjectId, () => {
  loadActiveProject();
});

async function loadActiveProject() {
  if (activeProjectId.value && activeProjectId.value !== "") {
    const { item } = await requestSingle<Project>(`/projects/${activeProjectId.value}`);
    if (item) {
      project.value = item;
      store.setActiveProjectName(item.name);
    }
  }
}

onMounted(() => {
  loadActiveProject();
});

function navigateToProject() {
  navigateTo(`/dashboard/projects/${activeProjectId.value}`);
}
</script>
