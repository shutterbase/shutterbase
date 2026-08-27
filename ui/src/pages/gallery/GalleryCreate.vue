<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <main class="max-w-3xl">
      <p class="label-mono text-accent-600 dark:text-accent-400">New gallery</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Create a public gallery</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
        The key is what a gallery deployment is pinned to (<code>GALLERY_KEY</code>) and cannot change later. Branding, texts and legal links are edited afterwards.
      </p>
      <div class="mt-10 space-y-12">
        <CreateGroup @edit="updateData" headline="Identity" subtitle="Key and display name" :fields="fields" />
        <button
          @click="create"
          :disabled="!draft.key || !draft.name"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 disabled:opacity-50"
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
import { useRouter } from "vue-router";
import CreateGroup, { Field, FieldType, CreateData } from "src/components/CreateGroup.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { api } from "src/api";
import { Gallery } from "src/types/api";
import { showNotificationToast } from "src/boot/mitt";

const router = useRouter();
const draft = ref<Partial<Gallery>>({});
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function updateData(editData: CreateData<Gallery>) {
  draft.value = { ...draft.value, ...editData };
}

async function create() {
  try {
    const g = await api.galleries.create({ key: draft.value.key!, name: draft.value.name! });
    showNotificationToast({ headline: `Gallery '${g.key}' created`, type: "success" });
    await router.push({ name: "gallery-edit", params: { id: g.id } });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

const fields: Field<Gallery>[] = [
  { key: "key", label: "Key", type: FieldType.TEXT, hint: "lowercase slug, e.g. fsg — pinned by the deployment, immutable" },
  { key: "name", label: "Name", type: FieldType.TEXT, hint: "shown as the site title" },
];
</script>
