<template>
  <q-page class="row items-center justify-evenly">
    <div class="text-center">
      <div v-if="user" class="text-h2 q-mb-xl">Hello {{ user.firstName }} {{ user.lastName }}</div>
      <div class="row">
        <dashboard-button title="Profile" :callback="() => router.push(ownUserLink)" icon="person" />
        <dashboard-button v-if="userStore.isAdmin()" title="Users" :callback="() => router.push('/dashboard/users')" icon="people" />
        <dashboard-button title="Logout" :callback="logout" icon="logout" />
      </div>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import * as apiAuthorization from "src/api/authorization";
import DashboardButton from "src/components/dashboard/DashboardButton.vue";

const userStore = useUserStore();
const router = useRouter();

const user = ref(userStore.ownUser());
userStore.$subscribe((mutation, state) => {
  user.value = state.ownUserJson === "" ? null : JSON.parse(state.ownUserJson);
});

const ownUserLink = computed(() => {
  return user.value ? `/dashboard/users/${user.value.id}` : `/dashboard`;
});

const logout = () => {
  apiAuthorization.logout();
};
</script>
