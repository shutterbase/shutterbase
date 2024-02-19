<template>
  <div class="col-4">
    <div class="dashboard-item">
      <div class="items-center justify-evenly dashboard-item-content" outline @click="callback">
        <div class="row">
          <q-icon class="col" :name="icon" size="xl" />
        </div>
        <div class="row">
          <div class="col text-center">{{ title }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

interface Props {
  title: string;
  callback: () => void;
  icon: string;
}
const props = withDefaults(defineProps<Props>(), {});
</script>

<style lang="sass">
.dashboard-item
  margin: 10px
  .dashboard-item-content
    padding: 20px
    border-radius: 5px
    border: 1px solid #ddd
    cursor: pointer
    &:hover
      background-color: #eee
</style>
