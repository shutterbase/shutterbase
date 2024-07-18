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
        class="mx-auto max-w-3xl transform divide-y divide-gray-100 dark:divide-gray-800 overflow-hidden rounded-xl bg-gray-50 dark:bg-gray-800 shadow-2xl ring-1 ring-black ring-opacity-5 transition-all"
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
            placeholder="Search tag..."
          />
        </div>

        <div v-if="(filteredTags.length !== 0 && searchText !== '') || (recentTags.length !== 0 && searchText === '')" class="flex transform-gpu divide-x divide-gray-100">
          <!-- Preview Visible: "sm:h-96" -->
          <div class="max-h-96 min-w-0 flex-auto scroll-py-4 overflow-y-auto px-6 py-4 sm:h-96">
            <!-- Default state, show/hide based on command palette state. -->
            <h2 v-if="filteredTags.length === 0 && searchText === ''" class="mb-4 mt-2 text-xs font-semibold text-gray-500">Recent tags</h2>
            <ul v-if="filteredTags.length === 0 && searchText === ''" class="-mx-2 text-sm text-gray-700" id="options" role="listbox">
              <!-- Active: "bg-gray-100 text-gray-900" -->
              <li
                @click="() => acceptTag(tag)"
                v-for="(tag, index) in recentTags"
                class="cursor-pointer group flex select-none items-center rounded-md p-2 hover:bg-gray-200 hover:text-gray-100 dark:hover:bg-gray-900 dark:hover:text-gray-100"
                role="option"
                tabindex="-1"
              >
                <div>
                  <kbd
                    class="px-2 py-1.5 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded-lg dark:bg-gray-600 dark:text-gray-100 dark:border-gray-500"
                    >Shift+{{ index + 1 }}</kbd
                  >
                </div>
                <div class="ml-10 flex-auto truncate">
                  <!-- Active: "text-gray-900", Not Active: "text-gray-700" -->
                  <p class="text-sm font-medium text-gray-700 dark:text-gray-300 truncate">{{ tag.name }}</p>
                  <!-- Active: "text-gray-700", Not Active: "text-gray-500" -->
                  <p class="text-sm text-gray-500 truncate">{{ tag.description }}</p>
                </div>
              </li>
            </ul>

            <!-- Results, show/hide based on command palette state. -->
            <ul class="-mx-2 text-sm text-gray-700" id="options" role="listbox">
              <!-- Active: "bg-gray-100 text-gray-900" -->
              <li
                @click="() => acceptTag(tag)"
                v-for="(tag, index) in filteredTags"
                class="cursor-pointer group flex select-none items-center rounded-md p-2 hover:bg-gray-200 hover:text-gray-100 dark:hover:bg-gray-900 dark:hover:text-gray-100"
                role="option"
                tabindex="-1"
              >
                <div v-if="filteredTags.length === 1" class="h-6 w-6">
                  <kbd
                    class="px-2 py-1.5 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded-lg dark:bg-gray-600 dark:text-gray-100 dark:border-gray-500"
                    >Enter</kbd
                  >
                </div>
                <div v-else-if="index <= 5">
                  <kbd
                    class="px-2 py-1.5 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded-lg dark:bg-gray-600 dark:text-gray-100 dark:border-gray-500"
                    >Shift+{{ index + 1 }}</kbd
                  >
                </div>
                <TagIcon v-else class="h-6 w-6 text-gray-400 dark:text-gray-600" />
                <div class="ml-10 flex-auto truncate">
                  <!-- Active: "text-gray-900", Not Active: "text-gray-700" -->
                  <p class="text-sm font-medium text-gray-700 dark:text-gray-300 truncate">{{ tag.name }}</p>
                  <!-- Active: "text-gray-700", Not Active: "text-gray-500" -->
                  <p class="text-sm text-gray-500 truncate">{{ tag.description }}</p>
                </div>
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
          <p class="mt-4 font-semibold text-gray-900 dark:text-gray-100 underline cursor-pointer">
            Create custom tag '<b>{{ searchText }}</b
            >'
          </p>
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
import { HotkeyEvent, onHotkey } from "src/util/keyEvents";
import { Image } from "src/util/fileProcessor";
import { ImageTagsResponse } from "src/types/pocketbase";
import { tagStack } from "src/pages/image/imageQueryLogic";

interface Props {
  image: ImageWithTagsType | null;
  shown: boolean;
}
const props = withDefaults(defineProps<Props>(), {});
const emit = defineEmits<{
  selected: [ImageWithTagsType, ImageTagsResponse];
  close: [];
  "close-and-next": [];
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
    if (props.image?.expand.image_tag_assignments_via_image?.some((assignment) => assignment.imageTag === tag.id)) {
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

const recentTags = computed(() => {
  if (!tagStack.value) {
    return [];
  }
  let tags = [...tagStack.value];
  tags.reverse();

  return tags.slice(0, 5).filter((tag) => {
    if (tag.type === "default" || tag.type === "template") {
      return false;
    }
    if (props.image?.expand.image_tag_assignments_via_image?.some((assignment) => assignment.imageTag === tag.id)) {
      return false;
    }
    return true;
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
  if (filteredTags.value.length === 1) {
    acceptTag(filteredTags.value[0]);
  }
  if (filteredTags.value.length === 0 && searchText.value === "") {
    emit("close-and-next");
  }
}

onHotkey({ key: "1", modifierKeys: [`shift`] }, getAcceptTagIndexFunction(0));
onHotkey({ key: "2", modifierKeys: [`shift`] }, getAcceptTagIndexFunction(1));
onHotkey({ key: "3", modifierKeys: [`shift`] }, getAcceptTagIndexFunction(2));
onHotkey({ key: "4", modifierKeys: [`shift`] }, getAcceptTagIndexFunction(3));
onHotkey({ key: "5", modifierKeys: [`shift`] }, getAcceptTagIndexFunction(4));
function getAcceptTagIndexFunction(index: number) {
  return (event: HotkeyEvent) => {
    if (recentTags.value.length !== 0 && filteredTags.value.length === 0 && searchText.value === "") {
      if (recentTags.value.length > index) {
        event.event.preventDefault();
        acceptTag(recentTags.value[index]);
      }
    }

    if (filteredTags.value.length <= 1) {
      return;
    }
    if (filteredTags.value.length > index) {
      event.event.preventDefault();
      acceptTag(filteredTags.value[index]);
    }
  };
}

function acceptTag(tag: ImageTagsResponse) {
  if (!props.image) {
    return;
  }
  emit("selected", props.image, tag);
  userStore.addTagToStack(tag);
}

defineExpose({
  focusSearchText,
  clearSearchText,
});
</script>
