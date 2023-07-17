<template>
  <div class="hero min-h-screen bg-base-200">
    <div class="hero-content flex-col">
      <div class="card w-100 shadow-2xl bg-base-100">
        <div class="card-body">
          <div class="form-control">
            <label class="label">
              <span class="label-text">Email</span>
            </label>
            <input
              type="text"
              placeholder="email"
              class="input input-bordered"
            />
            <label class="label">
              <NuxtLink to="login" class="label-text-alt link link-hover"
                >Back to login</NuxtLink
              >
            </label>
          </div>
          <div class="form-control mt-6">
            <button class="btn btn-secondary" @click="doPasswordReset">
              Reset password
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
function doPasswordReset() {
  console.log("doPasswordReset");
}
</script>

<style scoped></style>
