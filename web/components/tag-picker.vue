<template>
  <div class="h-96">
    <div>
      <input v-model="tagSearchTerm" ref="tagSearchInput" type="text" class="form-control" placeholder="Search for tags" />
    </div>
    <table class="table table-xs">
      <thead>
        <tr>
          <th scope="col">Hotkey</th>
          <th scope="col">Tag</th>
          <th scope="col">Description</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(tag, index) in tags" :key="tag.id" :class="`click hover hover:cursor-pointer ${selectedIndex === index ? 'bg-green-400' : ''}`" @click="tagSelected(tag)">
          <td>
            <kbd v-if="index + 1 < 10">{{ index + 1 }}</kbd>
          </td>
          <td>{{ tag.name }}</td>
          <td>{{ tag.description }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { requestList } from "~/api/common";
import { storeToRefs } from "pinia";
import { Tag } from "~/api/tag";
import { emitter } from "~/boot/mitt";
const emit = defineEmits(["selected"]);
const store = useStore();

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  active: {
    type: Boolean,
    default: true,
  },
  showDefaultTags: {
    type: Boolean,
    default: false,
  },
});

const tagSearchTerm = ref("");
const tagSearchInput = ref<HTMLInputElement | null>(null);
const loadedTags = ref<Array<Tag>>([]);
const lastAppliedTags = ref<Array<Tag>>([]);

const selectedIndex = ref(-1);

async function loadTags() {
  const result = await requestList<Tag>(`/projects/${props.projectId}/tags`, { limit: 1000, search: tagSearchTerm.value });
  if (result.items && result.total !== undefined) {
    loadedTags.value = result.items;
  }
}

const tags = computed(() => {
  let result = loadedTags.value;
  if (!props.showDefaultTags) {
    result = result.filter((t) => t.type !== "default");
  }
  if (tagSearchTerm.value !== "") {
    result = result.filter((t) => {
      return t.name.toLowerCase().includes(tagSearchTerm.value.toLowerCase()) || t.description.toLowerCase().includes(tagSearchTerm.value.toLowerCase());
    });
  } else if (lastAppliedTags.value.length > 0) {
    result = lastAppliedTags.value;
  }
  return result;
});

const total = computed(() => {
  return tags.value.length;
});

let debounceTimeout: any = null;
watch(tagSearchTerm, async () => {
  if (!props.active) return;
  if (debounceTimeout) {
    clearTimeout(debounceTimeout);
  }
  debounceTimeout = setTimeout(async () => {
    await loadTags();
  }, 100);
});

await loadTags();

emitter.on("display-tag-picker", () => {
  tagSearchInput.value?.focus();
  tagSearchTerm.value = "";
  selectedIndex.value = -1;
});

emitter.on("key-shift-hotkey", (args: any) => {
  const { event, keyNumber } = args;
  event.preventDefault();
  const tag = tags.value[keyNumber - 1];
  if (tag) {
    tagSelected(tag);
  }
});

// TODO add SHIFT-Enter to select tag and keep tag picker open
emitter.on("key-Enter", (event: any) => {
  if (!props.active) return;
  event.preventDefault();
  if (tags.value.length === 1) {
    tagSelected(tags.value[0]);
  }
});

function getIndexOfTag(tag: Tag): number {
  return tags.value.findIndex((t) => t.id === tag.id);
}

function tagSelected(tag: Tag) {
  selectedIndex.value = getIndexOfTag(tag);
  if (!lastAppliedTags.value.some((t) => t.id === tag.id)) {
    lastAppliedTags.value = [tag, ...lastAppliedTags.value];
    if (lastAppliedTags.value.length > 9) {
      lastAppliedTags.value = lastAppliedTags.value.slice(0, 9);
    }
  }

  setTimeout(() => {
    emit("selected", tag);
  }, 50);
}
</script>
