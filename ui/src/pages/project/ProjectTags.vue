<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
      <div class="pb-2">
        <h2 class="text-2xl font-semibold leading-7 text-primary-900 dark:text-primary-200">Project Tags</h2>
      </div>
    </div>

    <div>
      <div style="display: flex; flex-direction: column; margin-top: 24px;">
        <q-form
          @submit="uploadTag"
          style="width: '100%'; display: flex; justify-content: space-between;"
        >
          <q-input outlined v-model="tagName" label="Name*" width="150px" height="40px" :disabled="isTagUploading" required />
          <q-input outlined v-model="tagDescription" label="Description*" width="150px" height="40px" :disabled="isTagUploading" required />
          <q-toggle v-model="isTagAlbum" label="Is Album" />
          <q-select outlined v-model="tagType" :options="tagTypeOptions" label="Type" />
          <q-btn
            type="submit"
            style="background: green;
            color: white; margin-left: 24px"
            label="Add"
            :loading="isTagUploading"
          />
        </q-form>


        <div style="margin-top: 24px">
          <q-table
            :rows="tagList || []"
            ref="tableRef"
            :columns="columns"
            row-key="id"
            no-data-label="No tags found. Add your first"
            :loading="isTableActionPending"
            v-model:pagination="pagination"
            @request="loadTags"
          >
            <template v-slot:body="props">
              <q-tr :props="props">
                <q-td key="name" :props="props">
                  <span v-if="!isEditing">{{ props.row.name }}</span>
                  <q-input
                    v-if="isEditing"
                    v-model.text="props.row.name"
                    type="text"
                    dense
                    borderless
                  />
                </q-td>

                <q-td key="description" :props="props">
                  <span v-if="!isEditing">{{ props.row.description }}</span>
                  <q-input
                    v-if="isEditing"
                    v-model.text="props.row.description"
                    type="text"
                    dense
                    borderless
                  />
                </q-td>

                <q-td key="isAlbum" :props="props">
                  <span v-if="!isEditing">{{ props.row.isAlbum ? 'Yes' : 'No' }}</span>
                  <q-toggle
                    v-if="isEditing"
                    v-model="props.row.isAlbum"
                  />
                </q-td>

                <q-td key="type" :props="props">
                  <span v-if="!isEditing">{{ props.row.type }}</span>
                  <q-select v-if="isEditing" outlined v-model="props.row.type" :options="tagTypeOptions" label="Type" dense borderless/>
                </q-td>

                <q-td key="actions" :props="props">
                  <q-btn v-if="!isEditing" icon="mode_edit" @click="toggleEditing"></q-btn>
                  <q-btn v-if="isEditing" icon="check_box"  color="green" @click="updateTag(props.row)"></q-btn>
                  <q-btn v-if="isEditing" icon="close" @click="toggleEditing" style="margin-left: 8px"></q-btn>
                  <q-btn icon="delete" color="red" @click="showDeletePopup(props.row)" style="margin-left: 8px"></q-btn>
                </q-td>
              </q-tr>
            </template>
          </q-table>
        </div>
      </div>
    </div>
  </main>

  <q-dialog v-model="isConfirmDeleteVisible" persistent>
      <q-card>
        <q-card-section class="row items-center">
          <q-avatar icon="warning" color="warning" text-color="black" />
          <span class="q-ml-sm">Are you sure you want to delete <b>{{ tagToDelete.name }}</b> tag?</span>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Cancel" color="primary" v-close-popup />
          <q-btn flat label="Delete" color="negative" @click="deleteTag" :loading="isDeleteInProgress" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { useQuasar } from 'quasar'
import { ProjectsResponse, ImageTagsResponse, ImageTagsRecord, ImageTagsTypeOptions } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
const $q = useQuasar()
const route = useRoute();

const project: Ref<ProjectsResponse | null> = ref(null);
const projectId: string = `${route.params.id}`;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const tagList: Ref<ImageTagsRecord[] | null> = ref(null);
let isTagUploading = ref(false);
let isEditing = ref(false);
let isConfirmDeleteVisible = ref(false);
let isDeleteInProgress = ref(false);
let tagToDelete: Ref<{ id: string; name: string }> = ref({ id: '', name: ''});

