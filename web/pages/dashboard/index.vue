<template>
  <NuxtLayout name="logged-in">
    <div class="">
      <TotalUserStat v-if="store.isAdmin()" class="mr-16"></TotalUserStat>
      <ProjectStat v-if="store.isAdmin()" class="mr-16"></ProjectStat>
      <UploadStat class="mr-16"></UploadStat>
    </div>
  </NuxtLayout>
</template>

<script setup lang="ts">
const store = useStore();
</script>

<style scoped></style>
