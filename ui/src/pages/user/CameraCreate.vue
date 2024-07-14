<template>
  <div class="mx-auto max-w-7xl w-full lg:flex lg:gap-x-16 lg:px-8">
    <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
      <div class="mx-auto max-w-2xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
        <CreateGroup @edit="updateData" headline="Camera Information" subtitle="General information concerning the camera" :fields="informationFields" />
        <button
          @click="createItem"
          class="block rounded-md bg-secondary-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-secondary-500 dark:hover:bg-secondary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"
        >
          Create
        </button>
      </div>
    </main>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import CreateGroup, { Field, FieldType, CreateData } from "src/components/CreateGroup.vue";
import { CamerasResponse } from "src/types/pocketbase";
import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";

const router = useRouter();
const route = useRoute();

type ITEM_TYPE = CamerasResponse;
const ITEM_COLLECTION = "cameras";
const ITEM_NAME = "camera";

const userId = ref<string>(`${route.params.userid}`);

const item = ref<ITEM_TYPE>({} as ITEM_TYPE);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function updateData(editData: CreateData<ITEM_TYPE>) {
  item.value = { ...item.value, ...editData };
}

async function createItem() {
  try {
    console.log(`Creating ${ITEM_NAME} ${item.value.name}`);
    const response = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).create({ ...item.value, user: userId.value });
    const itemId = response.id;
    console.log(`Project created with ID ${itemId}`);
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} created`, type: "success" });
    await router.push({ name: `cameras`, params: { cameraid: itemId, userid: userId.value } });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ITEM_TYPE>[] = [{ key: "name", label: "Name", type: FieldType.TEXT }];
</script>