let tagName: Ref<ImageTagsRecord["name"]> = ref("");
let tagDescription: Ref<ImageTagsRecord["description"]> = ref("");
let isTagAlbum: Ref<ImageTagsRecord["isAlbum"]> = ref(false);
let tagType: Ref<keyof typeof ImageTagsTypeOptions> = ref('default');
const tagTypeOptions: [keyof typeof ImageTagsTypeOptions, keyof typeof ImageTagsTypeOptions, keyof typeof ImageTagsTypeOptions] = ['default', 'manual', 'custom']

// const tableRef = ref();
const columns = [
  { name: 'name', label: 'Name', field: 'name', required: true, sortable: true, align: 'left' },
  { name: 'description', label: 'Description', field: 'description', sortable: true, align: 'left' },
  { name: 'isAlbum', label: 'Is Album', field: 'isAlbum', format: (val: boolean) => val ? 'Yes' : 'No', sortable: true, align: 'left' },
  { name: 'type', label: 'Type', field: 'type', align: 'left' },
  { name: 'actions' }
]
const pagination = ref({
  page: 1,
  rowsPerPage: 5,
  rowsNumber: 10
})
let isTableActionPending = ref(false);

async function loadProject() {
  if (!projectId || projectId === "") {
    console.error("No project ID provided");
    return;
  }

  try {
    console.log(`Loading project ${projectId}`);

    const response = await pb.collection<ProjectsResponse>("projects").getOne(projectId);
    project.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function loadTags(props?: any) {
  try {
    isTableActionPending.value = true;

    console.log('pagination.value.page',pagination.value.page)

    const response = await pb.collection<ImageTagsResponse>("image_tags").getList(
      props ? props.pagination.page : pagination.value.page,
      pagination.value.rowsPerPage,
      {
        sort: '-created',
      }
    );

    console.log('Image tags response:', response);

    pagination.value.rowsNumber = response.totalItems;
    pagination.value.page = response.page;
    tagList.value = response.items;
    isTableActionPending.value = false;
  } catch (error: any) {
    isTableActionPending.value = false;
    unexpectedError.value = error ? error : "unknown error while loading tags";
    showUnexpectedErrorMessage.value = true;
  }
}

async function uploadTag() {
  const data: ImageTagsRecord = {
    name: tagName.value,
    description: tagDescription.value,
    isAlbum: isTagAlbum.value,
    type: tagType.value,
    project: projectId
  };

  try {
    isTagUploading.value = true;

    await pb.collection('image_tags').create(data);
    $q.notify(`Tag ${tagName.value.toUpperCase()} has been successfully added`);
    loadData();
  } catch (error: any) {
    unexpectedError.value = error ? error : "unknown error while uploading a tag";
    showUnexpectedErrorMessage.value = true;
  } finally {
    isTagUploading.value = false;
  }
}

async function toggleEditing() {
  isEditing.value = !isEditing.value;
}

async function updateTag(props: ImageTagsResponse) {
  const data: ImageTagsRecord = {
    name: props.name,
    description: props.description,
    isAlbum: props.isAlbum,
    type: props.type,
    project: projectId
  };

  try {
    isTableActionPending.value = true;
    await pb.collection('image_tags').update(props.id, data);
    $q.notify(`Tag ${props.name.toUpperCase()} has been successfully updated`);
    loadData();
  } catch (error: any) {
    unexpectedError.value = error ? error : "unknown error while uploading a tag";
    showUnexpectedErrorMessage.value = true;
  } finally {
    isEditing.value = false;
    isTagUploading.value = false;
  }
}

async function showDeletePopup(props: ImageTagsResponse) {
  isConfirmDeleteVisible.value = true;
  tagToDelete.value = { id: props.id, name: props.name };
}

async function deleteTag() {
  isDeleteInProgress.value = true;

  try {
    await pb.collection('image_tags').delete(tagToDelete.value.id);
    $q.notify(`Tag ${tagToDelete.value.name.toUpperCase()} has been successfully deleted`);
    tagToDelete.value = { id: '', name: '' };
    loadData();
  } catch (error: any) {
    unexpectedError.value = error ? error : "unknown error while uploading a tag";
    showUnexpectedErrorMessage.value = true;
  } finally {
    isConfirmDeleteVisible.value = false;
    isDeleteInProgress.value = false;
  }
}

async function loadData() {
  loadProject();
  loadTags();
}

watch(route, loadData);
onMounted(loadData);
</script>
