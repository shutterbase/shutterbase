<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <main class="max-w-3xl">
      <p class="label-mono text-accent-600 dark:text-accent-400">New project</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Create a project</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Set up the details, copyright, and AI options for your new project.</p>

      <div class="mt-10 space-y-12">
        <CreateGroup @edit="updateData" headline="Project Information" subtitle="General information concerning this project" :fields="informationFields" />
        <CreateGroup @edit="updateData" headline="Copyright Data" subtitle="Copyright information to be embedded into the EXIF data" :fields="copyrightFields" />
        <CreateGroup @edit="updateData" headline="AI Options" subtitle="Options for AI image tagging" :fields="aiFields" />
        <button
          @click="createProject"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
        >
          Create
        </button>
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import CreateGroup, { Field, FieldType, CreateData } from "src/components/CreateGroup.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { ProjectCreate } from "src/api/projects";
import { showNotificationToast } from "src/boot/mitt";

const router = useRouter();

const project = ref<ProjectsResponse>({} as ProjectsResponse);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function updateData(editData: CreateData<ProjectsResponse>) {
  project.value = { ...project.value, ...editData };
}

async function createProject() {
  try {
    console.log(`Creating project ${project.value.name}`);
    const response = await api.projects.create(project.value as unknown as ProjectCreate);
    const itemId = response.id;
    console.log(`Project created with ID ${itemId}`);
    showNotificationToast({ headline: `Project created`, type: "success" });
    await router.push({ name: "project-general", params: { id: itemId } });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ProjectsResponse>[] = [
  { key: "name", label: "Name", type: FieldType.TEXT },
  { key: "description", label: "Description", type: FieldType.TEXT },
];

const aiFields: Field<ProjectsResponse>[] = [{ key: "aiSystemMessage", label: "AI Message", type: FieldType.TEXT }];

const copyrightFields: Field<ProjectsResponse>[] = [
  { key: "copyright", label: "Copyright", type: FieldType.TEXT },
  { key: "copyrightReference", label: "Copyright reference", type: FieldType.TEXT },
  { key: "locationName", label: "Location name", type: FieldType.TEXT },
  { key: "locationCode", label: "Location code", type: FieldType.TEXT },
  { key: "locationCity", label: "Location city", type: FieldType.TEXT },
];
</script>
