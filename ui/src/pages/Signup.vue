<template>
  <section class="bg-gray-50 dark:bg-gray-900">
    <div class="flex flex-col items-center justify-center px-6 py-8 mx-auto md:h-screen lg:py-0">
      <a href="/" class="flex items-center mb-6 text-2xl font-semibold text-gray-900 dark:text-white">
        <img class="h-12 mr-2 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="logo" />
        <img class="h-12 mr-2 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="logo" />
      </a>
      <div class="w-full bg-white rounded-lg shadow dark:border md:mt-0 sm:max-w-md xl:p-0 dark:!bg-gray-800 dark:border-gray-700">
        <div class="p-6 space-y-4 md:space-y-6 sm:p-8">
          <h1 class="text-xl font-bold leading-tight tracking-tight text-gray-900 md:text-2xl dark:text-white">Create a new account</h1>
          <form class="space-y-4 md:space-y-6" action="#">
            <div>
              <label for="firstName" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Your first name</label>
              <input
                v-model="firstName"
                type="text"
                name="firstName"
                autocomplete="given-name"
                id="firstName"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
                placeholder="John"
              />
              <p v-if="firstNameErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ firstNameErrorMessage }}</span>
              </p>
            </div>
            <div>
              <label for="email" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Your last name</label>
              <input
                v-model="lastName"
                type="text"
                name="lastName"
                autocomplete="family-name"
                id="lastName"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
                placeholder="Doe"
              />
              <p v-if="lastNameErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ lastNameErrorMessage }}</span>
              </p>
            </div>
            <div>
              <label for="email" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Your email</label>
              <input
                v-model="email"
                type="email"
                name="email"
                autocomplete="username"
                id="email"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
                placeholder="foo@bar.de"
              />
              <p v-if="emailErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ emailErrorMessage }}</span>
              </p>
            </div>
            <div>
              <label for="password" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Password</label>
              <input
                v-model="password"
                type="password"
                name="password"
                autocomplete="new-password"
                id="password"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
              />
              <p v-if="passwordErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ passwordErrorMessage }}</span>
              </p>
            </div>
            <div>
              <label for="passwordConfirmation" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Confirm password</label>
              <input
                v-model="passwordConfirmation"
                type="password"
                name="passwordConfirmation"
                autocomplete="new-password"
                id="passwordConfirmation"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
              />
              <p v-if="passwordConfirmationErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ passwordConfirmationErrorMessage }}</span>
              </p>
            </div>
            <button
              type="submit"
              @click.prevent="signup"
              class="w-full text-white bg-primary-600 hover:bg-primary-700 focus:ring-4 focus:outline-none focus:ring-primary-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center dark:bg-primary-600 dark:hover:bg-primary-700 dark:focus:ring-primary-800"
            >
              Create an account
            </button>
            <p class="text-sm font-light text-gray-500 dark:text-gray-400">
              Already have an account? <router-link to="login" class="font-medium text-primary-600 hover:underline dark:text-primary-500">Login here</router-link>
            </p>
          </form>
        </div>
      </div>
    </div>
    <ModalMessage
      :show="showSuccessMessage"
      @closed="router.push('login')"
      :type="MessageType.SUCCESS"
      message="Your account has been created. Please confirm your email and log in."
      close-text="Continue"
    />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref } from "vue";
import pb from "src/boot/pocketbase";
import * as EmailValidator from "email-validator";
import { zxcvbn } from "zxcvbn-typescript";
import { useRouter } from "vue-router";
import { MessageType } from "src/components/ModalMessage.vue";
import ModalMessage from "src/components/ModalMessage.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { initFlowbite } from "flowbite";

onMounted(() => {
  initFlowbite();
  document.getElementById("successButton")?.click();
});

const router = useRouter();

const firstName = ref("");
const firstNameErrorMessage = ref("");
function validateFirstName() {
  if (firstName.value === "") {
    firstNameErrorMessage.value = "Please enter your first name";
    return false;
  }
  return true;
}

const lastName = ref("");
const lastNameErrorMessage = ref("");
function validateLastName() {
  if (lastName.value === "") {
    lastNameErrorMessage.value = "Please enter your last name";
    return false;
  }
  return true;
}

const email = ref("");
const emailErrorMessage = ref("");
function validateUsername() {
  if (email.value === "") {
    emailErrorMessage.value = "Please enter an email";
    return false;
  } else {
    if (!EmailValidator.validate(email.value)) {
      emailErrorMessage.value = "Please enter a valid email";
      return false;
    }
  }
  return true;
}

const password = ref("");
const passwordErrorMessage = ref("");
function validatePassword() {
  if (password.value === "") {
    passwordErrorMessage.value = "Please enter a password";
    return false;
  } else if (password.value.length < 8) {
    passwordErrorMessage.value = "The password must be at least 8 characters long";
    return false;
  } else if (zxcvbn(password.value).score < 3) {
    passwordErrorMessage.value = "The password is too weak";
    return false;
  }
  return true;
}

const passwordConfirmation = ref("");
const passwordConfirmationErrorMessage = ref("");
function validatePasswordConfirmation() {
  if (passwordConfirmation.value === "") {
    passwordConfirmationErrorMessage.value = "Please confirm your password";
    return false;
  }
  if (password.value !== passwordConfirmation.value) {
    passwordConfirmationErrorMessage.value = "Passwords do not match";
    return false;
  }
  return true;
}

const showSuccessMessage = ref(false);
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function clearErrorMessages() {
  firstNameErrorMessage.value = "";
  lastNameErrorMessage.value = "";
  emailErrorMessage.value = "";
  passwordErrorMessage.value = "";
  passwordConfirmationErrorMessage.value = "";
}

function validate() {
  clearErrorMessages();

  let valid = true;
  if (!validateFirstName()) valid = false;
  if (!validateLastName()) valid = false;
  if (!validateUsername()) valid = false;
  if (!validatePassword()) valid = false;
  if (!validatePasswordConfirmation()) valid = false;
  return valid;
}

async function signup() {
  if (!validate()) {
    return;
  }

  const data = {
    email: email.value,
    emailVisibility: true,
    password: password.value,
    passwordConfirm: passwordConfirmation.value,
    firstName: firstName.value,
    lastName: lastName.value,
  };

  try {
    const record = await pb.collection("users").create(data);
    await pb.collection("users").requestVerification(email.value);
    if (record) {
      showSuccessMessage.value = true;
    }
  } catch (error: any) {
    if (error.originalError?.data?.data?.email) {
      emailErrorMessage.value = error.originalError?.data?.data?.email.message;
      return;
    }
    if (error.originalError?.data?.data?.firstName?.code === "validation_not_unique" && error.originalError?.data?.data?.lastName?.code === "validation_not_unique") {
      firstNameErrorMessage.value = "A user with this name already exists";
      lastNameErrorMessage.value = "A user with this name already exists";
      return;
    }

    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
