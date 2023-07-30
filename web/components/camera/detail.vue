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
    <div class="collapse bg-base-200">
      <div>
        <h2 class="text-xl font-medium p-5">Time offsets</h2>
        <client-only>
          <table class="table">
            <thead>
              <th>Created At</th>
              <th>Server Time</th>
              <th>Camera Time</th>
              <th>Offset</th>
              <th>Actions</th>
            </thead>
            <tbody>
              <tr v-for="timeOffset in timeOffsets" :key="timeOffset.id">
                <td>{{ getDateTimeString(timeOffset.createdAt) }}</td>
                <td>{{ getDateTimeString(timeOffset.serverTime) }}</td>
                <td>{{ getDateTimeString(timeOffset.cameraTime) }}</td>
                <td>{{ timeOffset.offsetSeconds }} seconds</td>
                <td>
                  <button class="btn btn-sm btn-error" @click="deleteTimeOffset(timeOffset.id)">Delete</button>
                </td>
              </tr>
            </tbody>
            <tfoot></tfoot>
          </table>
        </client-only>
      </div>
    </div>
    <div class="divider"></div>
    <div class="flex flex-row">
      <div class="mr-8"><button class="btn btn-secondary" @click="createTimeOffset">Create time offset</button></div>
      <div class="mr-8"><button class="btn btn-primary" :disabled="!modified" @click="update">Update</button></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { UpdateCameraInput, Camera } from "~/api/camera";
import { Method, getFetchOptions, getDateTimeString } from "~/api/common";

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
  userId: {
    type: String,
    required: true,
  },
});

const { data: item } = await useFetch(`/users/${props.userId}/cameras/${props.id}`, getFetchOptions(Method.GET));
const initialItem = item as Ref<Camera>;

const { data: timeOffsetsResult } = await useFetch(`/users/${props.userId}/cameras/${props.id}/time-offsets`, getFetchOptions(Method.GET));
const timeOffsets = computed(() => {
  return timeOffsetsResult.value.items;
});

const name = ref("");
const description = ref("");

function updateEditValues(editItem: Ref<Camera>) {
  name.value = editItem.value.name;
  description.value = editItem.value.description;
}

updateEditValues(item as Ref<Camera>);

const modified = computed(() => {
  return name.value !== initialItem.value.name || description.value !== initialItem.value.description;
});

async function update() {
  const updateData = {} as UpdateCameraInput;
  if (name.value !== initialItem.value.name) {
    updateData.name = name.value;
  }
  if (description.value !== initialItem.value.description) {
    updateData.description = description.value;
  }

  const { data } = await useFetch(`/users/${props.userId}/cameras/${props.id}`, getFetchOptions(Method.PUT, updateData));
  const updatedItem = data as Ref<Camera>;
  if (data) {
    initialItem.value.name = updatedItem.value.name;
    initialItem.value.description = updatedItem.value.description;

    initialItem.value.updatedAt = updatedItem.value.updatedAt;

    updateEditValues(initialItem);
  }
}

function createTimeOffset() {
  navigateTo(`/dashboard/users/${props.userId}/cameras/${props.id}/offset`);
}

async function deleteTimeOffset(id: string) {
  const { data } = await useFetch(`/users/${props.userId}/cameras/${props.id}/time-offsets/${id}`, getFetchOptions(Method.DELETE, {}));
  if (data) {
    navigateTo(`/dashboard/users/${props.userId}/cameras/${props.id}`);
  }
}
</script>
