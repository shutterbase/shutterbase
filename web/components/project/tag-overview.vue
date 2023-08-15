<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Images: {{ data?.totalImages }}</h3>
      <div class="form-control max-w-xs border-4 rounded">
        <label class="label cursor-pointer">
          <span class="label-text">Show default tags</span>
          <input v-model="showDefaultTags" type="checkbox" class="toggle" checked />
        </label>
      </div>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Image Count</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items.filter((t:Tag) => t.type !== 'default' || showDefaultTags) || []" :key="item.id" class="hover click hover:cursor-pointer">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
            <td>{{ item.edges?.tagAssignments?.length || "-" }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Method, ListResult, API_BASE_URL, getFetchOptions } from "~/api/common";
import { Tag, TagOverviewResult } from "~/api/tag";
const limit = ref(1000);
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

const showDefaultTags = ref(true);

const { data, refresh } = await useFetch<TagOverviewResult>(`/projects/${props.projectId}/tags/overview`, {
  method: Method.GET,
  baseURL: API_BASE_URL,
  credentials: "include",
  watch: [offset],
  params: {
    limit,
    offset,
  },
});
</script>
