<template>
  <div ref="dropzoneContainer" id="dropzoneContainer" class="flex items-center justify-center w-full">
    <label
      for="dropzoneFile"
      class="flex flex-col items-center justify-center w-full h-64 border-2 border-gray-300 border-dashed rounded-lg cursor-pointer bg-gray-50 dark:hover:bg-bray-800 dark:bg-gray-700 hover:bg-gray-100 dark:border-gray-600 dark:hover:border-gray-500 dark:hover:bg-gray-600"
    >
      <div class="flex flex-col items-center justify-center pt-5 pb-6">
        <svg class="w-8 h-8 mb-4 text-gray-500 dark:text-gray-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 16">
          <path
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"
          />
        </svg>
        <p class="mb-2 text-sm text-gray-500 dark:text-gray-400"><span class="font-semibold">Click to upload</span> or drag and drop</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ fileTypeText }}</p>
      </div>
      <input ref="dropzoneFile" id="dropzoneFile" type="file" class="hidden" :accept="fileExtensionString" :multiple="multiple" />
    </label>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";

interface Props {
  uploading?: boolean;
  multiple?: boolean;
  fileExtensions?: string[];
  fileTypeText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  uploading: () => false,
  multiple: () => true,
  fileExtensions: () => ["jpeg", "jpg", "JPEG", "JPG"],
  fileTypeText: () => "JPEG",
});

const fileExtensionString = props.fileExtensions.map((e) => `.${e}`).join(", ");

const emit = defineEmits<{
  files: [FileList];
}>();

const dropzoneContainer = ref<HTMLElement | null>(null);
const dropzoneFile = ref<HTMLInputElement | null>(null);

function registerDropzone() {
  if (!dropzoneContainer.value || !dropzoneFile.value) {
    console.log("Drop container or file input not found");
    return;
  }

  dropzoneContainer.value.ondragover = dropzoneContainer.value.ondragenter = function (evt) {
    evt.preventDefault();
  };

  dropzoneContainer.value.ondrop = function (evt) {
    if (!dropzoneFile.value || !evt.dataTransfer) {
      return;
    }
    dropzoneFile.value.files = evt.dataTransfer.files;
    emit("files", dropzoneFile.value.files);
    evt.preventDefault();
  };

  dropzoneFile.value.onchange = function () {
    if (!dropzoneFile.value?.files) {
      return;
    }
    emit("files", dropzoneFile.value.files);
    dropzoneFile.value.value = "";
  };
}

onMounted(registerDropzone);
</script>
