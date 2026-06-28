<template>
  <Menu as="div" class="relative ml-3">
    <div>
      <MenuButton class="relative flex rounded-full bg-primary-200 text-sm transition focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:bg-primary-800 dark:focus-visible:ring-offset-primary-950">
        <span class="absolute -inset-1.5" />
        <span class="sr-only">Open user menu</span>
        <img id="user-menu-avatar" ref="userMenuAvatar" class="h-8 w-8 rounded-full ring-1 ring-primary-300 dark:ring-primary-700" src="" alt="" />
      </MenuButton>
    </div>
    <transition
      enter-active-class="transition ease-out duration-100"
      enter-from-class="transform opacity-0 scale-95"
      enter-to-class="transform opacity-100 scale-100"
      leave-active-class="transition ease-in duration-75"
      leave-from-class="transform opacity-100 scale-100"
      leave-to-class="transform opacity-0 scale-95"
    >
      <MenuItems
        :class="[
          'absolute right-0 z-20 mt-2 w-56 origin-top-right overflow-hidden rounded-lg divide-y py-1 focus:outline-none',
          'border border-primary-200 bg-surface shadow-panel divide-primary-200 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark dark:divide-primary-800',
        ]"
      >
        <div class="py-3 px-4">
          <span class="block text-sm font-semibold text-primary-900 dark:text-white">{{ firstName }} {{ lastName }}</span>
          <span class="block text-sm text-primary-500 truncate dark:text-primary-400">{{ userStore.user?.role?.description || userStore.user?.role?.key }}</span>
        </div>
        <ul class="py-1 text-primary-700 dark:text-primary-200">
          <li>
            <router-link
              :to="`/users/${userStore.user?.id || ''}/general`"
              class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white"
            >
              <UserIcon class="mr-2 w-5 h-5 text-primary-400 dark:text-primary-500" />
              My profile
            </router-link>
          </li>
        </ul>
        <ul class="py-1 text-primary-700 dark:text-primary-200">
          <li>
            <router-link
              :to="`/users/${userStore.user?.id || ''}/cameras`"
              class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white"
            >
              <CameraIcon class="mr-2 w-5 h-5 text-primary-400 dark:text-primary-500" />
              My Cameras
            </router-link>
          </li>
          <li>
            <a href="#" class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white">
              <CloudArrowUpIcon class="mr-2 w-5 h-5 text-primary-400 dark:text-primary-500" />
              My Uploads
            </a>
          </li>
        </ul>
        <ul class="py-1 text-primary-700 dark:text-primary-200">
          <li v-if="activeProjectId">
            <router-link :to="`/projects/${activeProjectId}/general`" class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white">
              Project: <span class="ml-2 font-semibold text-accent-600 dark:text-accent-400">{{ userStore.activeProject?.name }}</span>
            </router-link>
          </li>
          <li v-if="activeProjectId">
            <a href="#" @click.prevent="clearProjectSelection" class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white">
              Clear project selection</a
            >
          </li>
          <li v-else>
            <router-link
              :to="`/projects`"
              class="flex cursor-pointer items-center py-2 px-4 text-sm transition-colors hover:bg-primary-100 hover:text-primary-900 dark:hover:bg-primary-800 dark:hover:text-white"
            >
              Select a project
            </router-link>
          </li>
        </ul>
        <ul class="py-1">
          <li>
            <a href="/logout" class="block cursor-pointer py-2 px-4 text-sm font-medium text-error-700 transition-colors hover:bg-error-50 dark:text-error-300 dark:hover:bg-error-950/40">Sign out</a>
          </li>
        </ul>
      </MenuItems>
    </transition>
  </Menu>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItems } from "@headlessui/vue";
import { CameraIcon, CloudArrowUpIcon, UserIcon } from "@heroicons/vue/24/solid";
import { onMounted, ref } from "vue";
import { initFlowbite } from "flowbite";
import { useUserStore } from "src/stores/user-store";
import Avatar from "avatar-initials";
import { storeToRefs } from "pinia";

const userMenuAvatar = ref<HTMLImageElement>();

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const firstName = ref(userStore.user?.firstName);
const lastName = ref(userStore.user?.lastName);
const headshotUrl = ref("");

function clearProjectSelection() {
  userStore.clearActiveProject();
}

onMounted(() => {
  initFlowbite();
  if (!userMenuAvatar.value) return;
  if (headshotUrl.value) {
    userMenuAvatar.value.src = headshotUrl.value;
    return;
  }
  const avatar = Avatar.from(userMenuAvatar.value, {
    useGravatar: true,
    email: userStore.user?.email,
    initials: `${firstName.value?.charAt(0) ?? ""}${lastName.value?.charAt(0) ?? ""}`,
    color: "#FFFFFF",
    background: "#37465D",
    fontWeight: 400,
  });
});
</script>
