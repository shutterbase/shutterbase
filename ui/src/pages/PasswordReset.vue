<template>
  <q-page class="row items-center justify-evenly">
    <q-card class="text-center register-card">
      <q-card-section>
        <div class="text-h5">Set new password</div>
      </q-card-section>
      <q-card-section>
        <div class="row">
          <div class="col-md-12 col-sm-12 col-xs-12">
            <q-input
              class="col-md-12 col-sm-12 q-ma-sm"
              data-test="password"
              v-model="password"
              ref="passwordRef"
              :rules="passwordRules"
              lazy-rules
              label=".password"
              type="password"
              outlined
            />
          </div>
          <div class="col-md-12 col-sm-12 col-xs-12" data-test="passwordConfirmationColumn">
            <q-input
              class="col-md-12 col-sm-12 q-ma-sm"
              data-test="passwordConfirmation"
              v-model="passwordConfirmation"
              ref="passwordConfirmationRef"
              :rules="passwordConfirmationRules"
              lazy-rules
              label=".password.confirmation"
              type="password"
              outlined
            />
          </div>
        </div>
        <password-quality-bar :passwordScore="passwordScore" />
        <div class="row">
          <div class="col-12">
            <q-btn class="col-12 q-ma-sm" label=".set.new.password" color="primary" @click="setNewPassword" />
          </div>
        </div>
      </q-card-section>
    </q-card>
    <q-dialog v-model="showErrorMessage" persistent>
      <q-card>
        <q-card-section>
          <div class="text-h6">Error</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          {{ errorMessage }}
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Ok" color="primary" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
    <q-dialog v-model="showSuccessMessage" persistent>
      <q-card>
        <q-card-section>
          <div class="text-h6">Success</div>
        </q-card-section>

        <q-card-section class="q-pt-none"> Your new password has been set. Please log in now. </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Ok" color="primary" @click="closeSuccessMessage" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import * as apiAuthorization from "src/api/authorization";
import { ResponseCode } from "src/api/authorization";
import * as zxcvbn from "zxcvbn";
import PasswordQualityBar from "src/components/PasswordQualityBar.vue";

const router = useRouter();

const email = computed(() => router.currentRoute.value.query.email as string);
const key = computed(() => router.currentRoute.value.query.key as string);

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

const errorMessage = ref(``);
const showErrorMessage = ref(false);

const showSuccessMessage = ref(false);

function closeSuccessMessage() {
  showSuccessMessage.value = false;
  setTimeout(() => {
    router.push({ path: "/" });
  }, 1000);
}

const setNewPassword = async () => {
  passwordRef.value.validate();
  passwordConfirmationRef.value.validate();
  if (passwordRef.value.hasError || passwordConfirmationRef.value.hasError) {
    return;
  }
  const res = await apiAuthorization.passwordReset({ email: email.value, key: key.value, password: password.value });
  // handle happy case
  if (res.code === ResponseCode.OK) {
    showSuccessMessage.value = true;
    return;
  }

  // handle error codes
  switch (res.code) {
    case ResponseCode.EMAIL_PASSWORD_REQUIRED:
    case ResponseCode.EMAIL_REQUIRED:
    case ResponseCode.PASSWORD_REQUIRED:
      errorMessage.value = "Email and password are required.";
      break;
    case ResponseCode.KEY_REQUIRED:
      errorMessage.value = "A valid password reset key is required.";
      break;
    case ResponseCode.KEY_INVALID:
      errorMessage.value = "The password reset key is invalid or has already been used. Please request a new password reset email.";
      break;
    case ResponseCode.NETWORK_ERROR:
    case ResponseCode.BAD_REQUEST:
    case ResponseCode.SERVER_ERROR:
    case ResponseCode.ERROR_RESET_PASSWORD:
    default:
      errorMessage.value = "An error occurred. Please try again later.";
      break;
  }
  showErrorMessage.value = true;
};
</script>

<style lang="sass" scoped>
.register-card
  min-width: 350px
  @media (max-width: $breakpoint-xs)
    width: 95%
  @media (max-width: $breakpoint-sm)
    max-width: 550px
  @media (max-width: $breakpoint-md)
    max-width: 650px
  @media (max-width: $breakpoint-lg)
    max-width: 750px
</style>
