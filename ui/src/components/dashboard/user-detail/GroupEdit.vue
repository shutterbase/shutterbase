<template>
  <div>
    <q-card-section class="sub-container">
      <div>
        <div class="sub-headline">User role</div>
      </div>
    </q-card-section>
    <!--<div class="sub-headline-underline"></div>
      <div>
        <q-checkbox v-for="group in groups" :key="group.id" class="item-3" v-model="group.active" :label="`.${group.description.split(' ').join('.').toLocaleLowerCase()}`" />
        <q-btn class="item-margins" label=".save.groups" color="green" @click="saveGroups" :disable="!changed" />
      </div>
    <q-dialog v-model="showSuccessMessage" seamless position="bottom">
      <q-card class="text-center q-py-sm q-px-lg" style="color: white; background-color: green">
        <div class="text-weight-bold">Success!</div>
        <div class="">User groups saved</div>
      </q-card>
    </q-dialog>-->
  </div>
</template>

<script setup lang="ts">
import { Role, getRoles, updateUserRole, User } from "src/api/user";
import { computed, onMounted, ref, watch } from "vue";
import { useUserStore } from "src/stores/user-store";

const userStore = useUserStore();

interface Props {
  user: User;
}
const props = withDefaults(defineProps<Props>(), {});

const ownUserDisplayed = computed(() => {
  return props.user.id === userStore.ownUser?.id;
});
</script>
