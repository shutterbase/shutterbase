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
    <div v-show="shown" class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity"></div>

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
        class="mx-auto max-w-3xl transform divide-y divide-primary-200 dark:divide-primary-800 overflow-hidden rounded-xl border border-primary-200 bg-surface dark:border-primary-800 dark:bg-surface-dark shadow-2xl transition-all"
      >
        <div class="relative">
          <svg class="pointer-events-none absolute left-4 top-3.5 h-5 w-5 text-primary-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
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
            class="h-12 w-full border-0 bg-transparent pl-11 pr-4 text-sm text-primary-900 placeholder:text-primary-400 focus:outline-none focus:ring-0 dark:text-primary-100 dark:placeholder:text-primary-500"
            placeholder="Search tag..."
          />
        </div>

        <div
          v-if="(filteredTags.length !== 0 && searchText !== '') || (recentTags.length !== 0 && searchText === '')"
          class="flex transform-gpu divide-x divide-primary-200 dark:divide-primary-800"
        >
          <!-- Preview Visible: "sm:h-96" -->
          <div class="max-h-96 min-w-0 flex-auto scroll-py-4 overflow-y-auto px-6 py-4 sm:h-96">
            <!-- Default state, show/hide based on command palette state. -->
            <h2 v-if="filteredTags.length === 0 && searchText === ''" class="label-mono mb-4 mt-2 text-primary-500 dark:text-primary-400">Recent tags</h2>
            <ul v-if="filteredTags.length === 0 && searchText === ''" class="-mx-2 text-sm text-primary-700 dark:text-primary-200" id="options" role="listbox">
              <li
                @click="() => acceptTag(tag)"
                @keydown.enter="() => acceptTag(tag)"
                @keydown.space.prevent="() => acceptTag(tag)"
                v-for="(tag, index) in recentTags"
                class="cursor-pointer group flex select-none items-center rounded-md p-2 transition-colors hover:bg-primary-100 dark:hover:bg-primary-800"
                role="option"
                tabindex="0"
              >
                <div>
                  <kbd
                    v-if="acceptKeyLabel(index)"
                    class="font-data px-2 py-1.5 text-xs font-semibold text-primary-700 bg-primary-100 border border-primary-200 rounded-lg dark:bg-primary-800 dark:text-primary-200 dark:border-primary-700"
                    >{{ acceptKeyLabel(index) }}</kbd
                  >
                </div>
                <div class="ml-10 flex-auto truncate">
                  <p class="text-sm font-medium text-primary-800 dark:text-primary-100 truncate">{{ tag.name }}</p>
                  <p class="text-sm text-primary-500 dark:text-primary-400 truncate">{{ tag.description }}</p>
                </div>
              </li>
            </ul>

            <!-- Results, show/hide based on command palette state. -->
            <ul class="-mx-2 text-sm text-primary-700 dark:text-primary-200" id="options" role="listbox">
              <li
                @click="() => acceptTag(tag)"
                @keydown.enter="() => acceptTag(tag)"
                @keydown.space.prevent="() => acceptTag(tag)"
                v-for="(tag, index) in filteredTags"
                class="cursor-pointer group flex select-none items-center rounded-md p-2 transition-colors hover:bg-primary-100 dark:hover:bg-primary-800"
                role="option"
                tabindex="0"
              >
                <div v-if="filteredTags.length === 1 && enterKeyLabel" class="h-6 w-6">
                  <kbd
                    class="font-data px-2 py-1.5 text-xs font-semibold text-primary-700 bg-primary-100 border border-primary-200 rounded-lg dark:bg-primary-800 dark:text-primary-200 dark:border-primary-700"
                    >{{ enterKeyLabel }}</kbd
                  >
                </div>
                <div v-else-if="index <= 5 && acceptKeyLabel(index)">
                  <kbd
                    class="font-data px-2 py-1.5 text-xs font-semibold text-primary-700 bg-primary-100 border border-primary-200 rounded-lg dark:bg-primary-800 dark:text-primary-200 dark:border-primary-700"
                    >{{ acceptKeyLabel(index) }}</kbd
                  >
                </div>
                <TagIcon v-else class="h-6 w-6 text-primary-400 dark:text-primary-500" />
                <div class="ml-10 flex-auto truncate">
                  <p class="text-sm font-medium text-primary-800 dark:text-primary-100 truncate">{{ tag.name }}</p>
                  <p class="text-sm text-primary-500 dark:text-primary-400 truncate">{{ tag.description }}</p>
                </div>
              </li>
            </ul>
          </div>
        </div>

        <!-- Empty state, show/hide based on command palette state -->
        <div v-if="filteredTags.length === 0 && searchText !== ''" class="px-6 py-14 text-center text-sm sm:px-14">
          <TagIcon class="mx-auto h-6 w-6 text-primary-400 dark:text-primary-500" />
          <p class="mt-4 font-semibold text-primary-900 dark:text-white">No matching tags</p>
          <p class="mt-2 text-primary-500 dark:text-primary-400">No tag matching your search could be found. Please use a different keyword or create a 'custom' tag</p>
          <p
            @click="createCustomTag"
            @keydown.enter="createCustomTag"
            @keydown.space.prevent="createCustomTag"
            role="button"
            tabindex="0"
            class="mt-4 font-semibold text-accent-600 hover:text-accent-500 dark:text-accent-400 underline cursor-pointer"
          >
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
import { Ref, computed, nextTick, onUnmounted, ref, watch } from "vue";
import { emitter } from "src/boot/mitt";
import { debug } from "src/util/logger";
import { actionKeys, formatCombo, pushHotkeyContext, useHotkeyAction } from "src/util/hotkeys";
import { Image } from "src/util/fileProcessor";
import { ImageTagsResponse } from "src/types/pocketbase";
import { tagStack } from "src/pages/image/imageQueryLogic";
import { api } from "src/api";

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
    if (tag.type === "template") {
      return false;
    }
    if (tag.type === "default" && !userStore.isProjectAdminOrHigher()) {
      return false;
    }
    if (props.image?.tags?.some((assignment) => assignment.tag.id === tag.id)) {
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
    if (props.image?.tags?.some((assignment) => assignment.tag.id === tag.id)) {
      return false;
    }
    return true;
  });
});

