<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table
      dense
      :items="items"
      :columns="columns"
      name="Gallery"
      subtitle="Public, white-label gallery sites. Each deployment serves one gallery by its key; projects publish themselves onto a gallery in their settings."
      :loading="loading"
      :allow-add="userStore.isAdmin()"
      :add-callback="() => router.push({ name: 'gallery-create' })"
    ></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <ModalMessage
      :show="deleteCandidate !== null"
      :type="MessageType.CONFIRM_WARNING"
      headline="Delete gallery"
      :message="`Delete '${deleteCandidate?.name}'? Published projects are detached (not deleted); a deployment pinned to key '${deleteCandidate?.key}' serves nothing afterwards.`"
      confirmText="Delete"
      @confirmed="confirmDelete"
      @closed="deleteCandidate = null"
    />
  </div>
</template>

<script setup lang="ts">
import { Ref, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { api } from "src/api";
import { Gallery } from "src/types/api";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";

const router = useRouter();
const userStore = useUserStore();

const items: Ref<Gallery[]> = ref([]);
const loading = ref(false);
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const deleteCandidate: Ref<Gallery | null> = ref(null);

const columns: TableColumn<Gallery>[] = [
  { key: "key", label: "Key" },
  { key: "name", label: "Name" },
  { key: "domain", label: "Domain" },
  { key: "locale", label: "Locale" },
  { key: "active", label: "Status", formatter: (active) => (active ? "Active" : "Inactive") },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Settings", callback: (item) => router.push({ name: "gallery-edit", params: { id: item.id } }), type: TableRowActionType.EDIT },
      {
        key: "delete",
        label: "Delete",
        showCallback: () => userStore.isAdmin(),
        callback: (item) => (deleteCandidate.value = item),
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

async function load() {
  loading.value = true;
  try {
    items.value = (await api.galleries.list({ limit: 100 })).items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    loading.value = false;
  }
}

async function confirmDelete() {
  const g = deleteCandidate.value;
  deleteCandidate.value = null;
  if (!g) return;
  try {
    await api.galleries.remove(g.id);
    showNotificationToast({ headline: `Gallery '${g.key}' deleted`, type: "success" });
    await load();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(load);
</script>
