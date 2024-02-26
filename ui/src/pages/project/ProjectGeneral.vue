<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
      <DetailEditGroup @edit-save="saveProject" headline="Project Information" subtitle="General information concerning this project" :fields="informationFields" :item="project" />
      <DetailEditGroup
        @edit-save="saveProject"
        headline="Copyright Data"
        subtitle="Copyright information to be embedded into the EXIF data"
        :fields="copyrightFields"
        :item="project"
      />
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
const route = useRoute();

const project: Ref<ProjectsResponse | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadProject() {
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

async function saveProject(editData: EditData<ProjectsResponse>) {
  if (!project.value) {
    console.log("No project to save");
    return;
  }

  const rollbackData = { ...project.value };
  project.value = { ...project.value, ...editData };

  try {
    console.log(`Saving project ${project.value.id}`);
    const response = await pb.collection<ProjectsResponse>("projects").update(project.value.id, editData);
    project.value = response;
  } catch (error: any) {
    project.value = rollbackData;
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

watch(route, loadProject);
onMounted(loadProject);
</script>
