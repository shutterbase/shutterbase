<template>
  <q-page class="screen-card-page">
    <q-card class="screen-card">
      <div v-if="loading" class="q-pa-md">Loading...</div>
      <div v-else-if="user">
        <div class="main-container">
          <div class="main-container-row">
            <div class="main-headline">User Profile</div>
          </div>
          <div class="main-container-row">
            <div class="main-headline-underline"></div>
          </div>
          <account-edit :user="user" />
          <password-edit :user="user" />
          <q-card-section v-if="userStore.isAdmin()" class="sub-container">
            <div>
              <div class="sub-headline">Changelog</div>
            </div>
            <div class="sub-headline-underline"></div>
            <div>
              <q-input class="item-2" v-model="createdAt" label=".created.at" readonly />
              <div class="item-2" readonly>{{ user.edges.createdBy }}</div>
            </div>
            <div>
              <q-input class="item-2" v-model="updatedAt" label=".updated.at" readonly />
              <div class="item-2" readonly>{{ user.edges.updatedBy }}</div>
            </div>
          </q-card-section>
        </div>
      </div>
      <div v-else>Error loading user</div>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { getUserById, User } from "src/api/user";
import { useRoute } from "vue-router";
import { useUserStore } from "src/stores/user-store";
import { toDateTime } from "src/utils/date";
import PasswordEdit from "src/components/dashboard/user-detail/PasswordEdit.vue";
import AccountEdit from "src/components/dashboard/user-detail/AccountEdit.vue";
import { emitter } from "src/boot/mitt";

const route = useRoute();
const userId = computed<string>(() => {
  if (typeof route?.params.id !== "string") return ``;
  return `${route.params.id}`;
});

const userStore = useUserStore();

const loading = ref(false);
const user = ref<User | null>(null);

const createdAt = computed(() => {
  if (!user.value) return ``;
  return toDateTime(user.value.createdAt);
});
const updatedAt = computed(() => {
  if (!user.value) return ``;
  return toDateTime(user.value.updatedAt);
});

const edit = ref(false);

const loadUser = async () => {
  if (!userId.value || userId.value === ``) return;
  loading.value = true;
  const result = await getUserById(userId.value);
  if (result.response.ok) {
    user.value = result.item || null;
  } else {
    user.value = null;
    emitter.emit(`error`, { title: "Error loading user", message: result.response.message });
  }
  loading.value = false;
};

// Load user on mount
onMounted(async () => {
  await loadUser();
});
// load user on userId change
watch(userId, async () => {
  await loadUser();
});
</script>

<style lang="sass" scoped></style>
