<template>
  <q-drawer v-model="showDrawer" side="left" behavior="mobile" elevated>
    <q-list>
      <q-item-label header> Navigation </q-item-label>
      <q-item v-for="item in shownDrawerItems" :key="item.label" clickable tag="a" :href="item.to">
        <q-item-section v-if="item.icon" avatar>
          <q-icon :name="item.icon" />
        </q-item-section>

        <q-item-section>
          <q-item-label>{{ item.label }}</q-item-label>
          <q-item-label v-if="item.caption" caption>{{ item.caption }}</q-item-label>
        </q-item-section>
      </q-item>
    </q-list>
  </q-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useUserStore } from "../stores/user-store";
import { emitter } from "../boot/mitt";
import { storeToRefs } from "pinia";

const userStore = useUserStore();
const { ownUserJson } = storeToRefs(userStore);

const ownUserId = computed(() => (ownUserJson.value ? JSON.parse(ownUserJson.value)?.id : ""));

const showDrawer = ref(false);
emitter.on("toggleDrawer", () => {
  showDrawer.value = !showDrawer.value;
});
emitter.on("logout", () => {
  showDrawer.value = false;
});

const drawerItems = ref([
  {
    icon: "grid_view",
    label: "Dashboard",
    to: "/#/dashboard",
    show: () => true,
  },
  {
    icon: "person",
    label: "Profile",
    to: `/#/dashboard/users/${ownUserId.value}`,
    show: () => true,
  },
  {
    icon: "people",
    label: "Users",
    to: "/#/dashboard/users",
    show: () => userStore.isAdmin,
  },
  {
    icon: "logout",
    label: "Logout",
    to: "/#/logout",
    show: () => true,
  },
]);

const shownDrawerItems = computed(() => drawerItems.value.filter((item: any) => item.show()));
</script>
