<template>
  <Menu as="div" class="relative ml-3">
    <div>
      <MenuButton class="relative flex rounded-full bg-gray-800 text-sm focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-gray-800">
        <span class="absolute -inset-1.5" />
        <span class="sr-only">Open user menu</span>
        <img id="user-menu-avatar" ref="userMenuAvatar" class="h-8 w-8 rounded-full" src="" alt="" />
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
          'absolute right-0 z-20 mt-2 w-48 origin-top-right rounded-md divide-y py-1 shadow-lg ring-1 ring-opacity-5 focus:outline-none',
          'bg-white divide-gray-100 dark:!bg-gray-700 dark:!divide-gray-600 ring-black',
        ]"
      >
        <div class="py-3 px-4">
          <span class="block text-sm font-semibold text-gray-900 dark:text-white">{{ firstName }} {{ lastName }}</span>
          <span class="block text-sm text-gray-900 truncate dark:text-white">INSERT ROLE HERE</span>
        </div>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <router-link
              :to="`/users/${pb.authStore.model?.id || ''}/general`"
              class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white"
            >
              <UserIcon class="mr-2 w-5 h-5 text-gray-400" />
              My profile
            </router-link>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <router-link
              :to="`/users/${pb.authStore.model?.id || ''}/cameras`"
              class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white"
            >
              <CameraIcon class="mr-2 w-5 h-5 text-gray-400" />
              My Cameras
            </router-link>
          </li>
          <li>
            <a href="#" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              <CloudArrowUpIcon class="mr-2 w-5 h-5 text-gray-400" />
              My Uploads
            </a>
          </li>
          <li>
            <router-link
              :to="`/users/${pb.authStore.model?.id || ''}/hotkeys`"
              class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white"
            >
              <KeyIcon class="mr-2 w-5 h-5 text-gray-400" />
              My Hotkeys
            </router-link>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li v-if="activeProjectId">
            <router-link :to="`/projects/${activeProjectId}/general`" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              Project: <span class="text-bold ml-2">{{ userStore.activeProject?.name }}</span>
            </router-link>
          </li>
          <li v-if="activeProjectId">
            <a href="#" @click.prevent="clearProjectSelection" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              Clear project selection</a
            >
          </li>
          <li v-else>
            <router-link :to="`/projects`" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              Select a project
            </router-link>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <a href="/logout" class="block py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">Sign out</a>
          </li>
        </ul>
      </MenuItems>
    </transition>
  </Menu>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItems } from "@headlessui/vue";
import { CameraIcon, CloudArrowUpIcon, UserIcon, KeyIcon } from "@heroicons/vue/24/solid";
import { onMounted, ref } from "vue";
import { initFlowbite } from "flowbite";
import pb from "src/boot/pocketbase";
import { useUserStore } from "src/stores/user-store";
import Avatar from "avatar-initials";
import { storeToRefs } from "pinia";

const userMenuAvatar = ref<HTMLImageElement>();

const firstName = ref(pb.authStore.model?.firstName);
const lastName = ref(pb.authStore.model?.lastName);
const headshotUrl = ref("");

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

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
    email: pb.authStore.model?.email,
    initials: `${firstName.value?.charAt(0) ?? ""}${lastName.value?.charAt(0) ?? ""}`,
    color: "#FFFFFF",
    background: "#37465D",
    fontWeight: 400,
  });
});
</script>
