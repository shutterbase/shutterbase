<template>
  <div class="hero min-h-screen bg-base-200">
    <div class="hero-content flex-col lg:flex-row-reverse">
      <div class="text-center lg:text-left ml-10">
        <img src="/img/shutterbase-logo-wide-xl.png" alt="shutterbase logo" />
      </div>
      <div class="card flex-shrink-0 w-full max-w-sm shadow-2xl bg-base-100">
        <div class="card-body">
          <!--<h1>App health: {{ health }}</h1>-->
          <div class="form-control">
            <label class="label">
              <span class="label-text">Email</span>
            </label>
            <input v-model="username" type="text" placeholder="email" class="input input-bordered" />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">Password</span>
            </label>
            <input v-model="password" type="password" placeholder="password" class="input input-bordered" />
            <label class="label">
              <NuxtLink to="forgot-password" class="label-text-alt link link-hover">Forgot password?</NuxtLink>
            </label>
          </div>
          <div class="form-control">
            <label class="cursor-pointer label">
              <span class="label-text text-left">Remember me</span>
              <input type="checkbox" v-model="rememberMe" class="checkbox checkbox-success" />
            </label>
          </div>
          <div class="form-control mt-6">
            <button class="btn btn-secondary" @click="doLogin">Login</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Method, getFetchOptions } from "~/api/common";
import { ref } from "vue";
import { useStore } from "~/stores/store";
import * as apiAuthorization from "~/api/authorization";

const loginStore = useStore();

if (loginStore.isLoggedIn()) {
  navigateTo("/dashboard");
}

const username = ref("");
const password = ref("");
const rememberMe = ref(false);
const { data: health } = useFetch("/health", getFetchOptions(Method.GET));

async function doLogin() {
  const loginData = {
    email: username.value,
    password: password.value,
    rememberMe: rememberMe.value,
  };
  console.log("doLogin", loginData);
  const response = await apiAuthorization.login(loginData);
  console.log(response);

  // handle happy case
  if (response.code === apiAuthorization.ResponseCode.OK) {
    navigateTo("/dashboard");
    return;
  }

  // const result = await useFetch("/login", getFetchOptions(Method.POST, loginData));
  // if (result.status.value === "success") {
  //   console.log("login successful");
  //   navigateTo("/dashboard");
  // }
}
</script>

<style scoped></style>
