<template>
  <div class="">
    <div class="mx-auto max-w-7xl overflow-hidden sm:px-6 lg:px-8">
      <ImagesHeader :total-image-count="totalImageCount" @search="updateSearchText" />

      <div class="mt-10 grid grid-cols-1 border-l border-gray-200 dark:border-gray-600 sm:mx-0 md:grid-cols-3 lg:grid-cols-4">
        <ImageGridTile v-for="image in images" :image="image" :key="image.id" />
      </div>
      <ImagesFooter :current-image-count="images.length" :total-image-count="totalImageCount" :filtered="filtered" @load-more="() => loadImages(false)" />
    </div>
  </div>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>
<script setup lang="ts">
import { storeToRefs } from "pinia";
import pb from "src/boot/pocketbase";
import ImageGridTile from "src/components/image/ImageGridTile.vue";
import ImagesHeader, { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import ImagesFooter from "src/components/image/ImagesFooter.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { useUserStore } from "src/stores/user-store";
import { ImagesResponse } from "src/types/pocketbase";
import { onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useDebounceFn } from "@vueuse/core";

const router = useRouter();

const { activeProject, preferredImageSortOrder } = storeToRefs(useUserStore());

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const images = ref<ImagesResponse[]>([]);
const totalImageCount = ref(0);
const page = ref(1);
const loading = ref(false);
const filtered = ref(false);

const searchText = ref("");
function updateSearchText(text: string) {
  searchText.value = text;
}

window.onscroll = async function (ev) {
  if (window.innerHeight + window.scrollY + 100 >= document.body.scrollHeight) {
    if (totalImageCount.value > 0 && images.value.length < totalImageCount.value) {
      loadImages(false);
    }
  }
};

function getFilter() {
  const and = [];
  and.push(`project='${activeProject.value.id}'`);

  if (searchText.value) {
    filtered.value = true;
  } else {
    filtered.value = false;
  }

  if (searchText.value) {
    and.push(`(computedFileName ~ '${searchText.value}' || fileName ~ '%${searchText.value}%')`);
  }
  return `(${and.join(" && ")})`;
}

function getSort() {
  return preferredImageSortOrder.value === SORT_ORDER.LATEST_FIRST ? "-capturedAtCorrected" : "capturedAtCorrected";
}

onMounted(() => loadImages(true));
const reloadDebounced = useDebounceFn(() => loadImages(true), 500);
watch(preferredImageSortOrder, () => loadImages(true));
watch(searchText, reloadDebounced);
async function loadImages(reload: boolean) {
  if (loading.value) return;
  loading.value = true;
  try {
    if (reload) page.value = 1;
    const result = await pb.collection<ImagesResponse>("images").getList(page.value, 20, {
      filter: getFilter(),
      sort: getSort(),
      expand: "camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag", //"image_tag_assignments_via_image", //  image_tag_assignments_via_image.image_tag
    });
    totalImageCount.value = result.totalItems;
    page.value++;

    if (reload) {
      images.value = [];
    }
    images.value.push(...result.items);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    loading.value = false;
  }
}
</script>
