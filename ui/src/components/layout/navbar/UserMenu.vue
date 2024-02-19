<template>
  <Menu as="div" class="relative ml-3">
    <div>
      <MenuButton class="relative flex rounded-full bg-gray-800 text-sm focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-gray-800">
        <span class="absolute -inset-1.5" />
        <span class="sr-only">Open user menu</span>
        <img
          class="h-8 w-8 rounded-full"
          src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
          alt=""
        />
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
        <ul v-if="userStore.activeProjectId !== ''" class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <a href="#" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              Project: <span class="text-bold ml-2">{{ userStore.activeProject?.name }}</span>
            </a>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <a href="#" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              <UserIcon class="mr-2 w-5 h-5 text-gray-400" />
              My profile
            </a>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <a href="#" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              <CameraIcon class="mr-2 w-5 h-5 text-gray-400" />
              My Cameras
            </a>
          </li>
          <li>
            <a href="#" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              <CloudArrowUpIcon class="mr-2 w-5 h-5 text-gray-400" />
              My Uploads
            </a>
          </li>
        </ul>
        <ul class="py-1 text-gray-700 dark:text-gray-300">
          <li>
            <a href="#" @click.prevent="clearProjectSelection" class="flex items-center py-2 px-4 text-sm hover:bg-gray-100 dark:hover:bg-gray-600 dark:hover:text-white">
              Clear project selection</a
            >
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
import { CameraIcon, CloudArrowUpIcon, UserIcon } from "@heroicons/vue/24/solid";
import { onMounted, ref } from "vue";
import { initFlowbite } from "flowbite";
import pb from "src/boot/pocketbase";
import { useUserStore } from "src/stores/user-store";

const firstName = ref(pb.authStore.model?.firstName);
const lastName = ref(pb.authStore.model?.lastName);
const headshotUrl = ref("");

const userStore = useUserStore();
function clearProjectSelection() {
  userStore.clearActiveProject();
}

onMounted(() => {
  initFlowbite();
});
</script>
