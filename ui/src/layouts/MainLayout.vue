<template>
  <div class="bg-gray-50 dark:bg-primary-800 min-h-screen">
    <Disclosure as="nav" class="bg-gray-50 dark:bg-primary-900" v-slot="{ open }">
      <div class="mx-auto max-w-7xl px-2 sm:px-6 lg:px-8">
        <div class="relative flex h-16 items-center justify-between">
          <div class="absolute inset-y-0 left-0 flex items-center sm:hidden">
            <!-- Mobile menu button-->
            <DisclosureButton
              class="relative inline-flex items-center justify-center rounded-md p-2 text-gray-400 hover:bg-gray-700 hover:text-white focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
            >
              <span class="absolute -inset-0.5" />
              <span class="sr-only">Open main menu</span>
              <Bars3Icon v-if="!open" class="block h-6 w-6" aria-hidden="true" />
              <XMarkIcon v-else class="block h-6 w-6" aria-hidden="true" />
            </DisclosureButton>
          </div>
          <div class="flex flex-1 items-center justify-center sm:items-stretch sm:justify-start">
            <router-link to="/">
              <div class="flex flex-shrink-0 items-center">
                <img class="h-8 mr-2 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="logo" />
                <img class="h-8 mr-2 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="logo" />
              </div>
            </router-link>
            <div class="hidden sm:ml-6 sm:!block">
              <div class="flex space-x-4">
                <span
                  v-for="item in navigation"
                  :key="item.name"
                  @click="router.push(item.href)"
                  :class="[
                    item.current ? 'bg-primary-900 text-white dark:bg-primary-950' : 'text-primary-900 hover:bg-primary-200 dark:text-primary-200 dark:hover:bg-primary-800',
                    'rounded-md cursor-pointer px-3 py-2 text-sm font-medium',
                  ]"
                  :aria-current="item.current ? 'page' : undefined"
                >
                  {{ item.name }}
                </span>
              </div>
            </div>
          </div>
          <div class="absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
            <DarkMode />
            <UserMenu />
          </div>
        </div>
      </div>

      <DisclosurePanel class="sm:hidden">
        <div class="space-y-1 px-2 pb-3 pt-2">
          <DisclosureButton
            v-for="item in navigation"
            :key="item.name"
            as="a"
            href="#"
            @click="router.push(item.href)"
            :class="[item.current ? 'bg-gray-900 text-white' : 'text-gray-300 hover:bg-gray-700 hover:text-white', 'block rounded-md px-3 py-2 text-base font-medium']"
            :aria-current="item.current ? 'page' : undefined"
            >{{ item.name }}</DisclosureButton
          >
        </div>
      </DisclosurePanel>
    </Disclosure>
    <hr class="mb-2 sm:mb-12 dark:border-primary-400" />
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { Disclosure, DisclosureButton, DisclosurePanel } from "@headlessui/vue";
import { Bars3Icon, XMarkIcon } from "@heroicons/vue/24/outline";
import DarkMode from "src/components/layout/navbar/DarkMode.vue";
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useUserStore } from "src/stores/user-store";
import UserMenu from "src/components/layout/navbar/UserMenu.vue";
import { store } from "quasar/wrappers";
import { storeToRefs } from "pinia";
const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const router = useRouter();
const route = useRoute();

type NavigationItem = { name: string; href: string; current: boolean };
const navigation: Ref<NavigationItem[]> = ref([]);

function calculateNavigationItems() {
  const navigationItems = [] as NavigationItem[];
  if (activeProjectId.value && activeProjectId.value !== "") {
    navigationItems.push({ name: "Images", href: "/images", current: false });
    navigationItems.push({ name: "Uploads", href: "/uploads", current: false });
  }
  navigationItems.push({ name: "Projects", href: "/projects", current: false });

  const currentPath = route.path;
  navigationItems.forEach((item) => {
    item.current = currentPath.replace("/", "").startsWith(item.href.replace("/", ""));
  });

  navigation.value = navigationItems;
}

onMounted(calculateNavigationItems);
watch([route, activeProjectId], calculateNavigationItems);
</script>
