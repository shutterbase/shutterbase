<template>
  <div>
    <q-card-section class="sub-container">
      <div>
        <div class="sub-headline">Change password</div>
      </div>
      <div class="sub-headline-underline"></div>
      <div>
        <q-input class="item-2" v-model="password" ref="passwordRef" :rules="passwordRules" lazy-rules label=".password" type="password" />
        <q-input
          class="item-2"
          v-model="passwordConfirmation"
          ref="passwordConfirmationRef"
          :rules="passwordConfirmationRules"
          lazy-rules
          label=".password.confirmation"
          type="password"
        />
        <password-quality-bar class="item-1" :passwordScore="passwordScore" />
        <q-btn class="item-margins" label=".change.password" color="green" :disable="!submittable" @click="savePassword" />
      </div>
    </q-card-section>
    <q-dialog v-model="showSuccessMessage" seamless position="bottom">
      <q-card class="text-center q-py-sm q-px-lg" style="color: white; background-color: green">
        <div class="text-weight-bold">Success!</div>
        <div class="">Password updated</div>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup lang="ts">
import { updateUser, User } from "src/api/user";
import { computed, onMounted, ref, watch } from "vue";
import * as zxcvbn from "zxcvbn";
import PasswordQualityBar from "src/components/PasswordQualityBar.vue";

interface Props {
  user: User;
}
const props = withDefaults(defineProps<Props>(), {});

const password = ref<string>("");
const passwordRef = ref<any>(null);

const passwordConfirmation = ref<string>("");
const passwordConfirmationRef = ref<any>(null);

const passwordScore = computed(() => {
  if (password.value.length === 0) {
    return -1;
  }
  return zxcvbn(password.value).score;
});

const passwordRules = [(v: string) => !!v || "password is required", (v: string) => passwordScore.value >= 3 || "password is too weak"];
const passwordConfirmationRules = [(v: string) => !!v || "password confirmation is required", (v: string) => v === password.value || "passwords do not match"];

const initialPassword = ref<string>(``);

const submittable = computed(() => {
  return (
    password.value !== initialPassword.value && passwordConfirmation.value !== initialPassword.value && password.value === passwordConfirmation.value && passwordScore.value >= 3
  );
});

const showSuccessMessage = ref(false);

const savePassword = async () => {
  passwordRef.value.validate();
  passwordConfirmationRef.value.validate();
  if (passwordRef.value.hasError || passwordConfirmationRef.value.hasError) {
    return;
  }

  const response = await updateUserPassword(props.user.id, password.value);
  if (response === ResponseCode.OK) {
    showSuccessMessage.value = true;
    initialPassword.value = password.value;
    setTimeout(() => {
      showSuccessMessage.value = false;
    }, 3000);
  } else {
    // TODO show error message
  }
};
</script>
