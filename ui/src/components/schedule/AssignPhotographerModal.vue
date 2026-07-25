<template>
  <TransitionRoot as="template" :show="show">
    <Dialog as="div" class="relative z-50" @close="emit('closed')">
      <TransitionChild as="template" enter="ease-out duration-200" enter-from="opacity-0" enter-to="opacity-100" leave="ease-in duration-150" leave-from="opacity-100" leave-to="opacity-0">
        <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity"></div>
      </TransitionChild>

      <div class="fixed inset-0 z-50 w-screen overflow-y-auto">
        <div class="flex min-h-full items-center justify-center p-4">
          <TransitionChild
            as="template"
            enter="ease-out duration-200"
            enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100"
            leave="ease-in duration-150"
            leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <DialogPanel class="w-full max-w-sm transform rounded-lg border border-primary-200 bg-surface p-5 text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark">
              <DialogTitle as="h3" class="display text-lg text-primary-900 dark:text-white">Assign photographer</DialogTitle>
              <input
                v-model="query"
                type="search"
                placeholder="Search members…"
                aria-label="Search members"
                class="mt-3 h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
              />
              <ul class="mt-3 max-h-64 space-y-1 overflow-y-auto">
                <li v-for="member in filtered" :key="member.id">
                  <button
                    type="button"
                    class="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-primary-700 transition-colors hover:bg-accent-500/10 hover:text-accent-700 dark:text-primary-200 dark:hover:text-accent-200"
                    @click="emit('assign', member.id)"
                  >
                    <UserBubble :user="member" />
                    <span class="truncate">{{ member.firstName }} {{ member.lastName }}</span>
                  </button>
                </li>
                <li v-if="filtered.length === 0" class="px-2 py-4 text-center text-sm text-primary-400">No member matches.</li>
              </ul>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { computed, ref, watch } from "vue";
import UserBubble from "src/components/schedule/UserBubble.vue";
import { EmbeddedUser } from "src/types/api";

const props = defineProps<{ show: boolean; candidates: EmbeddedUser[] }>();
const emit = defineEmits<{ closed: []; assign: [string] }>();

const query = ref("");
watch(
  () => props.show,
  () => (query.value = ""),
);

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  const sorted = [...props.candidates].sort((a, b) => `${a.firstName} ${a.lastName}`.localeCompare(`${b.firstName} ${b.lastName}`));
  if (!q) return sorted;
  return sorted.filter((m) => `${m.firstName} ${m.lastName}`.toLowerCase().includes(q));
});
</script>
