<template>
  <div>
    <div v-if="image.edges.tagAssignments && image.edges.tagAssignments.length !== 0" class="flex flex-row">
      <div class="btn btn-xs" @click="openTagPicker">Add Tags</div>
      <div
        v-for="tagAssignment in tags"
        :class="`badge ${getTagTypeStyle(tagAssignment.edges.tag)} object-center p-3 ml-2 hover click hover:cursor-pointer`"
        @click="requestRemoveTag(tagAssignment.edges.tag)"
      >
        {{ tagAssignment.edges.tag.name }}
      </div>
    </div>
    <div v-else>
      <div class="btn btn-xs" @click="openTagPicker">Add Tags</div>
      No tags applied
    </div>
    <input type="checkbox" :checked="showTagPicker" id="tagPicker" class="modal-toggle" />
    <div class="modal">
      <form method="dialog" class="modal-box w-11/12 max-w-5xl">
        <h3 class="font-bold text-lg">Pick a tag to add</h3>
        <TagPicker :projectId="props.projectId" :active="showTagPicker" @selected="tagSelected" />
        <div class="modal-action">
          <!-- if there is a button, it will close the modal -->
          <button class="btn">Close</button>
        </div>
      </form>
    </div>
    <input type="checkbox" id="removeTagDialog" :checked="showRemoveTagDialog" class="modal-toggle" />
    <div class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg">Remove Tag</h3>
        <p class="py-4">Remove tag {{ removeTagCandidate?.name }} from this image</p>
        <div class="modal-action">
          <label class="btn" @click="showRemoveTagDialog = false">Cancel</label>
          <label class="btn" @click="removeTag">OK</label>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Tag } from "~/api/tag";
import { Image, TagAssignment } from "~/api/image";
import { emitter } from "~/boot/mitt";
import { Method, getFetchOptions } from "~/api/common";
import { useStore } from "~/stores/store";

const store = useStore();
const emit = defineEmits(["tag-picker-state", "image-update"]);

const props = defineProps({
  image: {
    type: Object as PropType<Image>,
    required: true,
  },
  projectId: {
    type: String,
    required: true,
  },
});

const tags = ref<Array<TagAssignment>>([]);
watchEffect(() => {
  console.log(`computing tags`);
  if (props.image.edges && props.image.edges.tagAssignments) {
    tags.value = props.image.edges.tagAssignments.sort((a, b) => {
      // sort by type first: default, manual, suggested
      if (a.edges.tag.type === b.edges.tag.type) {
        // sort by name
        return a.edges.tag.name.localeCompare(b.edges.tag.name);
      } else {
        if (a.edges.tag.type === "default") {
          return -1;
        } else if (b.edges.tag.type === "default") {
          return 1;
        } else if (a.edges.tag.type === "manual") {
          return -1;
        } else if (b.edges.tag.type === "manual") {
          return 1;
        } else if (a.edges.tag.type === "suggested") {
          return -1;
        } else if (b.edges.tag.type === "suggested") {
          return 1;
        } else {
          return 0;
        }
      }
    });
  } else {
    tags.value = [];
  }
});

const showTagPicker = ref(false);
const showRemoveTagDialog = ref(false);

emitter.on("key-t", (event: any) => {
  openTagPickerWithHotkey(event);
});

emitter.on("key-Escape", () => {
  showTagPicker.value = false;
  emit("tag-picker-state", false);
});

function openTagPickerWithHotkey(event: any) {
  if (showTagPicker.value) return;
  event.preventDefault();
  emitter.emit("display-tag-picker", event);
  showTagPicker.value = true;
  emit("tag-picker-state", true);
}

function openTagPicker() {
  if (showTagPicker.value) return;
  emitter.emit("display-tag-picker");
  showTagPicker.value = true;
  emit("tag-picker-state", true);
}

async function tagSelected(tag: Tag) {
  showTagPicker.value = false;
  emit("tag-picker-state", false);
  if (props.image) {
    let currentTagAssignments: Array<TagAssignment> = [];
    if (props.image.edges && props.image.edges.tagAssignments) {
      currentTagAssignments = props.image.edges.tagAssignments;
    }
    const url = `/projects/${props.projectId}/images/${props.image.id}`;
    const response = await useFetch(
      url,
      getFetchOptions(Method.PUT, {
        tags: [
          ...currentTagAssignments.map((t: TagAssignment) => {
            return {
              type: t.edges.tag.type,
              id: t.edges.tag.id,
            };
          }),
          {
            type: "manual",
            id: tag.id,
          },
        ],
      })
    );
    if (response.data.value) {
      const data = response.data.value as Image;
      const ownUser = store.getOwnUser();
      const updatedImage = props.image;
      updatedImage.updatedAt = data.updatedAt;
      // @ts-ignore
      if (!updatedImage.edges) updatedImage.edges = {};
      if (ownUser) {
        updatedImage.edges.updatedBy = ownUser;
      }
      updatedImage.edges.tagAssignments = [
        // @ts-ignore
        ...currentTagAssignments,
        {
          type: "manual",
          // @ts-ignore
          edges: {
            tag: tag,
          },
        },
      ];
      emit("image-update", updatedImage);
    }
  }
}

const removeTagCandidate = ref<Tag | null>(null);
function requestRemoveTag(tag: Tag) {
  if (tag.type === "default") return;
  removeTagCandidate.value = tag;
  showRemoveTagDialog.value = true;
}
async function removeTag() {
  showRemoveTagDialog.value = false;
  if (props.image) {
    let currentTagAssignments: Array<TagAssignment> = [];
    if (props.image.edges && props.image.edges.tagAssignments) {
      currentTagAssignments = props.image.edges.tagAssignments.filter((t: TagAssignment) => t.edges.tag.id !== removeTagCandidate.value?.id);
    }
    const url = `/projects/${props.projectId}/images/${props.image.id}`;
    const response = await useFetch(
      url,
      getFetchOptions(Method.PUT, {
        tags: [
          ...currentTagAssignments.map((t: TagAssignment) => {
            return {
              type: t.edges.tag.type,
              id: t.edges.tag.id,
            };
          }),
        ],
      })
    );
    if (response.data.value) {
      const data = response.data.value as Image;
      const ownUser = store.getOwnUser();
      const updatedImage = props.image;
      updatedImage.updatedAt = data.updatedAt;
      // @ts-ignore
      if (!updatedImage.edges) updatedImage.edges = {};
      if (ownUser) {
        updatedImage.edges.updatedBy = ownUser;
      }
      updatedImage.edges.tagAssignments = [
        // @ts-ignore
        ...currentTagAssignments.filter((t: TagAssignment) => t.edges.tag.id !== removeTagCandidate.value?.id),
      ];
      emit("image-update", updatedImage);
    }
  }
}

function getTagTypeStyle(tag: Tag): string {
  switch (tag.type) {
    case "default":
      return "badge-ghost";
    case "manual":
      return "badge-primary";
    case "suggested":
      return "badge-primary badge-outline";
    default:
      return "badge-ghost";
  }
}
</script>
