<template>
  <div class="mx-auto max-w-7xl lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
        <CreateGroup @edit="updateData" headline="Project Information" subtitle="General information concerning this project" :fields="informationFields" />
        <CreateGroup @edit="updateData" headline="Copyright Data" subtitle="Copyright information to be embedded into the EXIF data" :fields="copyrightFields" />
        <button
          @click="createProject"
          class="block rounded-md bg-secondary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
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
import pb from "src/boot/pocketbase";
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
    const response = await pb.collection<ProjectsResponse>("projects").create(project.value);
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

const copyrightFields: Field<ProjectsResponse>[] = [
  { key: "copyright", label: "Copyright", type: FieldType.TEXT },
  { key: "copyrightReference", label: "Copyright reference", type: FieldType.TEXT },
  { key: "locationName", label: "Location name", type: FieldType.TEXT },
  { key: "locationCode", label: "Location code", type: FieldType.TEXT },
  { key: "locationCity", label: "Location city", type: FieldType.TEXT },
];
</script>
