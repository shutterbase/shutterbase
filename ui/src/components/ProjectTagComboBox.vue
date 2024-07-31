<template>
  <div>
    <label for="combobox" class="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-100">Tags</label>
    <div class="relative mt-2">
      <input
        v-model="searchText"
        :placeholder="computedPlaceholder"
        id="combobox"
        type="text"
        class="w-full rounded-md border-0 py-1.5 pl-3 pr-12 shadow-sm ring-1 ring-inset focus:ring-2 focus:ring-inset sm:text-sm sm:leading-6 text-gray-900 placeholder:text-gray-400 focus:ring-primary-600 ring-gray-300 dark:ring-primary-600 focus:dark:ring-gray-400 dark:text-gray-100 dark:bg-primary-700"
        role="combobox"
        aria-controls="options"
        aria-expanded="false"
      />
      <button @click="toggleCombobox" type="button" class="absolute inset-y-0 right-0 flex items-center rounded-r-md px-2 focus:outline-none">
        <svg class="h-5 w-5 text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path
            fill-rule="evenodd"
            d="M10 3a.75.75 0 01.55.24l3.25 3.5a.75.75 0 11-1.1 1.02L10 4.852 7.3 7.76a.75.75 0 01-1.1-1.02l3.25-3.5A.75.75 0 0110 3zm-3.76 9.2a.75.75 0 011.06.04l2.7 2.908 2.7-2.908a.75.75 0 111.1 1.02l-3.25 3.5a.75.75 0 01-1.1 0l-3.25-3.5a.75.75 0 01.04-1.06z"
            clip-rule="evenodd"
          />
        </svg>
      </button>

      <ul
        v-if="comboboxVisible"
        class="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-gray-100 dark:bg-gray-700 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm"
        id="options"
        role="listbox"
      >
        <!--
        Combobox option, manage highlight styles based on mouseenter/mouseleave and keyboard navigation.

        Active: "text-white bg-indigo-600", Not Active: "text-gray-900"
      -->
        <li
          v-for="tag in filteredProjectTags"
          @click="toggleTagSelection(tag)"
          :key="tag.id"
          class="relative cursor-default select-none py-2 pl-8 pr-4 text-gray-900 dark:text-gray-100 bg-gray-100 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800"
          id="option-0"
          role="option"
          tabindex="-1"
        >
          <!-- Selected: "font-semibold" -->
          <span class="block truncate">{{ tag.name }}</span>

          <!--
          Checkmark, only display for selected option.

          Active: "text-white", Not Active: "text-indigo-600"
        -->
          <span v-if="selectedTags.some((t) => t.id === tag.id)" class="absolute inset-y-0 left-0 flex items-center pl-1.5 text-primary-600 dark:text-primary-400">
            <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path
                fill-rule="evenodd"
                d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
                clip-rule="evenodd"
              />
            </svg>
          </span>
        </li>

        <!-- More items... -->
      </ul>
    </div>
  </div>
</template>
<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { ImageTagsResponse, ImageTagsTypeOptions } from "src/types/pocketbase";
import { computed, ref, watch } from "vue";

const emit = defineEmits<{
  selected: [ImageTagsResponse[]];
}>();

const userStore = useUserStore();
const { projectTags } = storeToRefs(userStore);

const searchText = ref("");
const comboboxVisible = ref(false);

watch(searchText, (newValue: string, oldValue: string) => {
  if (oldValue === "" && newValue !== "") {
    comboboxVisible.value = true;
  }
});

const filteredProjectTags = computed(() => {
  const filteredTags = projectTags.value.filter((tag) => {
    if (tag.type === ImageTagsTypeOptions.template) {
      return false;
    }

    if (selectedTags.value.some((t) => t.id === tag.id)) {
      return false;
    }

    return tag.name.toLowerCase().includes(searchText.value.toLowerCase());
  });

  return [...selectedTags.value, ...filteredTags];
});

const computedPlaceholder = computed(() => {
  if (selectedTags.value.length === 0) {
    return "Select tags";
  }

  return selectedTags.value.map((tag) => tag.name).join(", ");
});

function toggleTagSelection(tag: ImageTagsResponse) {
  // selectedTags.value = [];
  if (selectedTags.value.some((t) => t.id === tag.id)) {
    selectedTags.value = selectedTags.value.filter((t) => t.id !== tag.id);
  } else {
    selectedTags.value = [...selectedTags.value, tag];
  }
}

function toggleCombobox() {
  comboboxVisible.value = !comboboxVisible.value;
}

const selectedTags = ref([] as ImageTagsResponse[]);
watch(selectedTags, () => emit("selected", selectedTags.value));
</script>
