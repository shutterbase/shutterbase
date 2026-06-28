<template>
  <div class="bg-primary-50 dark:bg-primary-950 text-primary-900 dark:text-primary-100 min-h-screen flex flex-col">
    <div v-if="userStore.isImpersonating" class="bg-warning-400 text-warning-950 text-sm font-medium px-4 py-2 flex items-center justify-center gap-x-3 z-40">
      <span>
        Impersonating <b>{{ userStore.user?.firstName }} {{ userStore.user?.lastName }}</b>
        &mdash; signed in as {{ userStore.impersonating?.realUserName }}
      </span>
      <button @click="stopImpersonate" class="rounded-md bg-warning-950 text-warning-50 px-3 py-1 text-xs font-semibold hover:bg-warning-900 transition-colors">Stop impersonating</button>
    </div>
    <Disclosure as="header" class="sticky top-0 z-30 border-b border-primary-200 dark:border-primary-800 bg-primary-50/90 dark:bg-primary-950/90 backdrop-blur" v-slot="{ open }">
      <div class="mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8">
        <div class="flex h-16 items-center justify-between gap-4">
          <div class="flex items-center gap-5 self-stretch">
            <router-link to="/" class="flex flex-shrink-0 items-center">
              <img class="h-7 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
              <img class="h-7 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
            </router-link>
            <div class="hidden sm:block h-6 w-px bg-primary-200 dark:bg-primary-800"></div>
            <nav class="hidden sm:flex items-stretch self-stretch gap-7">
              <span
                v-for="item in navigation"
                :key="item.name"
                @click="router.push(item.href)"
                :class="[
                  'group relative flex cursor-pointer items-center text-sm font-medium transition-colors',
                  item.current ? 'text-primary-900 dark:text-white' : 'text-primary-500 hover:text-primary-900 dark:text-primary-400 dark:hover:text-white',
                ]"
                :aria-current="item.current ? 'page' : undefined"
              >
                {{ item.name }}
                <span
                  :class="[
                    'absolute inset-x-0 -bottom-px h-0.5 transition-all',
                    item.current ? 'bg-accent-500' : 'bg-transparent group-hover:bg-primary-300 dark:group-hover:bg-primary-700',
                  ]"
                ></span>
              </span>
            </nav>
          </div>
          <div class="flex items-center gap-3">
            <div v-if="userStore.activeProject?.name" class="hidden md:flex items-center gap-2 pr-1">
              <span class="label-mono-sm text-primary-500 dark:text-primary-400">Project</span>
              <span class="max-w-[14rem] truncate text-sm font-medium text-primary-700 dark:text-primary-200">{{ userStore.activeProject.name }}</span>
            </div>
            <div class="hidden md:block h-6 w-px bg-primary-200 dark:bg-primary-800"></div>
            <DarkMode />
            <UserMenu />
            <DisclosureButton
              class="sm:hidden inline-flex items-center justify-center rounded-md p-2 text-primary-500 hover:bg-primary-100 dark:hover:bg-primary-800 hover:text-primary-900 dark:hover:text-white"
            >
              <span class="sr-only">Open main menu</span>
              <Bars3Icon v-if="!open" class="block h-5 w-5" aria-hidden="true" />
              <XMarkIcon v-else class="block h-5 w-5" aria-hidden="true" />
            </DisclosureButton>
          </div>
        </div>
      </div>

      <DisclosurePanel class="sm:hidden border-t border-primary-200 dark:border-primary-800">
        <div class="space-y-1 px-3 py-3">
          <DisclosureButton
            v-for="item in navigation"
            :key="item.name"
            as="a"
            href="#"
            @click="router.push(item.href)"
            :class="[
              item.current
                ? 'bg-accent-500/12 text-accent-700 dark:text-accent-200'
                : 'text-primary-600 dark:text-primary-300 hover:bg-primary-100 dark:hover:bg-primary-800',
              'block rounded-md px-3 py-2 text-base font-medium',
            ]"
            :aria-current="item.current ? 'page' : undefined"
            >{{ item.name }}</DisclosureButton
          >
        </div>
      </DisclosurePanel>
    </Disclosure>

    <main class="flex-1 w-full py-8">
      <router-view />
    </main>
    <Notification />
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
import Notification from "src/components/layout/Notification.vue";
import { store } from "quasar/wrappers";
import { storeToRefs } from "pinia";
const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const router = useRouter();
const route = useRoute();

type NavigationItem = { name: string; href: string; current: boolean };
const navigation: Ref<NavigationItem[]> = ref([]);

async function stopImpersonate() {
  await userStore.stopImpersonate();
  router.go(0); // reload to refresh effective-user-scoped data
}

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
