<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table dense :items="items" :columns="columns" name="Project" :subtitle="activeProjectText" :add-callback="() => router.push('/projects/create')"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import pb from "src/boot/pocketbase";
import { ProjectsResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
const router = useRouter();

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<ProjectsResponse[]> = ref([]);
const columns: TableColumn<ProjectsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "description", label: "Description" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "activate", label: "Activate", callback: activateProject, type: TableRowActionType.CUSTOM },
      { key: "edit", label: "Details", callback: (item) => router.push({ name: `project-general`, params: { id: item.id } }), type: TableRowActionType.EDIT },
    ],
  },
];

const activeProjectText = computed(() => {
  if (activeProjectId.value && activeProjectId.value !== "") {
    if (items.value && items.value.length > 0) {
      const activeProject = items.value.find((item) => item.id === activeProjectId.value);
      if (activeProject) {
        return "Active project: " + activeProject.name;
      }
    } else {
      return "Active project ID: " + activeProjectId.value;
    }
  }
  return "No active project";
});

async function requestItems() {
  try {
    const resultList = await pb.collection<ProjectsResponse>("projects").getList(1, 50, {});
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function activateProject(item: ProjectsResponse) {
  const userId = pb.authStore.model?.id;
  if (!userId) {
    return;
  }

  try {
    await pb.collection("users").update(userId, { activeProject: item.id });
    userStore.activeProjectId = item.id;
    userStore.activeProject = item;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
