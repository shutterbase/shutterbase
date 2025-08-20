<template>
  <aside class="flex overflow-x-auto border-b border-gray-900/5 py-4 lg:block lg:w-64 lg:flex-none lg:border-0">
    <nav class="flex-none px-4 sm:px-6 lg:px-0">
      <ul role="list" class="flex gap-x-3 gap-y-1 whitespace-nowrap lg:flex-col">
        <li v-for="item in navigationItems" :key="item.name">
          <a
            href="#"
            @click.prevent="() => router.push(item.href)"
            :class="[
              `group flex gap-x-3 rounded-md py-2 pl-2 pr-3 text-sm leading-6 font-semibold`,
              item.current ? `bg-primary-900 text-white dark:bg-primary-900` : `text-primary-900 hover:bg-primary-200 dark:text-primary-200 dark:hover:bg-primary-700`,
            ]"
          >
            <component :is="item.icon" :class="[`h-6 w-6 shrink-0`, item.current ? `text-white` : `text-primary-600 dark:text-white`]" />
            {{ item.name }}
          </a>
        </li>
      </ul>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { UserIcon, CameraIcon, KeyIcon } from "@heroicons/vue/24/outline";
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const BASE_URL = "/users";

type NavigationItem = {
  name: string;
  icon: any;
  href: string;
  current: boolean;
};
const navigationItems: Ref<NavigationItem[]> = ref([]);

function updateNavigationItems() {
  const itemId = route.params.userid;
  const items = [
    { name: "General", icon: UserIcon, href: `${BASE_URL}/${itemId}/general`, current: false },
    { name: "Cameras", icon: CameraIcon, href: `${BASE_URL}/${itemId}/cameras`, current: false },
    { name: "Hotkeys", icon: KeyIcon, href: `${BASE_URL}/${itemId}/hotkeys`, current: false },
  ];

  items.forEach((item) => {
    item.current = item.href === route.path;
  });

  navigationItems.value = items;
}

watch(route, updateNavigationItems);
onMounted(updateNavigationItems);
</script>
