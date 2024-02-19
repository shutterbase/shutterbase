<template>
  <div>
    <Table class="mx-auto max-w-7xl"></Table>
  </div>
</template>

<script setup lang="ts">
import { Ref, ref } from "vue";

import pb from "src/boot/pocketbase";
import Table from "src/components/Table.vue";

const show = ref(false);
const dialogError: Ref<any> = ref(null);

async function doStuff() {
  console.log("Doing stuff");
  try {
    const resultList = await pb.collection("foo").getList(1, 50, {});
  } catch (error) {
    dialogError.value = error;
    show.value = true;
  }
}
</script>
