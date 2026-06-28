<template>
  <main class="py-8">
    <div class="mx-auto max-w-3xl space-y-16 sm:space-y-20 lg:mx-0 lg:max-w-none">
      <DetailEditGroup @edit-save="saveItem" headline="User Information" subtitle="General information concerning this user" :fields="informationFields" :item="item" />
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import { UsersResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { capitalize } from "src/util/stringUtils";
const route = useRoute();

type ITEM_TYPE = UsersResponse;
const ITEM_COLLECTION = "users";
const ITEM_NAME = "user";

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadItem() {
  const itemId: string = `${route.params.userid}`;
  if (!itemId || itemId === "") {
    console.log(`No ${ITEM_NAME} ID provided`);
    return;
  }

  try {
    console.log(`Loading ${ITEM_NAME} ${itemId}`);
    const response = await api.users.get(itemId);
    item.value = response;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(editData: EditData<ITEM_TYPE>) {
  if (!item.value) {
    console.log("No item to save");
    return;
  }

  const rollbackData = { ...item.value };
  item.value = { ...item.value, ...editData };

  try {
    console.log(`Saving item ${item.value.id}`);
    const response = await api.users.update(item.value.id, editData as Record<string, any>);
    item.value = response;
    showNotificationToast({ headline: `${capitalize(ITEM_NAME)} saved`, type: "success" });
  } catch (error: any) {
    item.value = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const informationFields: Field<ITEM_TYPE>[] = [
  { key: "firstName", label: "First name", type: FieldType.TEXT },
  { key: "lastName", label: "Last name", type: FieldType.TEXT },
  { key: "username", label: "Username", type: FieldType.TEXT },
  { key: "email", label: "Email", type: FieldType.TEXT },
  { key: "copyrightTag", label: "Copyright Tag", type: FieldType.TEXT },
];

watch(route, loadItem);
onMounted(loadItem);
</script>
