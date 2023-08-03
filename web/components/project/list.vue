<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Projects: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
            <td>
              <button class="btn btn-xs btn-secondary mr-2" @click="makeActiveProject(item.id)">Make active project</button>
              <button class="btn btn-xs btn-secondary" @click="toDetailPage(item.id)">Edit</button>
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
    <div class="mr-8"><button class="btn btn-secondary" @click="toCreatePage">Create Project</button></div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Method, ListResult, API_BASE_URL } from "~/api/common";
import { Project } from "~/api/project";

const store = useStore();

const limit = ref(5);
const page = ref(1);
const offset = computed(() => {
  return (page.value - 1) * limit.value;
});

const { data } = await useFetch<ListResult<Project>>("/projects", {
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

const toDetailPage = (id: string) => {
  navigateTo(`/dashboard/projects/${id}`);
};

const toCreatePage = () => {
  navigateTo(`/dashboard/projects/create`);
};

function makeActiveProject(id: string) {
  store.activeProjectId = id;
}
</script>
