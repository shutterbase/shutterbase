<template>
  <section class="flex min-h-screen bg-surface dark:bg-primary-950">
    <!-- brand panel -->
    <aside class="relative hidden w-[46%] flex-col justify-between overflow-hidden bg-primary-950 p-12 lg:flex xl:w-1/2">
      <img src="~assets/img/shutterbase-icon.png" alt="" aria-hidden="true" class="pointer-events-none absolute -bottom-40 -right-40 w-[48rem] select-none opacity-[0.05]" />
      <img src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" class="relative h-7 w-auto self-start" />
      <div class="relative max-w-md">
        <p class="label-mono text-accent-400">Collaborative photography</p>
        <div class="relative mt-5 inline-block px-3 py-2">
          <CornerMarks />
          <h2 class="display text-[2.75rem] leading-[1.04] text-white">Join your<br />team.</h2>
        </div>
        <p class="mt-6 max-w-sm text-base leading-relaxed text-primary-300">Upload, time-sync across photographers, tag, and find any frame — together, in one shared library.</p>
        <ul class="mt-10 space-y-3.5">
          <li v-for="f in features" :key="f.label" class="flex items-center gap-3.5 text-sm text-primary-200">
            <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md border border-white/10 bg-white/[0.05] text-accent-300">
              <component :is="f.icon" class="h-[18px] w-[18px]" />
            </span>
            {{ f.label }}
          </li>
        </ul>
      </div>
      <div class="relative text-xs text-primary-500">© {{ year }} shutterbase</div>
    </aside>

    <!-- form panel -->
    <main class="flex w-full flex-1 items-center justify-center px-6 py-12 sm:px-12">
      <div class="w-full max-w-sm">
        <router-link to="/" class="mb-12 inline-block lg:hidden">
          <img class="h-9 dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
          <img class="hidden h-9 dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
        </router-link>

        <p class="label-mono text-accent-600 dark:text-accent-400">Get started</p>
        <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Create your account</h1>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Set up your details to join the shared library.</p>

        <form class="mt-9 space-y-5" action="#">
          <div>
            <label for="firstName" class="label-mono block text-primary-500 dark:text-primary-400">First name</label>
            <input
              v-model="firstName"
              type="text"
              name="firstName"
              autocomplete="given-name"
              id="firstName"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              placeholder="John"
            />
            <p v-if="firstNameErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ firstNameErrorMessage }}</p>
          </div>
          <div>
            <label for="lastName" class="label-mono block text-primary-500 dark:text-primary-400">Last name</label>
            <input
              v-model="lastName"
              type="text"
              name="lastName"
              autocomplete="family-name"
              id="lastName"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              placeholder="Doe"
            />
            <p v-if="lastNameErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ lastNameErrorMessage }}</p>
          </div>
          <div>
            <label for="email" class="label-mono block text-primary-500 dark:text-primary-400">Email</label>
            <input
              v-model="email"
              type="email"
              name="email"
              autocomplete="username"
              id="email"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              placeholder="you@example.com"
            />
            <p v-if="emailErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ emailErrorMessage }}</p>
          </div>
          <div>
            <label for="password" class="label-mono block text-primary-500 dark:text-primary-400">Password</label>
            <input
              v-model="password"
              type="password"
              name="password"
              autocomplete="new-password"
              id="password"
              placeholder="••••••••"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
            />
            <p v-if="passwordErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ passwordErrorMessage }}</p>
          </div>
          <div>
            <label for="passwordConfirmation" class="label-mono block text-primary-500 dark:text-primary-400">Confirm password</label>
            <input
              v-model="passwordConfirmation"
              type="password"
              name="passwordConfirmation"
              autocomplete="new-password"
              id="passwordConfirmation"
              placeholder="••••••••"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
            />
            <p v-if="passwordConfirmationErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ passwordConfirmationErrorMessage }}</p>
          </div>
          <button
            type="submit"
            @click.prevent="signup"
            class="flex h-11 w-full items-center justify-center rounded-md bg-accent-600 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface active:bg-accent-700 dark:focus-visible:ring-offset-primary-950"
          >
            Create an account
          </button>
        </form>

        <p class="mt-8 text-sm text-primary-500 dark:text-primary-400">
          Already have an account?
          <router-link to="login" class="font-medium text-accent-600 transition-colors hover:text-accent-500 dark:text-accent-400">Sign in</router-link>
        </p>
      </div>
    </main>

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
import * as EmailValidator from "email-validator";
import { zxcvbn } from "zxcvbn-typescript";
import { useRouter } from "vue-router";
import { MessageType } from "src/components/ModalMessage.vue";
import ModalMessage from "src/components/ModalMessage.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import CornerMarks from "src/components/CornerMarks.vue";
import { ClockIcon, TagIcon, MagnifyingGlassIcon } from "@heroicons/vue/24/outline";
import { initFlowbite } from "flowbite";

onMounted(() => {
  initFlowbite();
  document.getElementById("successButton")?.click();
});

const router = useRouter();

const year = new Date().getFullYear();
const features = [
  { icon: ClockIcon, label: "Time-sync every camera to one timeline" },
  { icon: TagIcon, label: "Tag collaboratively as a team" },
  { icon: MagnifyingGlassIcon, label: "Find any frame in seconds" },
];

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

// Self-signup is removed in the REST rewrite (§4.12: POST /users is admin-only).
// Accounts are provisioned by an administrator; this form now points users there.
async function signup() {
  if (!validate()) {
    return;
  }
  emailErrorMessage.value = "Self-signup is disabled. Please ask an administrator to create your account.";
}
</script>
