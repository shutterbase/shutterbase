<template>
  <div class="lg:flex lg:items-center lg:justify-between">
    <div class="min-w-0 flex-1">
      <h2 class="text-2xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight">Images: {{ activeProject.name }}</h2>
      <div class="mt-1 flex flex-col sm:mt-0 sm:flex-row sm:flex-wrap sm:space-x-6">
        <div class="mt-2 flex items-center text-sm text-gray-500">
          <PhotoIcon class="mr-1.5 h-5 w-5 flex-shrink-0 text-gray-400 dark:text-gray-600" />
          {{ totalImageCount }} images
        </div>
      </div>
    </div>
    <div class="lg:ml-4 flex">
      <ProjectTagComboBox class="mr-5" @selected="emitFilterTags" />
      <div class="sm:col-span-3 mr-5">
        <label for="search" class="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-100">Search</label>
        <div class="mt-2">
          <input
            id="search"
            v-model="searchText"
            placeholder="Search text"
            type="text"
            :class="[
              `block w-full rounded-md border-0 py-1.5 focus:ring-2 focus:ring-inset shadow-sm ring-1 ring-inset sm:text-sm sm:leading-6`,
              `text-gray-900 placeholder:text-gray-400 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-700`,
            ]"
          />
        </div>
      </div>
      <div class="sm:col-span-3">
        <label for="sorting" class="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-100">Sorting</label>
        <div class="mt-2">
          <select
            id="sorting"
            name="sorting"
            v-model="preferredImageSortOrder"
            class="block w-full rounded-md px-3.5 py-1.5 text-gray-900 bg-gray-50 border border-gray-300 focus:ring-primary-500 focus:border-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-primary-500 dark:focus:border-primary-500 sm:text-sm sm:leading-6"
          >
            <option value="latestFirst">Latest images first</option>
            <option value="oldestFirst">Oldest images first</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { PhotoIcon } from "@heroicons/vue/24/outline";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { onMounted, ref, watch } from "vue";
import ProjectTagComboBox from "../ProjectTagComboBox.vue";
import { ImageTagsResponse } from "src/types/pocketbase";

// const emit = defineEmits<{
//   sort: [SORT_ORDER];
// }>();

interface Props {
  totalImageCount: number;
}
const props = withDefaults(defineProps<Props>(), {
  totalImageCount: 0,
});

const emit = defineEmits<{
  search: [string];
  filterTags: [string[]];
}>();

const { activeProject, preferredImageSortOrder } = storeToRefs(useUserStore());

const searchText = ref("");
watch(searchText, () => emit("search", searchText.value));

function emitFilterTags(tags: ImageTagsResponse[]) {
  emit(
    "filterTags",
    tags.map((tag) => tag.id)
  );
}
</script>
<script lang="ts">
export enum SORT_ORDER {
  LATEST_FIRST = "latestFirst",
  OLDEST_FIRST = "oldestFirst",
}
</script>
