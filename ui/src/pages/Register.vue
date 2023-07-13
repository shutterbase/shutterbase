<template>
  <q-page class="screen-card-page">
    <q-card class="text-center screen-card">
      <q-card-section>
        <div class="text-h5" data-test="headline">Register</div>
      </q-card-section>
      <q-card-section>
        <div class="row">
          <div class="col-md-6 col-sm-6 col-xs-12">
            <q-input class="q-ma-sm" data-test="firstName" v-model="firstName" ref="firstNameRef" :rules="firstNameRules" lazy-rules label=".first.name" outlined />
          </div>
          <div class="col-md-6 col-sm-6 col-xs-12">
            <q-input class="col-md-6 col-sm-12 q-ma-sm" data-test="lastName" v-model="lastName" ref="lastNameRef" :rules="lastNameRules" lazy-rules label=".last.name" outlined />
          </div>
        </div>
        <div class="row">
          <div class="col-12" data-test="emailColumn">
            <q-input class="col-12 q-ma-sm" data-test="email" v-model="email" ref="emailRef" :rules="emailRules" lazy-rules label=".email" type="email" outlined />
          </div>
        </div>
        <div class="row">
          <div class="col-md-6 col-sm-6 col-xs-12">
            <q-input
              class="col-md-6 col-sm-12 q-ma-sm"
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
          <div class="col-md-6 col-sm-6 col-xs-12" data-test="passwordConfirmationColumn">
            <q-input
              class="col-md-6 col-sm-12 q-ma-sm"
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
        <div v-if="showErrorMessage" class="row q-pa-sm q-mb-md">
          <q-banner class="col-12 text-center text-white bg-red" data-test="errorMessage">
            {{ errorMessage }}
          </q-banner>
        </div>
        <div class="row">
          <div class="col-12">
            <q-btn class="col-12 q-ma-sm" data-test="registerButton" label=".register" color="primary" @click="register" />
          </div>
        </div>
        <div class="row q-pt-lg">
          <div class="col-12">
            <div>Already have login and password? <router-link data-test="signinButton" to="/login">Sign in</router-link></div>
          </div>
        </div>
      </q-card-section>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import zxcvbn from "zxcvbn";
import * as EmailValidator from "email-validator";
import * as apiAuthorization from "src/api/authorization";
import { ResponseCode } from "src/api/authorization";
import { useRouter } from "vue-router";
import PasswordQualityBar from "src/components/PasswordQualityBar.vue";

const router = useRouter();

const email = ref<string>("");
const emailRef = ref<any>(null);
const emailRules = [(v: string) => !!v || "email is required", (v: string) => EmailValidator.validate(v) || "email is invalid"];

const firstName = ref<string>("");
const firstNameRef = ref<any>(null);
const firstNameRules = [(v: string) => !!v || "first name is required"];

const lastName = ref<string>("");
const lastNameRef = ref<any>(null);
const lastNameRules = [(v: string) => !!v || "last name is required"];

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
  router.push({ path: "/" });
}

const register = async () => {
  emailRef.value.validate();
  firstNameRef.value.validate();
  lastNameRef.value.validate();
  passwordRef.value.validate();
  passwordConfirmationRef.value.validate();
  if (emailRef.value.hasError || firstNameRef.value.hasError || lastNameRef.value.hasError || passwordRef.value.hasError || passwordConfirmationRef.value.hasError) {
    return;
  }
  const res = await apiAuthorization.register({
    email: email.value,
    firstName: firstName.value,
    lastName: lastName.value,
    password: password.value,
  });
  console.log(res);

  // handle happy case
  if (res.code === ResponseCode.OK) {
    showSuccessMessage.value = true;
    router.push({ path: `/check-email`, query: { email: email.value, action: `confirm-account` } });
    return;
  }

  // handle error codes
  switch (res.code) {
    case ResponseCode.NETWORK_ERROR:
    case ResponseCode.BAD_REQUEST:
    case ResponseCode.SERVER_ERROR:
    case ResponseCode.ERROR_CREATE_USER:
      errorMessage.value = "An error occurred. Please try again later.";
      break;
    case ResponseCode.ERROR_SEND_EMAIL:
      errorMessage.value = "An error occurred while sending the email. Please contact an administrator to verify your account.";
      break;
    case ResponseCode.ERROR_PASSWORD:
      errorMessage.value = "An error occurred while setting your password. Please try again with a different password.";
      break;
    case ResponseCode.USER_EXISTS:
      errorMessage.value = "An account with this email already exists. Please login or reset your password.";
      break;
  }
  showErrorMessage.value = true;
};
</script>

<style lang="sass" scoped>
.register-card
  @media (max-width: $breakpoint-xs)
    width: 95%
  @media (max-width: $breakpoint-sm)
    max-width: 550px
  @media (max-width: $breakpoint-md)
    max-width: 650px
  @media (max-width: $breakpoint-lg)
    max-width: 750px
</style>