// kbd hints reflect the user's effective bindings, not the shipped defaults
function acceptKeyLabel(index: number): string {
  const keys = actionKeys(userStore.user?.hotkeys, `tagging.accept-${index + 1}`);
  return keys.length ? formatCombo(keys[0]) : "";
}
const enterKeyLabel = computed(() => {
  const keys = actionKeys(userStore.user?.hotkeys, "tagging.accept-only-result");
  return keys.length ? formatCombo(keys[0]) : "";
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

// While shown, this dialog owns the hotkey context: gallery hotkeys pause and
// the accept/close actions below become active (they fire inside the search
// input too — allowInInputs on their action definitions).
let popContext: (() => void) | null = null;
watch(
  () => props.shown,
  (shown) => {
    if (shown && !popContext) {
      popContext = pushHotkeyContext("tagging-dialog");
    } else if (!shown && popContext) {
      popContext();
      popContext = null;
    }
  },
  { immediate: true },
);
onUnmounted(() => {
  popContext?.();
  popContext = null;
});

useHotkeyAction("tagging.accept-only-result", acceptOnlyResult);
function acceptOnlyResult() {
  if (filteredTags.value.length === 1) {
    acceptTag(filteredTags.value[0]);
  }
  if (filteredTags.value.length === 0 && searchText.value === "") {
    emit("close-and-next");
  }
}

for (let index = 0; index < 5; index++) {
  useHotkeyAction(`tagging.accept-${index + 1}`, getAcceptTagIndexFunction(index));
}
function getAcceptTagIndexFunction(index: number) {
  return () => {
    if (recentTags.value.length !== 0 && filteredTags.value.length === 0 && searchText.value === "") {
      if (recentTags.value.length > index) {
        acceptTag(recentTags.value[index]);
      }
    }

    if (filteredTags.value.length <= 1) {
      return;
    }
    if (filteredTags.value.length > index) {
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

async function createCustomTag() {
  if (!props.image) {
    return;
  }
  try {
    const createdTag = await api.imageTags.create({
      name: searchText.value,
      description: `Custom tag '${searchText.value}'`,
      type: "custom",
      projectId: props.image.project.id,
    });
    userStore.projectTags.push(createdTag);
    acceptTag(createdTag);
  } catch (error: any) {
    emitter.emit(`notification`, {
      headline: `Error creating custom tag`,
      type: "error",
    });
  }
}

defineExpose({
  focusSearchText,
  clearSearchText,
});
</script>
