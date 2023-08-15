<template>
  <client-only>
    <div class="object-top">
      <div class="my-2">
        <input type="text" v-model="filterTagsText" placeholder="Filter tags" class="input input-bordered w-full max-w-xs mr-2" />
        <input type="text" v-model="searchText" placeholder="Search" class="input input-bordered w-full max-w-xs mr-2" />
        <div :class="`btn mr-2`" @click="loadImages">Go!</div>
      </div>
      <div class="grid gap-2 grid-cols-1 md:grid-cols-1 lg:grid-cols-2 2xl:grid-cols-3">
        <div v-for="image in images" :key="image.id" class="max-h-80" style="max-height: 20rem">
          <div class="relative before:content-[''] before:rounded-md before:absolute before:inset-0 before:bg-black before:bg-opacity-20">
            <img :src="getImageThumbnailUrl(image)" class="rounded-md" style="max-height: 20rem; margin: 0 auto" />
            <div class="absolute inset-0 p-8 text-white flex flex-col">
              <div class="relative">
                <a class="absolute inset-0" :href="getDetailLink(image)"></a>
                <h1 class="text-md font-bold mb-3">{{ image.fileName }}</h1>
                <p class="font-sm font-light">{{ image.edges.createdBy.firstName }} {{ image.edges.createdBy.lastName }}</p>
              </div>
              <!--<div class="mt-auto flex flex-row">
              <span class="bg-white bg-opacity-60 py-1 px-4 rounded-md text-black">#tag</span>
            </div>-->
            </div>
          </div>
        </div>
      </div>
    </div>
  </client-only>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getDateTimeString, requestList, API_BASE_URL } from "~/api/common";

const router = useRouter();
const batchId = ref(router.currentRoute.value.query.batch as string);
const filterTagsText = ref("");
const searchText = ref("");

const images = ref<Image[]>([]);

const requestUrl = computed(() => {
  let url = `/projects/${props.projectId}/images?`;
  const tags = filterTagsText.value
    .replace(/,/g, " ")
    .split(" ")
    .filter((tag) => tag.length > 0)
    .map((tag) => tag.trim())
    .join(",");
  let queryParams = [];

  if (batchId.value) {
    queryParams.push(`batch=${batchId.value}`);
  }
  if (tags) {
    queryParams.push(`tags=${tags}`);
  }

  if (searchText.value && searchText.value.length > 0) {
    queryParams.push(`search=${searchText.value}`);
  }

  url += queryParams.join("&");
  return url;
});

function getFetchOptions(method: Method, body?: any, watch?: any[]) {
  const headers = useRequestHeaders(["cookie"]);
  const options: any = {
    method,
    headers,
    baseURL: API_BASE_URL,
    credentials: "include",
  };
  if (body) {
    options.body = body;
  }
  if (watch) {
    options.watch = watch;
  }
  return options;
}

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

function getImageThumbnailUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/thumb?size=512`;
}

function getDetailLink(image: Image): string {
  let url = `/dashboard/projects/${props.projectId}/image-detail?image=${image.id}`;
  if (batchId.value) {
    url += `&batch=${batchId.value}`;
  }
  return url;
}

async function loadImages() {
  const { items } = await requestList<Image>(requestUrl.value, getFetchOptions(Method.GET));
  if (items) images.value = items;
}

onMounted(() => {
  loadImages();
});

// watch([batchId, searchText], loadImages, { immediate: true });
</script>
