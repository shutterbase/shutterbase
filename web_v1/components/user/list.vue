<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Users: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>First name</th>
            <th>Last name</th>
            <th>Email</th>
            <th>Role</th>
            <th>Active</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in data?.items || []" :key="user.id" class="hover click hover:cursor-pointer" @click="toDetailPage(user.id)">
            <td>{{ user.firstName }}</td>
            <td>{{ user.lastName }}</td>
            <td>{{ user.email }}</td>
            <td>{{ user.edges.role?.description }}</td>
            <td>{{ user.active }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>
    <div class="join">
      <button class="join-item btn" :disabled="isFirstPage" @click="page--">«</button>
      <button class="join-item btn">Page {{ page }}</button>
      <button class="join-item btn" :disabled="isLastPage" @click="page++">»</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { User } from "api/user";
import { requestList, getFetchOptions, Method, ListResult, API_BASE_URL } from "~/api/common";
const limit = ref(5);
const page = ref(1);
const offset = computed(() => {
  return (page.value - 1) * limit.value;
});

const { data } = await useFetch<ListResult<User>>("/users", {
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
  navigateTo(`/dashboard/users/${id}`);
};
</script>
