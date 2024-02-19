<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Batches: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>User</th>
            <th>Created</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer" @click="navigateToItem(item)">
            <td>{{ item.name }}</td>
            <td>{{ item.edges.createdBy.firstName }} {{ item.edges.createdBy.lastName }}</td>
            <td>{{ item.createdAt }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>
  </div>
</template>

<script setup lang="ts">
import { Batch } from "~/api/batch";
import { ref } from "vue";
import { Method, ListResult, API_BASE_URL } from "~/api/common";

const router = useRouter();
const store = useStore();

const ownUserId = store.getOwnUser()?.id;

const limit = ref(1000);

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  userOnly: {
    type: Boolean,
    required: false,
    default: false,
  },
});

const { data, refresh } = await useFetch<ListResult<Batch>>(`/projects/${props.projectId}/batches`, {
  method: Method.GET,
  baseURL: API_BASE_URL,
  credentials: "include",
  params: {
    limit,
    sort: "created_at",
    order: "desc",
    userId: props.userOnly ? ownUserId : "",
  },
});

function navigateToItem(item: Batch) {
  router.push(`/dashboard/projects/${props.projectId}/images?batch=${item.id}`);
}
</script>
