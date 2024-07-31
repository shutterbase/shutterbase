<template>
  <input type="button" value="Sync Tags" @click="syncTags" />
  <input type="text" v-model="filter" @change="doRequest" />
  {{ images.length }}
  <p v-for="image in images" :key="image.id">
    {{ image.capturedAtCorrected }} - {{ image.computedFileName }}
    <span class="mx-2" v-for="imageTagAssignment in image.expand.image_tag_assignments_via_image" :key="imageTagAssignment.id">{{ imageTagAssignment.expand.imageTag.name }}</span>
  </p>
</template>

<script setup lang="ts">
import { Ref, ref } from "vue";

import pb from "src/boot/pocketbase";
import TaggingDialog from "src/components/image/TaggingDialog.vue";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { ImageWithTagsType } from "src/types/custom";

const filter: Ref<string> = ref("");
const images: Ref<ImageWithTagsType[]> = ref([]);

async function doRequest() {
  const result = await pb.collection<ImageWithTagsType>("images").getList(0, 1000, {
    filter: filter.value, // `serialized_image_tags_via_image.imageTags ?~ '"qtzb27wceinc1x3"' && serialized_image_tags_via_image.imageTags ?~ '"89ewmdtae932h0r"'`,
    sort: "-capturedAtCorrected",
    expand: "camera, project, image_tag_assignments_via_image, image_tag_assignments_via_image.imageTag",
  });
  images.value = result.items;
}

async function syncTags() {
  await pb.send("/api/sync-image-tags", {});
}
</script>
