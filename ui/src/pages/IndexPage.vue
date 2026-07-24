<template>
  <main class="mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
    <p class="label-mono text-accent-600 dark:text-accent-400">Welcome back</p>
    <h1 class="display mt-3 text-3xl text-primary-900 dark:text-white sm:text-5xl">Your shared photo library.</h1>
    <p class="mt-5 max-w-xl text-base leading-7 text-primary-600 dark:text-primary-300">Jump back into a project, browse the latest frames, or set up a camera to start syncing your timeline with the team.</p>

    <div class="mt-10 grid gap-4 sm:grid-cols-3">
      <router-link :to="{ name: 'projects' }" :class="cardClasses">
        <FolderIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">Projects</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Open a shared project and its library.</span>
      </router-link>

      <router-link :to="{ name: 'images' }" :class="cardClasses">
        <PhotoIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">Images</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Browse, search and tag the latest frames.</span>
      </router-link>

      <router-link :to="`/users/${userStore.user?.id || ''}/cameras`" :class="cardClasses">
        <CameraIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">My Cameras</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Add a camera and set its time offset.</span>
      </router-link>
    </div>

    <!-- Administration. Each entry is gated on the role that can actually use it,
         so the section simply does not exist for a plain photographer. -->
    <section v-if="adminItems.length" class="mt-16">
      <p class="label-mono text-accent-600 dark:text-accent-400">Administration</p>
      <h2 class="display mt-2 text-2xl text-primary-900 dark:text-white">Run the shoot.</h2>
      <div class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <router-link v-for="entry in adminItems" :key="entry.label" :to="entry.to" :class="cardClasses">
          <component :is="entry.icon" class="h-6 w-6 text-accent-600 dark:text-accent-400" />
          <span class="display text-lg text-primary-900 dark:text-white">{{ entry.label }}</span>
          <span class="text-sm text-primary-500 dark:text-primary-400">{{ entry.hint }}</span>
        </router-link>
      </div>
      <p v-if="!activeProjectId && isProjectAdmin" class="mt-4 text-sm text-primary-500 dark:text-primary-400">
        Select a project to reach its members, tags and review board.
      </p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { CameraIcon, FolderIcon, PhotoIcon, UsersIcon, UserGroupIcon, TagIcon, Cog6ToothIcon, RectangleStackIcon } from "@heroicons/vue/24/solid";

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const cardClasses =
  "group flex cursor-pointer flex-col gap-3 rounded-lg border border-primary-200 bg-surface p-6 shadow-panel transition-colors hover:border-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark dark:hover:border-accent-500";

const isProjectAdmin = computed(() => userStore.isProjectAdminOrHigher());

const adminItems = computed(() => {
  const items: { label: string; hint: string; to: string; icon: any }[] = [];
  if (userStore.isAdmin()) {
    items.push({ label: "Users", hint: "Create accounts, activate signups, reset passwords.", to: "/users", icon: UsersIcon });
  }
  if (isProjectAdmin.value && activeProjectId.value) {
    const base = `/projects/${activeProjectId.value}`;
    items.push(
      { label: "Members", hint: "Assign photographers and roles to this project.", to: `${base}/members`, icon: UserGroupIcon },
      { label: "Tags", hint: "Manage the project's tag vocabulary.", to: `${base}/tags`, icon: TagIcon },
      { label: "Project settings", hint: "Copyright, AI options and the upload review flow.", to: `${base}/general`, icon: Cog6ToothIcon },
      { label: "Upload review", hint: "Work the review board across all photographers.", to: "/uploads", icon: RectangleStackIcon },
    );
  }
  return items;
});
</script>
