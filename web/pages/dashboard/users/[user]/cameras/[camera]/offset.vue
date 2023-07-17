<template>
  <NuxtLayout name="logged-in">
    <div class="card bg-white shadow-md p-10">
      <div class="card hero">
        <div class="hero-content flex-col lg:flex-row">
          <img :src="qrCodeUrl" class="w-96 h-96 rounded-lg shadow-2xl" />
          <div class="ml-10">
            <h1 class="text-5xl font-bold">
              Photograph QR code with your <b>{{ camera?.name || "" }}</b>
            </h1>
            <p class="py-6">For best results, upload a JPG of the entire screen in lowest camera-native resolution</p>
            <button class="btn btn-primary" onclick="upload_dialog.showModal()">Upload</button>
          </div>
        </div>
      </div>
    </div>
    <dialog id="upload_dialog" class="modal">
      <form method="dialog" class="modal-box">
        <h3 class="font-bold text-lg">Drop your time offset image below:</h3>
        <div id="offsetDropzone" ref="offsetDropzone" class="dropzone"></div>
        <div class="modal-action">
          <!-- if there is a button in form, it will close the modal -->
          <button class="btn">Close</button>
        </div>
      </form>
    </dialog>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import uid from "tiny-uid";
import { API_BASE_URL, getFetchOptions, Method } from "~/api/common";
import Dropzone from "dropzone";

const router = useRouter();
const userId = router.currentRoute.value.params.user;
const cameraId = router.currentRoute.value.params.camera;

function newQrCodeUrl() {
  return `${API_BASE_URL}/time/qr/${uid()}`;
}

const qrCodeUrl = ref(newQrCodeUrl());
const uploadUrl = `${API_BASE_URL}/users/${userId}/cameras/${cameraId}/time-offsets`;
const timerInterval = ref<any>(null);
const offsetDropzone = ref(null);

onMounted(() => {
  console.log(uploadUrl);
  timerInterval.value = setInterval(() => {
    qrCodeUrl.value = newQrCodeUrl();
  }, 250);

  new Dropzone("div#offsetDropzone", {
    url: uploadUrl,
    method: "POST",
    autoProcessQueue: true,
    withCredentials: true,
    maxFiles: 1,
  });
});

onUnmounted(() => {
  if (timerInterval.value) {
    clearInterval(timerInterval.value);
  }
});

onMounted(() => {});

const { data: camera } = await useFetch(`/users/${userId}/cameras/${cameraId}`, getFetchOptions(Method.GET));
</script>

<style scoped></style>
