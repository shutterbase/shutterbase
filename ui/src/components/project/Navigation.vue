<template>
  <aside class="flex overflow-x-auto border-b border-gray-900/5 py-4 lg:block lg:w-64 lg:flex-none lg:border-0">
    <nav class="flex-none px-4 sm:px-6 lg:px-0">
      <ul role="list" class="flex gap-x-3 gap-y-1 whitespace-nowrap lg:flex-col">
        <li>
          <!-- Current: "bg-gray-50 text-primary-600", Default: "text-gray-700 hover:text-primary-600 hover:bg-gray-50" -->
          <a href="#" class="bg-gray-50 text-primary-600 group flex gap-x-3 rounded-md py-2 pl-2 pr-3 text-sm leading-6 font-semibold">
            <RectangleStackIcon class="h-6 w-6 shrink-0 text-primary-600" />
            General
          </a>
        </li>
        <li>
          <a href="#" class="text-gray-700 hover:text-primary-600 hover:bg-gray-50 group flex gap-x-3 rounded-md py-2 pl-2 pr-3 text-sm leading-6 font-semibold">
            <TagIcon class="h-6 w-6 shrink-0 text-primary-600" />
            Tags
          </a>
        </li>
        <li>
          <a href="#" class="text-gray-700 hover:text-primary-600 hover:bg-gray-50 group flex gap-x-3 rounded-md py-2 pl-2 pr-3 text-sm leading-6 font-semibold">
            <PresentationChartLineIcon class="h-6 w-6 shrink-0 text-primary-600" />
            Statistics
          </a>
        </li>
      </ul>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { PresentationChartLineIcon, RectangleStackIcon, TagIcon } from "@heroicons/vue/24/outline";
</script>
