<template>
  <div class="card bg-white shadow-md p-10">
    <h1>Create new project</h1>
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
      <div class="mr-8"><button class="btn btn-primary" :disabled="!valid" @click="create">Create</button></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CreateProjectInput, Project } from "~/api/project";
import { ref, Ref } from "vue";
import { Method, getFetchOptions } from "~/api/common";

const name = ref("");
const description = ref("");

const valid = computed(() => {
  return name.value.length > 3 && description.value.length > 5;
});

async function create() {
  const createData = {} as CreateProjectInput;
  createData.name = name.value;
  createData.description = description.value;

  const { data } = await useFetch(`/projects`, getFetchOptions(Method.POST, createData));
  if (data) {
    const createdItem = data.value as Project;
    navigateTo(`/dashboard/projects/${createdItem.id}`);
  }
}
</script>
