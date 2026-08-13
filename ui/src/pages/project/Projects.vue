<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table
      dense
      :items="items"
      :columns="columns"
      name="Project"
      :subtitle="activeProjectText"
      :allow-add="userStore.isAdmin()"
      :add-callback="() => router.push('/projects/create')"
    ></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />

    <!-- User feedback: people activated a project and didn't notice anything
         happened. Hard to miss now. -->
    <ModalMessage
      :show="activatedProject !== null"
      :type="MessageType.SUCCESS"
      headline="Project activated"
      :message="`'${activatedProject?.name}' is now your active project — images, uploads and tags are scoped to it.`"
      closeText="Let's go"
      @closed="activatedProject = null"
    />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import { api } from "src/api";
import { ProjectsResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { celebrate } from "src/util/confetti";
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
    const resultList = await api.projects.list({ limit: 50 });
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const activatedProject = ref<ProjectsResponse | null>(null);
const activating = ref(false);

async function activateProject(item: ProjectsResponse) {
  // single-flight: two overlapping activations could commit one project
  // server-side while the slower response claims the other locally
  if (activating.value) return;
  activating.value = true;
  try {
    await api.users.setActiveProject(item.id);
    userStore.setProject(item);
    activatedProject.value = item;
    celebrate();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    activating.value = false;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
