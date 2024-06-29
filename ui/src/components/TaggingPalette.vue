<template>
  <div class="relative z-10" role="dialog" aria-modal="true">
    <!--
    Background backdrop, show/hide based on modal state.

    Entering: "ease-out duration-300"
      From: "opacity-0"
      To: "opacity-100"
    Leaving: "ease-in duration-200"
      From: "opacity-100"
      To: "opacity-0"
  -->
    <div v-show="shown" class="fixed inset-0 bg-gray-500 bg-opacity-25 transition-opacity"></div>

    <div class="fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-20">
      <!--
      Command palette, show/hide based on modal state.

      Entering: "ease-out duration-300"
        From: "opacity-0 scale-95"
        To: "opacity-100 scale-100"
      Leaving: "ease-in duration-200"
        From: "opacity-100 scale-100"
        To: "opacity-0 scale-95"
    -->
      <div
        v-show="shown"
        class="mx-auto max-w-3xl transform divide-y divide-gray-100 overflow-hidden rounded-xl bg-white shadow-2xl ring-1 ring-black ring-opacity-5 transition-all"
      >
        <div class="relative">
          <svg class="pointer-events-none absolute left-4 top-3.5 h-5 w-5 text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path
              fill-rule="evenodd"
              d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
              clip-rule="evenodd"
            />
          </svg>
          <input
            type="text"
            class="h-12 w-full border-0 bg-transparent pl-11 pr-4 text-gray-800 placeholder:text-gray-400 focus:ring-0 sm:text-sm"
            placeholder="Search..."
            role="combobox"
            aria-expanded="false"
            aria-controls="options"
          />
        </div>

        <div class="flex transform-gpu divide-x divide-gray-100">
          <!-- Preview Visible: "sm:h-96" -->
          <div class="max-h-96 min-w-0 flex-auto scroll-py-4 overflow-y-auto px-6 py-4 sm:h-96">
            <!-- Default state, show/hide based on command palette state. -->
            <h2 class="mb-4 mt-2 text-xs font-semibold text-gray-500">Recent searches</h2>
            <ul class="-mx-2 text-sm text-gray-700" id="recent" role="listbox">
              <!-- Active: "bg-gray-100 text-gray-900" -->
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="recent-1" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1463453091185-61582044d556?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Floyd Miles</span>
                <!-- Not Active: "hidden" -->
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
            </ul>

            <!-- Results, show/hide based on command palette state. -->
            <ul class="-mx-2 text-sm text-gray-700" id="options" role="listbox">
              <!-- Active: "bg-gray-100 text-gray-900" -->
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="option-1" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Tom Cook</span>
                <!-- Not Active: "hidden" -->
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="option-2" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1438761681033-6461ffad8d80?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Courtney Henry</span>
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="option-3" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Dries Vincent</span>
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="option-4" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1500917293891-ef795e70e1f6?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Kristin Watson</span>
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
              <li class="group flex cursor-default select-none items-center rounded-md p-2" id="option-5" role="option" tabindex="-1">
                <img
                  src="https://images.unsplash.com/photo-1517070208541-6ddc4d3efbcb?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                  alt=""
                  class="h-6 w-6 flex-none rounded-full"
                />
                <span class="ml-3 flex-auto truncate">Jeffrey Webb</span>
                <svg class="ml-3 hidden h-5 w-5 flex-none text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path
                    fill-rule="evenodd"
                    d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
                    clip-rule="evenodd"
                  />
                </svg>
              </li>
            </ul>
          </div>

          <!-- Active item side-panel, show/hide based on active state -->
          <div class="h-96 w-1/2 flex-none flex-col divide-y divide-gray-100 overflow-y-auto sm:flex">
            <div class="flex-none p-6 text-center">
              <img
                src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
                alt=""
                class="mx-auto h-16 w-16 rounded-full"
              />
              <h2 class="mt-3 font-semibold text-gray-900">Tom Cook</h2>
              <p class="text-sm leading-6 text-gray-500">Director, Product Development</p>
            </div>
            <div class="flex flex-auto flex-col justify-between p-6">
              <dl class="grid grid-cols-1 gap-x-6 gap-y-3 text-sm text-gray-700">
                <dt class="col-end-1 font-semibold text-gray-900">Phone</dt>
                <dd>881-460-8515</dd>
                <dt class="col-end-1 font-semibold text-gray-900">URL</dt>
                <dd class="truncate"><a href="https://example.com" class="text-indigo-600 underline">https://example.com</a></dd>
                <dt class="col-end-1 font-semibold text-gray-900">Email</dt>
                <dd class="truncate"><a href="#" class="text-indigo-600 underline">tomcook@example.com</a></dd>
              </dl>
              <button
                type="button"
                class="mt-6 w-full rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
              >
                Send message
              </button>
            </div>
          </div>
        </div>

        <!-- Empty state, show/hide based on command palette state -->
        <div class="px-6 py-14 text-center text-sm sm:px-14">
          <svg class="mx-auto h-6 w-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z"
            />
          </svg>
          <p class="mt-4 font-semibold text-gray-900">No people found</p>
          <p class="mt-2 text-gray-500">We couldn’t find anything with that term. Please try again.</p>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref } from "vue";

interface Props {
  shown: boolean;
}

const props = withDefaults(defineProps<Props>(), {});
</script>
