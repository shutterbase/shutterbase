<template>
  <q-page class="row items-center justify-evenly">
    <q-card class="text-center register-card">
      <q-card-section>
        <div class="text-h5">Request Password Reset</div>
      </q-card-section>
      <q-card-section>
        <div class="row">
          <div class="col-12">
            <q-input class="col-12 q-ma-sm" data-test="email" v-model="email" ref="emailRef" :rules="emailRules" lazy-rules label=".email" type="email" outlined />
          </div>
        </div>
        <div v-if="showErrorMessage" class="row q-pa-sm q-mb-md">
          <q-banner class="col-12 text-center text-white bg-red" data-test="errorMessage">
            {{ errorMessage }}
          </q-banner>
        </div>
        <div class="row">
          <div class="col-12">
            <q-btn class="col-12 q-ma-sm" label=".request.new.password" color="primary" @click="requestNewPassword" />
          </div>
        </div>
        <div class="row q-pt-lg">
          <div class="col-12">
            <div>Already have login and password? <router-link data-test="registerButton" to="/login">Sign in</router-link></div>
          </div>
        </div>
      </q-card-section>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import * as EmailValidator from "email-validator";
import * as apiAuthorization from "src/api/authorization";
import { ResponseCode } from "src/api/authorization";

const router = useRouter();

const email = ref<string>("");
const emailRef = ref<any>(null);
const emailRules = [(v: string) => !!v || "email is required", (v: string) => EmailValidator.validate(v) || "email is invalid"];

const errorMessage = ref(``);
const showErrorMessage = ref(false);
const showSuccessMessage = ref(false);

function closeSuccessMessage() {
  showSuccessMessage.value = false;
  setTimeout(() => {
    router.push({ path: "/" });
  }, 1000);
}

const requestNewPassword = async () => {
  emailRef.value.validate();
  if (emailRef.value.hasError) {
    return;
  }
  const res = await apiAuthorization.requestPasswordReset({ email: email.value });
  // handle happy case
  if (res.code === ResponseCode.OK) {
    showSuccessMessage.value = true;
    router.push({ path: `/check-email`, query: { email: email.value, action: `reset-password` } });
    return;
  }

  // handle error codes
  switch (res.code) {
    case ResponseCode.NETWORK_ERROR:
    case ResponseCode.BAD_REQUEST:
    case ResponseCode.SERVER_ERROR:
    case ResponseCode.ERROR_RESET_PASSWORD:
      errorMessage.value = "An error occurred. Please try again later.";
      break;
    case ResponseCode.ERROR_SEND_EMAIL:
      errorMessage.value = "An error occurred while sending the email. Please contact an administrator to verify your account.";
      break;
    case ResponseCode.TOO_MANY_REQUESTS:
      errorMessage.value = "A new password has been requested less than a minute ago. Please wait and try again.";
      break;
  }
  showErrorMessage.value = true;
};
</script>

<style lang="sass" scoped>
.register-card
  width: 350px
  @media (max-width: $breakpoint-xs)
    width: 95%
</style>
