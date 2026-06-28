<template>
  <div ref="dropzoneContainer" id="dropzoneContainer" class="flex items-center justify-center w-full">
    <label
      for="dropzoneFile"
      :class="[
        'flex flex-col items-center justify-center w-full h-64 rounded-lg border-2 border-dashed cursor-pointer transition-colors',
        isDragOver
          ? 'border-accent-500 bg-accent-500/10 dark:border-accent-400 dark:bg-accent-500/15'
          : 'border-primary-300 bg-surface-muted hover:bg-primary-100 hover:border-primary-400 dark:border-primary-700 dark:bg-surface-dark-muted dark:hover:bg-primary-800 dark:hover:border-primary-600',
      ]"
    >
      <div class="flex flex-col items-center justify-center pt-5 pb-6">
        <svg
          :class="['w-8 h-8 mb-4 transition-colors', isDragOver ? 'text-accent-600 dark:text-accent-400' : 'text-primary-400 dark:text-primary-500']"
          aria-hidden="true"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 20 16"
        >
          <path
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"
          />
        </svg>
        <p class="mb-2 text-sm text-primary-600 dark:text-primary-300"><span class="font-semibold text-primary-800 dark:text-primary-100">Click to upload</span> or drag and drop</p>
        <p class="label-mono-sm text-primary-500 dark:text-primary-400">{{ fileTypeText }}</p>
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
  files: [File[]];
}>();

const dropzoneContainer = ref<HTMLElement | null>(null);
const dropzoneFile = ref<HTMLInputElement | null>(null);
const isDragOver = ref(false);

function registerDropzone() {
  if (!dropzoneContainer.value || !dropzoneFile.value) {
    console.log("Drop container or file input not found");
    return;
  }

  dropzoneContainer.value.ondragover = dropzoneContainer.value.ondragenter = function (evt) {
    evt.preventDefault();
    isDragOver.value = true;
  };

  dropzoneContainer.value.ondragleave = function () {
    isDragOver.value = false;
  };

  dropzoneContainer.value.ondrop = function (evt) {
    isDragOver.value = false;
    if (!dropzoneFile.value || !evt.dataTransfer) {
      return;
    }
    dropzoneFile.value.files = evt.dataTransfer.files;
    emit("files", Array.from(dropzoneFile.value.files));
    evt.preventDefault();
  };

  dropzoneFile.value.onchange = function () {
    if (!dropzoneFile.value?.files) {
      return;
    }
    emit("files", Array.from(dropzoneFile.value.files));
    dropzoneFile.value.value = "";
  };
}

onMounted(registerDropzone);
</script>
