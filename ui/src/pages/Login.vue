<template>
  <section class="bg-gray-50 dark:bg-gray-900">
    <div class="flex flex-col items-center justify-center px-6 py-8 mx-auto md:h-screen lg:py-0">
      <a href="/" class="flex items-center mb-6 text-2xl font-semibold text-gray-900 dark:text-white">
        <img class="h-12 mr-2 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="logo" />
        <img class="h-12 mr-2 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="logo" />
      </a>
      <div class="w-full bg-white rounded-lg shadow dark:border md:mt-0 sm:max-w-md xl:p-0 dark:!bg-gray-800 dark:border-gray-700">
        <div class="p-6 space-y-4 md:space-y-6 sm:p-8">
          <h1 class="text-xl font-bold leading-tight tracking-tight text-gray-900 md:text-2xl dark:text-white">Sign in to your account</h1>
          <form class="space-y-4 md:space-y-6" action="#">
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
                autocomplete="current-password"
                id="password"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
              />
              <p v-if="passwordErrorMessage != ''" class="mt-2 text-sm text-red-600 dark:text-red-500">
                <span class="font-medium">{{ passwordErrorMessage }}</span>
              </p>
            </div>
            <div class="flex items-center justify-between">
              <a href="#" class="text-sm font-medium text-primary-600 hover:underline dark:text-primary-500">Forgot password?</a>
            </div>
            <button
              type="submit"
              @click.prevent="login"
              class="w-full text-white bg-primary-600 hover:bg-primary-700 focus:ring-4 focus:outline-none focus:ring-primary-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center dark:bg-primary-600 dark:hover:bg-primary-700 dark:focus:ring-primary-800"
            >
              Sign in
            </button>
            <p class="text-sm font-light text-gray-500 dark:text-gray-400">
              Don't have an account yet?
              <router-link to="/signup" class="font-medium text-primary-600 hover:underline dark:text-primary-500">Sign up</router-link>
            </p>
          </form>
        </div>
      </div>
    </div>
    <UnexpectedErrorMessage
      :show="showUnexpectedErrorMessage"
      :error="unexpectedError"
      :headline="unexpectedErrorHeadline"
      :message="unexpectedErrorMessage"
      @closed="showUnexpectedErrorMessage = false"
    />
  </section>
</template>
<script setup lang="ts">
import { ref } from "vue";
import pb from "src/boot/pocketbase";
import { useRouter } from "vue-router";
import * as EmailValidator from "email-validator";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";

const router = useRouter();

const email = ref("");
const emailErrorMessage = ref("");
function validateUsername() {
  if (email.value === "") {
    emailErrorMessage.value = "Please enter a username";
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
  }
  return true;
}

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const unexpectedErrorHeadline = ref("");
const unexpectedErrorMessage = ref("");

function clearErrorMessages() {
  emailErrorMessage.value = "";
  passwordErrorMessage.value = "";
}

async function login() {
  clearErrorMessages();

  if (!validateUsername() || !validatePassword()) {
    return;
  }

  try {
    const authData = await pb.collection("users").authWithPassword(email.value, password.value);
    if (authData) {
      router.push("/");
    }
  } catch (error: any) {
    if (error.status === 400) {
      passwordErrorMessage.value = "Invalid username or password";
      return;
    }

    if (error.response?.message) {
      unexpectedErrorHeadline.value = "Error";
      unexpectedErrorMessage.value = error.response.message;
    }

    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
