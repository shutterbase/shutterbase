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

    <!-- info panel: self-signup is disabled (accounts are admin-provisioned) -->
    <main class="flex w-full flex-1 items-center justify-center px-6 py-12 sm:px-12">
      <div class="w-full max-w-sm">
        <router-link to="/" class="mb-12 inline-block lg:hidden">
          <img class="h-9 dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
          <img class="hidden h-9 dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
        </router-link>

        <span class="flex h-11 w-11 items-center justify-center rounded-md border border-primary-200 bg-surface-muted text-accent-600 dark:border-primary-700 dark:bg-primary-900 dark:text-accent-300">
          <UserPlusIcon class="h-5 w-5" />
        </span>
        <p class="label-mono mt-6 text-accent-600 dark:text-accent-400">Accounts</p>
        <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Invite only</h1>
        <p class="mt-3 text-sm leading-relaxed text-primary-500 dark:text-primary-400">
          Self-signup is disabled. Accounts are created by an administrator — ask your team admin to set one up for you, then sign in.
        </p>

        <router-link
          to="login"
          class="mt-9 inline-flex h-11 items-center justify-center gap-1.5 rounded-md bg-accent-600 px-5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface active:bg-accent-700 dark:focus-visible:ring-offset-primary-950"
        >
          Back to sign in
        </router-link>
      </div>
    </main>
  </section>
</template>

<script setup lang="ts">
import CornerMarks from "src/components/CornerMarks.vue";
import { ClockIcon, TagIcon, MagnifyingGlassIcon, UserPlusIcon } from "@heroicons/vue/24/outline";

const year = new Date().getFullYear();
const features = [
  { icon: ClockIcon, label: "Time-sync every camera to one timeline" },
  { icon: TagIcon, label: "Tag collaboratively as a team" },
  { icon: MagnifyingGlassIcon, label: "Find any frame in seconds" },
];
</script>
