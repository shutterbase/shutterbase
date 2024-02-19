<template>
  <q-page class="row justify-evenly"> </q-page>
</template>

<script setup lang="ts">
import { logout } from "src/api/authorization";
import { onMounted } from "vue";
onMounted(() => {
  logout();
});
</script>

<style lang="sass" scoped></style>
