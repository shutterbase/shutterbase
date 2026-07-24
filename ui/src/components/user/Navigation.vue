<template>
  <!-- scrollbar-hide: overflow-x-auto promotes overflow-y to auto, and the -bottom-px
       active underline overflows 1px → phantom vertical scrollbar. Hide it; horizontal
       swipe-scroll still works for narrow screens. -->
  <nav class="-mb-px flex items-stretch gap-7 overflow-x-auto border-b border-primary-200 dark:border-primary-800 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
    <a
      v-for="item in navigationItems"
      :key="item.name"
      href="#"
      @click.prevent="() => router.push(item.href)"
      :aria-current="item.current ? 'page' : undefined"
      :class="[
        `group relative flex cursor-pointer items-center gap-2 whitespace-nowrap py-3 text-sm font-medium transition-colors`,
        item.current ? `text-primary-900 dark:text-white` : `text-primary-500 hover:text-primary-900 dark:text-primary-400 dark:hover:text-white`,
      ]"
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
</template>

<script setup lang="ts">
import { UserIcon, CameraIcon, CommandLineIcon } from "@heroicons/vue/24/outline";
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
    { name: "Hotkeys", icon: CommandLineIcon, href: `${BASE_URL}/${itemId}/hotkeys`, current: false },
  ];

  items.forEach((item) => {
    item.current = item.href === route.path;
  });

  navigationItems.value = items;
}

watch(route, updateNavigationItems);
onMounted(updateNavigationItems);
</script>
