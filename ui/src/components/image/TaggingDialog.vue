<template>
  <div v-show="shown" class="relative z-10" role="dialog" aria-modal="true">
    <!--
    Background backdrop, show/hide based on modal state.

    Entering: "ease-out duration-300"
      From: "opacity-0"
      To: "opacity-100"
    Leaving: "ease-in duration-200"
      From: "opacity-100"
      To: "opacity-0"
  -->
    <div v-show="shown" class="fixed inset-0 bg-gray-500 bg-opacity-25 transition-opacity"></div>

    <div v-show="shown" class="fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-20">
      <!--
      Command palette, show/hide based on modal state.

      Entering: "ease-out duration-300"
        From: "opacity-0 scale-95"
        To: "opacity-100 scale-100"
      Leaving: "ease-in duration-200"
        From: "opacity-100 scale-100"
        To: "opacity-0 scale-95"
    -->
      <div
        v-show="shown"
        class="mx-auto max-w-3xl transform divide-y divide-gray-100 overflow-hidden rounded-xl bg-white shadow-2xl ring-1 ring-black ring-opacity-5 transition-all"
      >
        <div class="relative">
          <svg class="pointer-events-none absolute left-4 top-3.5 h-5 w-5 text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path
              fill-rule="evenodd"
              d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
              clip-rule="evenodd"
            />
          </svg>
          <input
            autofocus
            ref="searchTextInput"
            v-model="searchText"
            type="text"
            class="h-12 w-full border-0 bg-transparent pl-11 pr-4 text-gray-800 dark:text-gray-200 placeholder:text-gray-400 focus:ring-0 sm:text-sm"
            placeholder="Search..."
          />
        </div>

        <div v-if="filteredTags.length !== 0" class="flex transform-gpu divide-x divide-gray-100">
          <!-- Preview Visible: "sm:h-96" -->
          <div class="max-h-96 min-w-0 flex-auto scroll-py-4 overflow-y-auto px-6 py-4 sm:h-96">
            <!-- Default state, show/hide based on command palette state. -->
            <!-- <h2 class="mb-4 mt-2 text-xs font-semibold text-gray-500">Recent tags</h2>
            <ul class="-mx-2 text-sm text-gray-700" id="recent" role="listbox">

              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="recent-1" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1463453091185-61582044d556?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Floyd Miles</span>

                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
            </ul> -->

            <!-- Results, show/hide based on command palette state. -->
            <ul class="-mx-2 text-sm text-gray-700" id="options" role="listbox">
              <!-- Active: "bg-gray-100 text-gray-900" -->
              <li v-for="tag in filteredTags" class="group flex cursor-default select-none items-center rounded-md p-2" role="option" tabindex="-1">
                <TagIcon class="h-6 w-6 flex-none text-gray-400" />
                <div class="ml-4 flex-auto truncate">
                  <!-- Active: "text-gray-900", Not Active: "text-gray-700" -->
                  <p class="text-sm font-medium text-gray-700 truncate">{{ tag.name }}</p>
                  <!-- Active: "text-gray-700", Not Active: "text-gray-500" -->
                  <p class="text-sm text-gray-500 truncate">{{ tag.description }}</p>
                </div>
                <!-- Not Active: "hidden" -->
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
            </ul>
          </div>

          <!-- Active item side-panel, show/hide based on active state -->
          <!-- <div class="h-96 w-1/2 flex-none flex-col divide-y divide-gray-100 overflow-y-auto sm:flex">
            <div class="flex-none p-6 text-center">
              <img
                src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                alt=""
                class="mx-auto h-16 w-16 rounded-full"
              />
              <h2 class="mt-3 font-semibold text-gray-900">Tom Cook</h2>
              <p class="text-sm leading-6 text-gray-500">Director, Product Development</p>
            </div>
            <div class="flex flex-auto flex-col justify-between p-6">
              <dl class="grid grid-cols-1 gap-x-6 gap-y-3 text-sm text-gray-700">
                <dt class="col-end-1 font-semibold text-gray-900">Phone</dt>
                <dd>881-460-8515</dd>
                <dt class="col-end-1 font-semibold text-gray-900">URL</dt>
                <dd class="truncate"><a href="https://example.com" class="text-indigo-600 underline">https://example.com</a></dd>
                <dt class="col-end-1 font-semibold text-gray-900">Email</dt>
                <dd class="truncate"><a href="#" class="text-indigo-600 underline">tomcook@example.com</a></dd>
              </dl>
              <button
                type="button"
                class="mt-6 w-full rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
              >
                Send message
              </button>
            </div>
          </div>  -->
        </div>

        <!-- Empty state, show/hide based on command palette state -->
        <div v-if="filteredTags.length === 0 && searchText !== ''" class="px-6 py-14 text-center text-sm sm:px-14">
          <TagIcon class="mx-auto h-6 w-6 text-gray-500" />
          <p class="mt-4 font-semibold text-gray-900 dark:text-gray-100">No matching tags</p>
          <p class="mt-2 text-gray-500">No tag matching your search could be found. Please use a different keyword or create a 'custom' tag</p>
          <p class="mt-4 font-semibold text-gray-900 dark:text-gray-100 underline cursor-pointer">Create custom tag</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "src/stores/user-store";
import { storeToRefs } from "pinia";
import { ImageWithTagsType } from "src/types/custom";
import { TagIcon } from "@heroicons/vue/24/outline";
import { Ref, computed, nextTick, onMounted, ref } from "vue";
import { emitter } from "src/boot/mitt";
import { debug } from "src/util/logger";
import { onHotkey } from "src/util/keyEvents";
import { Image } from "src/util/fileProcessor";
import { ImageTagsResponse } from "src/types/pocketbase";

interface Props {
  image: ImageWithTagsType | null;
  shown: boolean;
}
const props = withDefaults(defineProps<Props>(), {});
const emit = defineEmits<{
  selected: [ImageWithTagsType, ImageTagsResponse];
}>();

const userStore = useUserStore();
const { projectTags } = storeToRefs(userStore);

const searchText = ref("");
const searchTextInput: Ref<HTMLInputElement | null> = ref(null);

const filteredTags = computed(() => {
  if (!projectTags.value) {
    return [];
  }
  if (searchText.value === "") {
    return [];
  }
  return projectTags.value.filter((tag) => {
    if (tag.type === "default" || tag.type === "template") {
      return false;
    }
    if (tag.name.toLowerCase().includes(searchText.value.toLowerCase())) {
      return true;
    }
    if (tag.description.toLowerCase().includes(searchText.value.toLowerCase())) {
      return true;
    }
    return false;
  });
});

function focusSearchText() {
  debug("focusing search text");
  nextTick(() => {
    searchTextInput.value?.focus();
  });
}

function clearSearchText() {
  searchText.value = "";
}

onHotkey({ key: "Enter", modifierKeys: [] }, acceptOnlyResult);
function acceptOnlyResult() {
  if (!props.image) {
    return;
  }
  if (filteredTags.value.length === 1) {
    emit("selected", props.image, filteredTags.value[0]);
  }
}

defineExpose({
  focusSearchText,
  clearSearchText,
});
</script>
