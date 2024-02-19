<template>
  <ClientOnly>
    <input v-model="itemSearchTerm" type="text" placeholder="Search" class="input w-full max-w-xs" />
    <div class="overflow-x-auto">
      <table class="table">
        <tbody>
          <tr
            v-for="(item, index) in items"
            :key="item.id"
            :class="`click hover hover:cursor-pointer ${selectedIndex === index ? 'bg-green-400' : ''}`"
            @click="itemSelected(item)"
          >
            <td>{{ item.firstName }} {{ item.lastName }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </ClientOnly>
</template>

<script setup lang="ts">
import { requestList } from "~/api/common";
import { User } from "~/api/user";
type T = User;
const itemRequestUrl = "/users/minimal";

const emit = defineEmits(["selected"]);
const itemSearchTerm = ref("");
const items = ref<Array<T>>([]);
const selectedIndex = ref(-1);

async function loadItems() {
  const result = await requestList<T>(itemRequestUrl, { limit: 100, search: itemSearchTerm.value });
  if (result.items && result.total !== undefined) {
    items.value = result.items;
  }
}

function getItemIndex(item: T) {
  return items.value.findIndex((i) => i.id === item.id);
}

function itemSelected(item: T) {
  selectedIndex.value = getItemIndex(item);
  setTimeout(() => {
    emit("selected", item);
  }, 200);
}

watch(itemSearchTerm, loadItems, { immediate: true });
</script>
