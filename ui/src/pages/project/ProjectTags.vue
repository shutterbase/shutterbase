<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table
      dense
      :items="item?.tags || []"
      :columns="imageTagColumns"
      name="Project Tag"
      :allow-add="userStore.isProjectAdminOrHigher()"
      :add-callback="startTagCreate"
    ></Table>
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
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { ProjectWithTagsType } from "src/types/custom";
import TagDialog from "src/components/project/TagDialog.vue";
import BulkTagCreationDialog from "src/components/project/BulkTagCreationDialog.vue";
import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
const route = useRoute();

const userStore = useUserStore();
const { projectTags } = storeToRefs(userStore);

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
    const [project, tags] = await Promise.all([api.projects.get(itemId), api.imageTags.list({ projectId: itemId, limit: 500 })]);
    item.value = { ...project, tags: tags.items };
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

    const response = await api.imageTags.create({
      name: input.name,
      description: input.description,
      isAlbum: input.isAlbum,
      type: input.type,
      projectId: item.value.id,
    });
    showTagDialog.value = false;
    const itemId = response.id;
    console.log(`Tag with ID ${itemId} created`);
    showNotificationToast({ headline: `Tag created`, type: "success" });
    item.value.tags = [...(item.value.tags || []), response];
    projectTags.value?.push(response);
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
    try {
      const response = await api.imageTags.create({
        name: tag.name,
        description: tag.description,
        isAlbum: tag.isAlbum,
        type: tag.type,
        projectId: item.value.id,
      });
      item.value.tags = [...(item.value.tags || []), response];
      projectTags.value?.push(response);
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
    const response = await api.imageTags.update(input.id, {
      name: input.name,
      description: input.description,
      isAlbum: input.isAlbum,
      type: input.type,
    });
    showTagDialog.value = false;
    console.log(`Tag with ID ${input.id} updated`);
    showNotificationToast({ headline: `Tag updated`, type: "success" });

    const tags = item.value.tags || [];
    const index = tags.findIndex((t) => t.id === input.id);
    if (index !== -1) tags[index] = response;

    const projectTagIndex = projectTags.value?.findIndex((t) => t.id === input.id);
    if (projectTagIndex !== undefined && projectTagIndex !== -1) {
      projectTags.value[projectTagIndex] = response;
    }
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function deleteTag(tag: ImageTagsResponse) {
  if (item.value === null) {
    console.log(`No ${ITEM_NAME} loaded`);
    return;
  }

  try {
    console.log(`Deleting tag ${tag.name} from project ${item.value.name}`);
    await api.imageTags.remove(tag.id);
    console.log(`Tag with ID ${tag.id} deleted`);
    showNotificationToast({ headline: `Tag deleted`, type: "success" });
    item.value.tags = (item.value.tags || []).filter((t) => t.id !== tag.id);
    projectTags.value = projectTags.value?.filter((t) => t.id !== tag.id);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function showTagEdit() {
  return userStore.isProjectAdminOrHigher();
}

const imageTagColumns: TableColumn<ImageTagsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "description", label: "Description" },
  { key: "type", label: "Type" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Edit", showCallback: showTagEdit, callback: startTagEdit, type: TableRowActionType.EDIT },

      {
        key: "delete",
        label: "Delete",
        showCallback: showTagEdit,
        callback: deleteTag,
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

watch(route, loadItem);
onMounted(loadItem);
</script>
