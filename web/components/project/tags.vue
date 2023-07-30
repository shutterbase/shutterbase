<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Tags: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Album</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
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
    <div class="mr-8"><button class="btn btn-secondary" @click="openAddDialog">Add tags</button></div>
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

const { data } = await useFetch<ListResult<Tag>>(`/projects/${props.projectId}/tags`, {
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
  // TODO remove from list locally
}

function openAddDialog() {}
</script>
