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

    <!-- signup panel: accounts are created inactive and activated by an admin -->
    <main class="flex w-full flex-1 items-center justify-center px-6 py-12 sm:px-12">
      <div class="w-full max-w-sm">
        <router-link to="/" class="mb-12 inline-block lg:hidden">
          <img class="h-9 dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
          <img class="hidden h-9 dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
        </router-link>

        <span
          class="flex h-11 w-11 items-center justify-center rounded-md border border-primary-200 bg-surface-muted text-accent-600 dark:border-primary-700 dark:bg-primary-900 dark:text-accent-300"
        >
          <UserPlusIcon class="h-5 w-5" />
        </span>

        <!-- submitted -->
        <template v-if="submitted">
          <p class="label-mono mt-6 text-accent-600 dark:text-accent-400">Accounts</p>
          <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Almost there</h1>
          <p class="mt-3 text-sm leading-relaxed text-primary-500 dark:text-primary-400">
            Your account has been created but is not active yet. A platform administrator has to approve it — you will be able to sign in once they do.
          </p>
          <router-link to="login" :class="primaryButton" class="mt-9">Back to sign in</router-link>
        </template>

        <!-- signup closed -->
        <template v-else-if="signupEnabled === false">
          <p class="label-mono mt-6 text-accent-600 dark:text-accent-400">Accounts</p>
          <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Invite only</h1>
          <p class="mt-3 text-sm leading-relaxed text-primary-500 dark:text-primary-400">
            Self-signup is disabled. Accounts are created by an administrator — ask your team admin to set one up for you, then sign in.
          </p>
          <router-link to="login" :class="primaryButton" class="mt-9">Back to sign in</router-link>
        </template>

        <!-- signup form -->
        <template v-else>
          <p class="label-mono mt-6 text-accent-600 dark:text-accent-400">Accounts</p>
          <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Create your account</h1>
          <p class="mt-3 text-sm leading-relaxed text-primary-500 dark:text-primary-400">An administrator activates new accounts before the first sign-in.</p>

          <form class="mt-8 space-y-4" @submit.prevent="submit">
            <div v-for="field in fields" :key="field.key">
              <label :for="field.key" class="label-mono block text-primary-500 dark:text-primary-400">{{ field.label }}</label>
              <input :id="field.key" v-model="form[field.key]" :type="field.type ?? 'text'" :autocomplete="field.autocomplete" :class="inputClasses" class="mt-2" />
              <PasswordRequirements v-if="field.key === 'password'" ref="pwReqs" :password="form.password" always-visible />
            </div>

            <p v-if="errorMessage" class="text-sm font-medium text-error-600 dark:text-error-400">{{ errorMessage }}</p>

            <button type="submit" :disabled="!canSubmit || busy" :class="primaryButton" class="!flex h-11 w-full">Create account</button>
          </form>

          <p class="mt-6 text-sm text-primary-500 dark:text-primary-400">
            Already have an account?
            <router-link to="login" class="font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400">Sign in</router-link>
          </p>
        </template>
      </div>
    </main>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import CornerMarks from "src/components/CornerMarks.vue";
import PasswordRequirements from "src/components/PasswordRequirements.vue";
import { ClockIcon, TagIcon, MagnifyingGlassIcon, UserPlusIcon } from "@heroicons/vue/24/outline";
import { api } from "src/api";

const year = new Date().getFullYear();
const features = [
  { icon: ClockIcon, label: "Time-sync every camera to one timeline" },
  { icon: TagIcon, label: "Tag collaboratively as a team" },
  { icon: MagnifyingGlassIcon, label: "Find any frame in seconds" },
];

const inputClasses =
  "block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600";
const primaryButton =
  "inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface active:bg-accent-700 disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950 h-11 cursor-pointer";

// null until /version answers — the form renders optimistically, the "invite
// only" panel only after the server says signup is off.
const signupEnabled = ref<boolean | null>(null);
const submitted = ref(false);
const busy = ref(false);
const errorMessage = ref("");

const form = reactive({ firstName: "", lastName: "", username: "", email: "", password: "" });
type FormKey = keyof typeof form;
const fields: { key: FormKey; label: string; type?: string; autocomplete?: string }[] = [
  { key: "firstName", label: "First name", autocomplete: "given-name" },
  { key: "lastName", label: "Last name", autocomplete: "family-name" },
  { key: "username", label: "Username", autocomplete: "username" },
  { key: "email", label: "Email", type: "email", autocomplete: "email" },
  { key: "password", label: "Password", type: "password", autocomplete: "new-password" },
];

const pwReqs = ref<any>(null);
const canSubmit = computed(() => {
  const reqs = Array.isArray(pwReqs.value) ? pwReqs.value[0] : pwReqs.value;
  return !!form.firstName && !!form.lastName && !!form.username && !!form.email && !!reqs?.allMet;
});

onMounted(async () => {
  try {
    signupEnabled.value = (await api.auth.serverInfo()).signupEnabled;
  } catch {
    signupEnabled.value = null; // server unreachable — let the submit surface it
  }
});

async function submit() {
  errorMessage.value = "";
  busy.value = true;
  try {
    await api.auth.signup({ ...form });
    submitted.value = true;
  } catch (error: any) {
    const status = error?.response?.status;
    if (status === 403) {
      signupEnabled.value = false;
    } else if (status === 429) {
      errorMessage.value = "Too many attempts. Please wait a minute and try again.";
    } else {
      errorMessage.value = error?.response?.data?.message ?? "Could not create the account. Please try again.";
    }
  } finally {
    busy.value = false;
  }
}
</script>
