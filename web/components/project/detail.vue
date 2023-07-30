<template>
  <div class="card bg-white shadow-md p-10">
    <ItemDescriptorLine :item="initialItem" />
    <div class="divider"></div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <label class="label">
          <span class="label-text">Name</span>
        </label>
        <input type="text" placeholder="Name" v-model="name" class="input input-bordered w-full max-w-xs" />
      </div>
      <div>
        <label class="label">
          <span class="label-text">Description</span>
        </label>
        <input type="text" placeholder="Description" v-model="description" class="input input-bordered w-full max-w-xs" />
      </div>
    </div>

    <div class="divider"></div>

    <div class="flex flex-row">
      <div class="mr-8"><button class="btn btn-primary" :disabled="!modified" @click="update">Update</button></div>
      <div class="mr-8"><button class="btn btn-primary" @click="toTagsPage">Manage Tags</button></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { UpdateProjectInput, Project } from "~/api/project";
import { Method, getFetchOptions, getDateTimeString } from "~/api/common";

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

const { data: item } = await useFetch(`/projects/${props.id}`, getFetchOptions(Method.GET));
const initialItem = item as Ref<Project>;

const name = ref("");
const description = ref("");

function updateEditValues(editItem: Ref<Project>) {
  name.value = editItem.value.name;
  description.value = editItem.value.description;
}

updateEditValues(item as Ref<Project>);

const modified = computed(() => {
  return name.value !== initialItem.value.name || description.value !== initialItem.value.description;
});

async function update() {
  const updateData = {} as UpdateProjectInput;
  if (name.value !== initialItem.value.name) {
    updateData.name = name.value;
  }
  if (description.value !== initialItem.value.description) {
    updateData.description = description.value;
  }

  const { data } = await useFetch(`/projects/${props.id}`, getFetchOptions(Method.PUT, updateData));
  const updatedItem = data as Ref<Project>;
  if (data) {
    initialItem.value.name = updatedItem.value.name;
    initialItem.value.description = updatedItem.value.description;

    initialItem.value.updatedAt = updatedItem.value.updatedAt;

    updateEditValues(initialItem);
  }
}

function toTagsPage() {
  navigateTo(`/dashboard/projects/${props.id}/tags`);
}
</script>
