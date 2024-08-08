<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Project Information"
        subtitle="General information concerning this project"
        :fields="informationFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="Copyright Data"
        subtitle="Copyright information to be embedded into the EXIF data"
        :fields="copyrightFields"
        :item="item"
      />
      <DetailEditGroup
        :allow-edit="userStore.isProjectAdminOrHigher()"
        @edit-save="saveItem"
        headline="AI Options"
        subtitle="Options for AI image tagging"
        :fields="aiFields"
        :item="item"
      />
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import { ProjectsResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
import { useUserStore } from "src/stores/user-store";
const route = useRoute();

const userStore = useUserStore();

type ITEM_TYPE = ProjectsResponse;
const ITEM_COLLECTION = "projects";
const ITEM_NAME = "project";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadItem() {
  const itemId: string = `${route.params.id}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).getOne(itemId);
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(editData: EditData<ITEM_TYPE>) {
  if (!item.value) {
    console.log(`No ${ITEM_NAME} to save`);
    return;
  }

  const rollbackData = { ...item.value };
  item.value = { ...item.value, ...editData };

  try {
    console.log(`Saving ${ITEM_NAME} ${item.value.id}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).update(item.value.id, editData);
    item.value = response;
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} saved`, type: "success" });
  } catch (error: any) {
    item.value = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ITEM_TYPE>[] = [
  { key: "name", label: "Name", type: FieldType.TEXT },
  { key: "description", label: "Description", type: FieldType.TEXT },
];

const aiFields: Field<ITEM_TYPE>[] = [{ key: "aiSystemMessage", label: "System Message", type: FieldType.TEXT }];

const copyrightFields: Field<ITEM_TYPE>[] = [
  { key: "copyright", label: "Copyright", type: FieldType.TEXT },
  { key: "copyrightReference", label: "Copyright reference", type: FieldType.TEXT },
  { key: "locationName", label: "Location name", type: FieldType.TEXT },
  { key: "locationCode", label: "Location code", type: FieldType.TEXT },
  { key: "locationCity", label: "Location city", type: FieldType.TEXT },
];

watch(route, loadItem);
onMounted(loadItem);
</script>
