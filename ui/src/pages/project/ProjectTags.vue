<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table dense :items="item?.expand.image_tags_via_project" :columns="imageTagColumns" name="Project Tag" :add-callback="startTagCreate"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <TagDialog :show="showTagDialog" :create="createTag" :tag="editTagData" @add="addTag" @edit="editTag" @bulk="switchToBulkDialog" @closed="() => (showTagDialog = false)" />
    <BulkTagCreationDialog :show="showBulkTagDialog" @add="addBulkTags" @closed="() => (showBulkTagDialog = false)" />
  </div>
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import { ImageTagsResponse, ProjectsResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { ProjectWithTagsType } from "src/types/custom";
import TagDialog from "src/components/project/TagDialog.vue";
import BulkTagCreationDialog from "src/components/project/BulkTagCreationDialog.vue";
const route = useRoute();

type ITEM_TYPE = ProjectWithTagsType;
const ITEM_COLLECTION = "projects";
const ITEM_NAME = "project";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const showTagDialog = ref(false);
const createTag = ref(false);

const showBulkTagDialog = ref(false);
function switchToBulkDialog() {
  showTagDialog.value = false;
  showBulkTagDialog.value = true;
}

const editTagData: Ref<ImageTagsResponse> = ref({} as ImageTagsResponse);

async function loadItem() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).getOne(itemId, {
      expand: "image_tags_via_project",
    });
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function addTag(input: ImageTagsResponse) {
  if (item.value === null) {
    console.log(`No ${ITEM_NAME} loaded`);
    return;
  }

  try {
    console.log(`Adding tag ${input.name} to project ${item.value?.name}`);
    input.project = item.value.id;

    const response = await pb.collection<ImageTagsResponse>("image_tags").create(input);
    showTagDialog.value = false;
    const itemId = response.id;
    console.log(`Tag with ID ${itemId} created`);
    showNotificationToast({ headline: `Tag created`, type: "success" });
    item.value.expand.image_tags_via_project.push(response);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function addBulkTags(input: ImageTagsResponse[]) {
  if (item.value === null) {
    console.log(`No ${ITEM_NAME} loaded`);
    return;
  }

  console.log(`Adding ${input.length} tags to project ${item.value.name}`);

  for (const tag of input) {
    tag.project = item.value.id;
    try {
      const response = await pb.collection<ImageTagsResponse>("image_tags").create(tag);
      item.value.expand.image_tags_via_project.push(response);
      console.log(`Tag with ID ${response.id} created`);
    } catch (error: any) {
      console.log(`Error creating tag ${tag.name}`);
      showNotificationToast({ headline: `Error creatin tag ${tag.name}`, type: "error", timeout: 10000 });
      unexpectedError.value = error;
      showUnexpectedErrorMessage.value = true;
      break;
    }
  }

  showBulkTagDialog.value = false;
  console.log(`${input.length} tags created`);
  showNotificationToast({ headline: `${input.length} tags created`, type: "success" });
}

function startTagCreate() {
  showTagDialog.value = true;
  createTag.value = true;
  editTagData.value = {} as ImageTagsResponse;
}

function startTagEdit(tag: ImageTagsResponse) {
  showTagDialog.value = true;
  createTag.value = false;
  editTagData.value = tag;
}

async function editTag(input: ImageTagsResponse) {
  if (item.value === null) {
    console.log(`No ${ITEM_NAME} loaded`);
    return;
  }

  try {
    console.log(`Editing tag ${input.name} in project ${item.value.name}`);
    const response = await pb.collection<ImageTagsResponse>("image_tags").update(input.id, input);
    showTagDialog.value = false;
    console.log(`Tag with ID ${input.id} updated`);
    showNotificationToast({ headline: `Tag updated`, type: "success" });
    const index = item.value.expand.image_tags_via_project.findIndex((t) => t.id === input.id);
    item.value.expand.image_tags_via_project[index] = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function deleteTag(tag: ImageTagsResponse) {
  if (item.value === null) {
    console.log(`No ${ITEM_NAME} loaded`);
    return;
  }

  try {
    console.log(`Deleting tag ${tag.name} from project ${item.value.name}`);
    const response = pb.collection<ImageTagsResponse>("image_tags").delete(tag.id);
    console.log(`Tag with ID ${tag.id} deleted`);
    showNotificationToast({ headline: `Tag deleted`, type: "success" });
    item.value.expand.image_tags_via_project = item.value.expand.image_tags_via_project.filter((t) => t.id !== tag.id);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const imageTagColumns: TableColumn<ImageTagsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "description", label: "Description" },
  { key: "type", label: "Type" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Edit", callback: startTagEdit, type: TableRowActionType.EDIT },

      {
        key: "delete",
        label: "Delete",
        callback: deleteTag,
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

watch(route, loadItem);
onMounted(loadItem);
</script>
