<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Cameras: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer" @click="toDetailPage(item.id)">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>

    <div class="join">
      <button class="join-item btn" :disabled="isFirstPage" @click="page--">«</button>
      <button class="join-item btn">Page {{ page }}</button>
      <button class="join-item btn" :disabled="isLastPage" @click="page++">»</button>
    </div>
    <div class="mr-8"><button class="btn btn-secondary" @click="toCreatePage">Create Camera</button></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { Camera } from "~/api/camera";
import { requestList, getFetchOptions, Method, ListResult, API_BASE_URL } from "~/api/common";

const props = defineProps({
  userId: {
    type: String,
    required: true,
  },
});

const limit = ref(5);
const page = ref(1);
const offset = computed(() => {
  return (page.value - 1) * limit.value;
});

const { data } = await useFetch(`/users/${props.userId}/cameras`, {
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
  navigateTo(`/dashboard/users/${props.userId}/cameras/${id}`);
};

const toCreatePage = () => {
  navigateTo(`/dashboard/users/${props.userId}/cameras/create`);
};
</script>
