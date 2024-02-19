<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Please select one of your {{ data?.total }} cameras:</h3>
      <table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer" @click="selectCamera(item)">
            <td>{{ item.name }}</td>
            <td>{{ item.description }}</td>
          </tr>
        </tbody>
      </table>
    </client-only>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { ref, computed, defineEmits } from "vue";
import { Camera } from "~/api/camera";
import { requestList, getFetchOptions, Method, ListResult, API_BASE_URL } from "~/api/common";
const emit = defineEmits(["selected"]);

const store = useStore();

const user = store.getOwnUser();

const limit = ref(100);
const offset = ref(0);

// FIXME: camera picker is only able to handle a maximum of 100 cameras per user
const { data } = await useFetch(`/users/${user.id}/cameras`, {
  method: Method.GET,
  baseURL: API_BASE_URL,
  credentials: "include",
  watch: [offset],
  params: {
    limit,
    offset,
  },
});

function selectCamera(item: Camera) {
  emit("selected", item);
}
</script>
