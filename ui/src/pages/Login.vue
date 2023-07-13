<template>
  <q-page class="row items-center justify-evenly">
    <q-card class="text-center register-card">
      <q-card-section>
        <div class="text-h5">Login</div>
      </q-card-section>
      <q-card-section>
        <q-form @submit="login">
          <div class="row">
            <div class="col-12">
              <q-input class="col-12 q-ma-sm" data-test="email" v-model="email" ref="emailRef" :rules="emailRules" lazy-rules label=".email" type="email" outlined />
            </div>
          </div>
          <div class="row">
            <div class="col-12">
              <q-input
                class="col-12 q-mx-sm"
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
          </div>
          <div class="row q-mb-sm">
            <div class="col-12 text-left">
              <q-checkbox data-test="rememberMe" v-model="rememberMe" label=".rememberMe" />
            </div>
          </div>
          <div v-if="showErrorMessage" class="row q-pa-sm q-my-md">
            <q-banner class="col-12 text-center text-white bg-red" data-test="errorMessage">
              {{ errorMessage }}
            </q-banner>
          </div>
          <div class="row">
            <div class="col-12 text-center q-pl-sm">
              <router-link data-test="resetPasswordButton" to="/request-password-reset">Forgot your password?</router-link>
            </div>
          </div>
          <div class="row q-py-md">
            <div class="col-12">
              <q-btn class="col-12 q-ma-sm" type="submit" data-test="loginButton" label=".login" color="primary" />
            </div>
          </div>
          <div class="row q-pt-md">
            <div class="col-12 text-center q-px-sm">
              <div>Don't have an account yet? <router-link data-test="registerButton" to="/register">Register now</router-link></div>
            </div>
          </div>
        </q-form>
      </q-card-section>
    </q-card>
    <!--<q-dialog v-model="showErrorMessage" persistent>
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
    </q-dialog>-->
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import * as EmailValidator from "email-validator";
import * as apiAuthorization from "src/api/authorization";
import { ResponseCode } from "src/api/authorization";
import { useRouter } from "vue-router";
import { emitter } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { useLoginStore } from "src/stores/login-store";

const router = useRouter();
const userStore = useUserStore();
const loginStore = useLoginStore();

if (loginStore.isLoggedIn) {
  router.push("/dashboard");
}

const email = ref<string>("");
const emailRef = ref<any>(null);
const emailRules = [(v: string) => !!v || "email is required", (v: string) => EmailValidator.validate(v) || "email is invalid"];

const password = ref<string>("");
const passwordRef = ref<any>(null);

const passwordRules = [(v: string) => !!v || "password is required"];

const rememberMe = ref<boolean>(false);

const showErrorMessage = ref<boolean>(false);
const errorMessage = ref<string>(``);
const showSuccessMessage = ref<boolean>(false);

const login = async () => {
  emailRef.value.validate();
  passwordRef.value.validate();
  if (emailRef.value.hasError || passwordRef.value.hasError) {
    return;
  }
  const response = await apiAuthorization.login({ email: email.value, password: password.value, rememberMe: rememberMe.value });
  console.log(response);

  // handle happy case
  if (response.code === ResponseCode.OK) {
    showSuccessMessage.value = true;
    router.push("/dashboard");
    return;
  }

  // handle error codes
  switch (response.code) {
    case ResponseCode.USER_NOT_ACTIVE:
      errorMessage.value = "The user account is not activated. Please wait for activation or get in touch with an administrator.";
      break;
    case ResponseCode.EMAIL_NOT_VALIDATED:
      router.push({ path: `/check-email`, query: { email: email.value, action: `confirm-account` } });
      break;
    case ResponseCode.LOGIN_PASSWORD_INVALID:
      errorMessage.value = "Username or password are invalid.";
      break;
    case ResponseCode.NETWORK_ERROR:
    case ResponseCode.BAD_REQUEST:
    case ResponseCode.SERVER_ERROR:
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
