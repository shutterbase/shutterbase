<template>
  <div class="navbar bg-base-100">
    <div class="navbar-start">
      <div class="dropdown">
        <label tabindex="0" class="btn btn-ghost btn-circle">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
          </svg>
        </label>
        <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
          <li><a>Homepage</a></li>
        </ul>
      </div>
      <ActiveProjectHeader />
    </div>
    <div class="navbar-center">
      <div class="btn btn-ghost normal-case text-xl" @click="doHeaderButtonNavigation">shutterbase</div>
    </div>
    <div class="navbar-end">
      <NuxtLink to="/logout"><button class="btn">Logout</button></NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useStore } from "~/stores/store";
const store = useStore();

function doHeaderButtonNavigation() {
  if (store.isLoggedIn()) {
    navigateTo("/dashboard");
  } else {
    navigateTo("/login");
  }
}
</script>
