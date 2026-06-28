<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <nav class="flex gap-6 overflow-x-auto border-b border-primary-200 dark:border-primary-800" aria-label="Project sections">
      <a
        v-for="item in navigationItems"
        :key="item.name"
        href="#"
        @click.prevent="() => router.push(item.href)"
        :class="[
          `group relative flex cursor-pointer items-center gap-2 whitespace-nowrap py-3.5 text-sm font-medium transition-colors`,
          item.current ? `text-primary-900 dark:text-white` : `text-primary-500 hover:text-primary-900 dark:text-primary-400 dark:hover:text-white`,
        ]"
        :aria-current="item.current ? 'page' : undefined"
      >
        <component :is="item.icon" class="h-[18px] w-[18px] shrink-0" />
        {{ item.name }}
        <span
          :class="[
            `absolute inset-x-0 -bottom-px h-0.5 transition-all`,
            item.current ? `bg-accent-500` : `bg-transparent group-hover:bg-primary-300 dark:group-hover:bg-primary-700`,
          ]"
        ></span>
      </a>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ExclamationTriangleIcon, PresentationChartLineIcon, RectangleStackIcon, TagIcon, UserGroupIcon } from "@heroicons/vue/24/outline";
import { useUserStore } from "src/stores/user-store";
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

const userStore = useUserStore();

const route = useRoute();
const router = useRouter();

const BASE_URL = "/projects";

type NavigationItem = {
  name: string;
  icon: any;
  href: string;
  current: boolean;
};
const navigationItems: Ref<NavigationItem[]> = ref([]);

function updateNavigationItems() {
  const itemId = route.params.id;
  const items = [
    { name: "General", icon: PresentationChartLineIcon, href: `${BASE_URL}/${itemId}/general`, current: false },
    { name: "Tags", icon: TagIcon, href: `${BASE_URL}/${itemId}/tags`, current: false },
    { name: "Statistics", icon: RectangleStackIcon, href: `${BASE_URL}/${itemId}/statistics`, current: false },
  ];

  if (userStore.isProjectAdminOrHigher()) {
    items.push({ name: "Members", icon: UserGroupIcon, href: `${BASE_URL}/${itemId}/members`, current: false });
  }

  if (userStore.isAdmin()) {
    items.push({ name: "Danger Zone", icon: ExclamationTriangleIcon, href: `${BASE_URL}/${itemId}/danger-zone`, current: false });
  }

  items.forEach((item) => {
    item.current = item.href === route.path;
  });

  navigationItems.value = items;
}

watch(route, updateNavigationItems);
onMounted(updateNavigationItems);
</script>
