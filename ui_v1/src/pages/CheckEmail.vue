<template>
  <q-page class="row justify-evenly q-mt-xl">
    <div class="text-center row">
      <div class="col-12" data-test="checkEmailFrame">
        <div class="text-h4">Almost done...</div>
        <div class="text-bold q-pt-md">{{ actionMessage }}</div>
        <div v-if="action === actions.CONFIRM_ACCOUNT">
          <div style="height: 1px; border: 1px solid #bbb" class="q-my-lg"></div>
          <div v-if="!resent" class="">No email received? Check your spam folder or <span class="link" @click="requestNewEmail">request a new confirmation email</span></div>
          <div v-else class="">A new confirmation email has been sent</div>
          <div class="text-white bg-red q-pa-md q-mt-md" v-if="errorMessage">{{ errorMessage }}</div>
        </div>
      </div>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import { requestConfirmationEmail, ResponseCode } from "src/api/authorization";
import { QPage } from "quasar";
const route = useRoute();

const email = computed(() => route.query.email as string);
const action = computed(() => route.query.action as string);

enum actions {
  CONFIRM_ACCOUNT = "confirm-account",
  RESET_PASSWORD = "reset-password",
  UNKNOWN = "",
}

const actionMessage = computed(() => {
  switch (action.value) {
    case actions.CONFIRM_ACCOUNT:
      return `Please check your email (${email.value}) to confirm your account`;
    case actions.RESET_PASSWORD:
      return `Please check your email (${email.value}) to reset your password`;
    default:
      return `Please check your email (${email.value})`;
  }
});

const resent = ref(false);
const errorMessage = ref("");

const requestNewEmail = async () => {
  const response = await requestConfirmationEmail({ email: email.value });
  if (response.code === ResponseCode.OK) {
    resent.value = true;
    errorMessage.value = "";
    return;
  }

  switch (response.code) {
    case ResponseCode.TOO_MANY_REQUESTS:
      errorMessage.value = "You have requested too many confirmation emails. Please try again later";
      break;
    default:
      errorMessage.value = "An error occurred. Please try again later";
  }
};
</script>

<style lang="sass" scoped>
.link
  color: #3b82f6
  cursor: pointer
  &:hover
    text-decoration: underline
</style>
