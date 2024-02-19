<style scoped></style>
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
              <span class="label-text">First name</span>
            </label>
            <input v-model="firstName" type="text" placeholder="first name" class="input input-bordered" />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">Last name</span>
            </label>
            <input v-model="lastName" type="text" placeholder="last name" class="input input-bordered" />
          </div>
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
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">Confirm password</span>
            </label>
            <input v-model="passwordConfirmation" type="password" placeholder="password confirmation" class="input input-bordered" />
          </div>
          <div class="form-control mt-6">
            <button class="btn btn-secondary" @click="doRegister">Register</button>
          </div>
          <label class="label">
            <NuxtLink to="login" class="label-text-alt link link-hover">Back to login</NuxtLink>
          </label>
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
const firstName = ref("");
const lastName = ref("");
const passwordConfirmation = ref("");

async function doRegister() {
  if (password.value !== passwordConfirmation.value) {
    alert("Passwords do not match");
    return;
  }
  const registrationData = {
    email: username.value,
    password: password.value,
    firstName: firstName.value,
    lastName: lastName.value,
  };
  const response = await apiAuthorization.register(registrationData);
  console.log(response);

  // handle happy case
  if (response.code === apiAuthorization.ResponseCode.OK) {
    navigateTo("/login");
    return;
  } else {
    alert("Registration failed");
  }

  // const result = await useFetch("/login", getFetchOptions(Method.POST, loginData));
  // if (result.status.value === "success") {
  //   console.log("login successful");
  //   navigateTo("/dashboard");
  // }
}
</script>
