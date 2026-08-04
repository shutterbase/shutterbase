<template>
  <main class="mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
    <p class="label-mono text-accent-600 dark:text-accent-400">Welcome back</p>
    <h1 class="display mt-3 text-3xl text-primary-900 dark:text-white sm:text-5xl">Your shared photo library.</h1>
    <p class="mt-5 max-w-xl text-base leading-7 text-primary-600 dark:text-primary-300">
      Jump back into a project, browse the latest frames, or set up a camera to start syncing your timeline with the team.
    </p>

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

    <!-- My schedule (S15): the photographer's next covered items, straight off
         the dashboard (transcript 12:44). Only exists with an active project. -->
    <section v-if="activeProjectId && upcoming.length" class="mt-16">
      <p class="label-mono text-accent-600 dark:text-accent-400">My schedule</p>
      <h2 class="display mt-2 text-2xl text-primary-900 dark:text-white">Up next for you.</h2>
      <ul
        class="mt-6 max-w-3xl divide-y divide-primary-100 rounded-lg border border-primary-200 bg-surface shadow-panel dark:divide-primary-800 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark"
      >
        <li v-for="entry in upcoming" :key="entry.id">
          <router-link :to="{ name: 'schedule' }" class="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-primary-50 dark:hover:bg-primary-900/40">
            <span :class="['h-2.5 w-2.5 flex-shrink-0 rounded-full border', statusDot(entry)]"></span>
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium text-primary-900 dark:text-white">{{ entry.title }}</span>
              <span class="block text-xs tabular-nums text-primary-500 dark:text-primary-400">{{ formatWindow(entry) }}</span>
            </span>
            <span class="text-xs text-primary-400">{{ entry.assignees.length }}/{{ entry.cardinality }}</span>
          </router-link>
        </li>
      </ul>
      <router-link :to="{ name: 'schedule' }" class="mt-3 inline-block text-sm font-medium text-accent-600 underline-offset-2 hover:underline dark:text-accent-400">
        Open the full schedule →
      </router-link>
    </section>

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
      <p v-if="!activeProjectId && isProjectAdmin" class="mt-4 text-sm text-primary-500 dark:text-primary-400">Select a project to reach its members, tags and review board.</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, Ref } from "vue";
import { DateTime } from "luxon";
import { storeToRefs } from "pinia";
import { api } from "src/api";
import { ScheduleItem } from "src/types/api";
import { occupancyStatus } from "src/util/schedule";
import { useUserStore } from "src/stores/user-store";
import { CameraIcon, FolderIcon, PhotoIcon, UsersIcon, UserGroupIcon, TagIcon, Cog6ToothIcon, RectangleStackIcon } from "@heroicons/vue/24/solid";

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

// Next 5 items the user covers, from now on. Quietly absent on error — the
// dashboard must never block on a widget.
const upcoming: Ref<ScheduleItem[]> = ref([]);
onMounted(async () => {
  if (!activeProjectId.value) return;
  try {
    const result = await api.scheduleItems.list({
      projectId: activeProjectId.value,
      mine: true,
      from: new Date().toISOString(),
      sort: "start",
      order: "asc",
      limit: 5,
    });
    upcoming.value = result.items;
  } catch {
    upcoming.value = [];
  }
});

const statusDots: Record<string, string> = {
  empty: "border-blue-400 bg-blue-500/40",
  partial: "border-yellow-400 bg-yellow-400/50",
  full: "border-green-500 bg-green-500/40",
  over: "border-violet-500 bg-violet-500/50",
};
const statusDot = (item: ScheduleItem) => statusDots[occupancyStatus(item.assignees.length, item.cardinality)];
const formatWindow = (item: ScheduleItem) => `${DateTime.fromISO(item.start).toFormat("ccc dd.LL. HH:mm")} – ${DateTime.fromISO(item.end).toFormat("HH:mm")}`;

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
