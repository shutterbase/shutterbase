<template>
  <q-page class="row items-center justify-evenly">
    <q-card class="text-center register-card">
      <q-card-section>
        <div v-if="!showSuccessMessage && !showError" class="text-h5">Confirming your email address</div>
        <div v-else-if="!showError" class="text-h5">Your email address has been confirmed</div>
        <div v-else class="text-h5">Error confirming your email address</div>
      </q-card-section>
      <q-card-section>
        <div v-if="!showSuccessMessage && !showError">
          <div>Please wait while we are validating your email address</div>
          <q-circular-progress indeterminate size="50px" :thickness="0.22" rounded color="primary" track-color="grey-3" class="q-ma-md" />
        </div>
        <div v-else-if="showSuccessMessage" data-test="emailVerificationSuccess">
          <div>Please proceed to login</div>
          <q-icon name="check_circle" size="50px" color="green" />
        </div>
        <div v-else-if="showError">
          <div>{{ errorMessage }}</div>
          <q-icon name="error" size="50px" color="red" />
        </div>
      </q-card-section>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import * as apiAuthorization from "src/api/authorization";
import { ResponseCode } from "src/api/authorization";

const route = useRoute();
const router = useRouter();
const email = route.query.email as string;
const key = route.query.key as string;

const showSuccessMessage = ref(false);
const showErrorMessage = ref(false);
const showError = ref(false);
const errorMessage = ref(``);

const proceedToLogin = ref(false);

if (!email || email.length === 0 || !key || key.length === 0) {
  console.error("Email or validation key not provided");
  useRouter().push("/");
}

onMounted(async () => {
  const MIN_LOADING_TIME = 2000; // ms
  const startTime = new Date().getTime();
  const response = await apiAuthorization.confirmEmail({ email, key });
  console.log(response);

  const loadingTime = new Date().getTime() - startTime;
  if (loadingTime < MIN_LOADING_TIME) {
    await new Promise((resolve) => setTimeout(resolve, MIN_LOADING_TIME - loadingTime));
  }

  // handle happy case
  if (response.code === ResponseCode.OK) {
    showSuccessMessage.value = true;
    setTimeout(() => {
      router.push("/login");
    }, 5000);
    return;
  }

  // handle error codes
  switch (response.code) {
    case ResponseCode.EMAIL_ALREADY_VALIDATED:
      errorMessage.value = "This email address has already been validated. Please proceed to login.";
      proceedToLogin.value = true;
      break;
    case ResponseCode.KEY_INVALID:
      errorMessage.value = "The provided validation key is invalid.";
      break;
    default:
      errorMessage.value = "An error occurred. Please try again later.";
      break;
  }
  showErrorMessage.value = true;
  showError.value = true;

  if (proceedToLogin.value) {
    setTimeout(() => {
      router.push("/login");
    }, 5000);
  }
});
</script>

<style lang="sass" scoped>
.email-confirmation-card
  min-width: 375px
</style>
