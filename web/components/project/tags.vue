<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Tags: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Type</th>
            <th>Album</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
            <td>{{ item.type }}</td>
            <td>{{ item.isAlbum }}</td>
            <td>
              <button class="btn btn-sm btn-error" @click="deleteTag(item.id)">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </client-only>
    <div class="join">
      <button class="join-item btn" :disabled="isFirstPage" @click="page--">«</button>
      <button class="join-item btn">Page {{ page }}</button>
      <button class="join-item btn" :disabled="isLastPage" @click="page++">»</button>
    </div>
    <div class="mr-8"><button class="btn btn-secondary" onclick="addTagDialog.showModal()">Add single tag</button></div>
    <div class="mr-8"><button class="btn btn-secondary" onclick="addTagsDialog.showModal()">Add CSV</button></div>
    <dialog id="addTagDialog" class="modal">
      <form method="dialog" class="modal-box">
        <h3 class="font-bold text-lg">Add a single tag</h3>
        <input v-model="tagName" type="text" placeholder="Name" class="input input-bordered w-full max-w-xs m-2" />
        <input v-model="tagDescription" type="text" placeholder="Description" class="input input-bordered w-full max-w-xs m-2" />
        <select v-model="tagType" class="select select-bordered w-full max-w-xs m-2">
          <option v-for="option in tagTypeOptions">{{ option }}</option>
        </select>
        <div class="form-control m-2">
          <label class="label cursor-pointer">
            <span class="label-text">Is Album</span>
            <input type="checkbox" :checked="tagIsAlbum" class="checkbox" />
          </label>
        </div>
        <div class="modal-action">
          <button class="btn" onclick="addTagDialog.close()">Cancel</button>
          <button class="btn" @click="addTag">Add</button>
        </div>
      </form>
    </dialog>
    <dialog id="addTagsDialog" class="modal">
      <form method="dialog" class="modal-box w-11/12 max-w-5xl">
        <h3 class="font-bold text-lg">Add multiple tags as CSV</h3>
        <textarea v-model="tagsCsv" placeholder="<name>,<description>,<isAlbum>" class="textarea w-full font-mono"></textarea>
        <div class="modal-action">
          <button class="btn" onclick="addTagsDialog.close()">Cancel</button>
          <button class="btn" @click="addTags">Add</button>
        </div>
      </form>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Method, ListResult, API_BASE_URL, getFetchOptions } from "~/api/common";
import { Tag } from "~/api/tag";
const limit = ref(100);
const page = ref(1);
const offset = computed(() => {
  return (page.value - 1) * limit.value;
});

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

const { data, refresh } = await useFetch<ListResult<Tag>>(`/projects/${props.projectId}/tags`, {
  method: Method.GET,
  baseURL: API_BASE_URL,
  credentials: "include",
  watch: [offset],
  params: {
    limit,
    offset,
  },
});

const isFirstPage = computed(() => page.value === 1);
const isLastPage = computed(() => page.value === Math.ceil((data.value?.total || 0) / limit.value));

async function deleteTag(id: string) {
  const { data } = await useFetch(`/projects/${props.projectId}/tags/${id}`, getFetchOptions(Method.DELETE, {}));
  await refresh();
}

const addTagDialog = ref<HTMLDialogElement | null>(null);
const tagName = ref("");
const tagDescription = ref("");
const tagIsAlbum = ref(false);
const tagType = ref("manual");
const tagTypeOptions = ref(["manual", "default"]);
const tagsCsv = ref("");

async function addTag() {
  const { data } = await useFetch(
    `/projects/${props.projectId}/tags`,
    getFetchOptions(Method.POST, { name: tagName.value, description: tagDescription.value, type: tagType.value, isAlbum: tagIsAlbum.value })
  );
  await refresh();
  tagName.value = "";
  tagDescription.value = "";
  tagType.value = "manual";
  tagIsAlbum.value = false;
  addTagDialog.value?.close();
}

async function addTags() {
  const tags = [] as { name: string; description: string; isAlbum: boolean }[];
  for (const line of tagsCsv.value.split("\n")) {
    const [name, description, isAlbum] = line.split(",");
    if (!name || !description) continue;
    tags.push({ name, description, isAlbum: isAlbum === "true" });
  }
  const { data } = await useFetch(`/projects/${props.projectId}/tags/bulk`, getFetchOptions(Method.POST, { tags }));
  await refresh();
  tagsCsv.value = "";
}
</script>
