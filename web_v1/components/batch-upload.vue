<template>
  <div class="card bg-white shadow-md p-10">
    <h2>
      Batch upload into <b>{{ activeProjectName }}</b>
      <span v-if="cameraSelected">
        with images from your <b>{{ camera.name }}</b></span
      >
    </h2>
    <div class="divider"></div>
    <ul class="steps">
      <li :class="`step step-primary ${batchCreated ? '' : 'hover click hover:cursor-pointer'}`" @click="resetCamera">Select Camera</li>
      <li :class="`step ${cameraSelected ? 'step-primary' : ''}`">Create batch</li>
      <li :class="`step ${batchCreated ? 'step-primary' : ''}`">Upload</li>
      <li :class="`step ${doneUploading ? 'step-primary' : ''}`">Done</li>
    </ul>
    <div class="divider"></div>
    <div v-if="!cameraSelected">
      <CameraPicker @selected="setCamera"></CameraPicker>
    </div>
    <div v-if="cameraSelected && !batchCreated">
      <div class="flex">
        <div class="form-control w-full mr-2">
          <input v-model="batchName" type="text" placeholder="Type here" class="input input-bordered w-full" />
        </div>
        <button class="btn btn-outline btn-primary w-64" @click="createBatch">Next</button>
      </div>
    </div>

    <div v-if="batchCreated && !doneUploading">
      <div id="uploadDropzone" ref="uploadDropzone" class="dropzone"></div>
      <button class="btn btn-outline btn-primary w-64" @click="finalizeUpload">Done</button>
    </div>

    <div v-if="doneUploading">
      <button class="btn btn-outline btn-primary w-64" @click="navigateToBatch">Go to batch</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Camera } from "~/api/camera";
import { storeToRefs } from "pinia";
import { getDateTimeString, requestCreate, API_BASE_URL } from "~/api/common";
import { Batch } from "~/api/batch";
import Dropzone from "dropzone";

const store = useStore();
const { activeProjectId, activeProjectName } = storeToRefs(store);
const router = useRouter();

const cameraSelected = ref(false);
const batchCreated = ref(false);
const doneUploading = ref(false);

const batchName = ref("");
const batch = ref({} as Batch);

const uploadProgress = ref(0.0);

let uploadDropzone: Dropzone;

const uploadUrl = `${API_BASE_URL}/projects/${activeProjectId.value}/images`;

function setDefaultBatchName() {
  const now = new Date();
  batchName.value = `${activeProjectName.value} - Batch ${getDateTimeString(now.toISOString())} - ${camera.value.name}`;
}

const camera: Ref<Camera> = ref({} as Camera);

const setCamera = (c: Camera) => {
  camera.value = c;
  cameraSelected.value = true;
  setDefaultBatchName();
};

function resetCamera() {
  if (batchCreated.value) return;
  cameraSelected.value = false;
  camera.value = {} as Camera;
}

async function createBatch() {
  batchCreated.value = true;
  const { item } = await requestCreate<Batch>(`/projects/${activeProjectId.value}/batches`, {
    name: batchName.value,
  });
  if (!item) {
    console.log("Error creating batch");
    return;
  }
  batch.value = item;
  uploadDropzone = new Dropzone("div#uploadDropzone", {
    url: uploadUrl,
    method: "POST",
    maxThumbnailFilesize: 50,
    autoProcessQueue: true,
    withCredentials: true,
  });

  uploadDropzone.on("sending", function (file: any, xhr: any, formData: any) {
    formData.append("cameraId", camera.value.id);
    formData.append("batchId", batch.value.id);
  });
}

function finalizeUpload() {
  doneUploading.value = true;
}

function navigateToBatch() {
  router.push(`/dashboard/projects/${activeProjectId.value}/images?batch=${batch.value.id}`);
}
</script>
