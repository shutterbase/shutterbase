<template>
  <div>
    <q-card-section class="sub-container">
      <div>
        <div class="sub-headline">Account</div>
      </div>
      <div class="sub-headline-underline"></div>

      <div>
        <q-input class="item-2" v-model="user.firstName" label=".first.name" />
        <q-input class="item-2" v-model="user.lastName" label=".last.name" />
      </div>
      <div>
        <q-input class="item-2" v-model="user.email" label=".email" readonly />
        <q-select
          :options="USER_LANGUAGE_OPTIONS"
          option-value="value"
          option-label="text"
          map-options
          emit-value
          class="item-2"
          v-model="user.locale"
          label=".preferred.language"
        />
      </div>
      <div v-if="userStore.isAdmin">
        <q-checkbox class="item-2" v-model="user.emailValidated" label=".email.validated" readonly />
        <q-checkbox class="item-2" v-model="user.active" label=".account.active" />
      </div>
      <div>
        <q-btn class="item-margins" label=".save.account.settings" color="green" :disable="!submittable" @click="saveSettings" />
      </div>
    </q-card-section>
    <q-dialog v-model="showSuccessMessage" seamless position="bottom">
      <q-card class="text-center q-py-sm q-px-lg" style="color: white; background-color: green">
        <div class="text-weight-bold">Success!</div>
        <div class="">User settings saved</div>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup lang="ts">
import { Role, getRoles, updateUserRole, User, updateUser, UpdateUserInput } from "src/api/user";
import { computed, onMounted, ref, watch } from "vue";
import { useUserStore } from "src/stores/user-store";
import { QCardSection, QInput, QSelect, QCheckbox, QBtn, QDialog, QCard } from "quasar";

const userStore = useUserStore();

interface Props {
  user: User;
}
const props = withDefaults(defineProps<Props>(), {});

const initialUser = ref<User>({} as User);

const ownUserDisplayed = computed(() => {
  return props.user.id === userStore.ownUser?.id;
});

const setInitialUserState = () => {
  initialUser.value = JSON.parse(JSON.stringify(props.user));
};

const submittable = computed(() => {
  if (!initialUser.value) return false;
  if (initialUser.value.firstName !== props.user.firstName) return true;
  if (initialUser.value.lastName !== props.user.lastName) return true;
  if (initialUser.value.locale !== props.user.locale) return true;
  if (initialUser.value.active !== props.user.active) return true;
  if (initialUser.value.emailValidated !== props.user.emailValidated) return true;
  return false;
});

watch(
  () => props.user,
  () => {
    setInitialUserState();
  },
  { immediate: true }
);

const showSuccessMessage = ref(false);

const saveSettings = async () => {
  const editUserData: UpdateUserInput = {
    firstName: props.user.firstName,
    lastName: props.user.lastName,
    locale: props.user.locale,
  };

  if (userStore.isAdmin) {
    editUserData.active = props.user.active;
    editUserData.emailValidated = props.user.emailValidated;
  }

  const response = await updateUser(props.user.id, editUserData);
  if (response === ResponseCode.OK) {
    showSuccessMessage.value = true;
    setInitialUserState();
    if (ownUserDisplayed.value) {
      userStore.updateUser();
    }
    setTimeout(() => {
      showSuccessMessage.value = false;
    }, 3000);
  } else {
    // TODO show error message
  }
};
</script>
