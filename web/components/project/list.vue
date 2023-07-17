<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Projects: {{ total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id" class="hover click" @click="toDetailPage(item.id)">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Project } from "~/api/project";
import { requestList, getFetchOptions, Method } from "~/api/common";
const limit = ref(10);
const offset = ref(0);

const { items, total } = await requestList<Project>("/projects", { limit, offset });

const toDetailPage = (id: string) => {
  navigateTo(`/dashboard/projects/${id}`);
};
</script>
