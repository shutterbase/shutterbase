<template>
  <TransitionRoot as="template" :show="show">
    <Dialog as="div" class="relative z-10" @close="emit('closed')">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity"></div>
      </TransitionChild>

      <div class="fixed inset-0 z-10 w-screen overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enter-to="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 translate-y-0 sm:scale-100"
            leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel
              class="relative w-full max-w-lg transform overflow-hidden rounded-lg border border-primary-200 bg-surface text-left shadow-panel transition-all dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark sm:my-8"
            >
              <!-- header -->
              <div class="flex items-start justify-between gap-4 border-b border-primary-100 px-6 py-5 dark:border-primary-800">
                <div class="flex items-start gap-3">
                  <span
                    class="mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-500/10 text-accent-600 dark:bg-accent-500/15 dark:text-accent-400"
                  >
                    <UserGroupIcon class="h-5 w-5" aria-hidden="true" />
                  </span>
                  <div>
                    <p class="label-mono text-accent-600 dark:text-accent-400">Project member</p>
                    <DialogTitle as="h3" class="display mt-1 text-xl text-primary-900 dark:text-white">{{ create ? "Add member" : "Change role" }}</DialogTitle>
                  </div>
                </div>
                <button
                  type="button"
                  class="-mr-1 -mt-1 inline-flex h-8 w-8 flex-shrink-0 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:hover:bg-primary-800 dark:hover:text-primary-200"
                  @click="emit('closed')"
                >
                  <span class="sr-only">Close</span>
                  <XMarkIcon class="h-5 w-5" aria-hidden="true" />
                </button>
              </div>

              <!-- body -->
              <div class="space-y-5 px-6 py-5">
                <div v-if="create">
                  <label for="member-user" class="label-mono block text-primary-500 dark:text-primary-400">User</label>
                  <SearchSelect
                    id="member-user"
                    v-model="selectedUserId"
                    class="mt-2"
                    aria-label="User"
                    placeholder="Select a user…"
                    empty-text="No user matches"
                    width-class="w-full"
                    :disabled="availableUsers.length === 0"
                    :options="userOptions"
                  />
                  <p v-if="availableUsers.length === 0" class="mt-2 text-sm text-primary-500 dark:text-primary-400">Every user is already a member of this project.</p>
                </div>
                <div v-else>
                  <p class="label-mono text-primary-500 dark:text-primary-400">User</p>
                  <p class="mt-2 text-sm font-medium text-primary-900 dark:text-white">{{ member?.user.firstName }} {{ member?.user.lastName }}</p>
                </div>

                <div>
                  <label for="member-role" class="label-mono block text-primary-500 dark:text-primary-400">Role</label>
                  <select id="member-role" v-model="selectedRoleId" :class="fieldClass">
                    <option value="" disabled>Select a role…</option>
                    <option v-for="r in roles" :key="r.id" :value="r.id">{{ prettyRole(r.key) }}</option>
                  </select>
                </div>
              </div>

              <!-- footer -->
              <div class="flex flex-row-reverse gap-3 border-t border-primary-100 px-6 py-4 dark:border-primary-800">
                <button
                  type="button"
                  :disabled="!canSave"
                  class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
                  @click="save"
                >
                  {{ create ? "Add member" : "Save" }}
                </button>
                <button
                  type="button"
                  class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
                  @click="emit('closed')"
                >
                  Cancel
                </button>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import SearchSelect, { SearchSelectOption } from "src/components/SearchSelect.vue";
import { UserGroupIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { computed, ref, watch } from "vue";
import { ProjectAssignment, Role, User } from "src/types/api";

interface Props {
  show: boolean;
  create: boolean;
  roles: Role[];
  availableUsers?: User[];
  member?: ProjectAssignment | null;
}
const props = withDefaults(defineProps<Props>(), { create: false, roles: () => [], availableUsers: () => [], member: null });

const emit = defineEmits<{
  add: [{ userId: string; roleId: string }];
  edit: [{ id: string; roleId: string }];
  closed: [];
}>();

const fieldClass =
  "mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:hover:border-primary-600";

const userOptions = computed<SearchSelectOption[]>(() => props.availableUsers.map((u) => ({ value: u.id, label: `${u.firstName} ${u.lastName}`, hint: u.username })));

const selectedUserId = ref("");
const selectedRoleId = ref("");

// Reset the form each time the dialog opens (prefill the role when editing).
watch(
  () => props.show,
  (open) => {
    if (!open) return;
    selectedUserId.value = "";
    selectedRoleId.value = props.member?.role?.id ?? "";
  },
);

const canSave = computed(() => (props.create ? !!selectedUserId.value : true) && !!selectedRoleId.value);

function prettyRole(key: string): string {
  return key.replace(/([A-Z])/g, " $1").replace(/^./, (c) => c.toUpperCase());
}

function save() {
  if (!canSave.value) return;
  if (props.create) {
    emit("add", { userId: selectedUserId.value, roleId: selectedRoleId.value });
  } else if (props.member) {
    emit("edit", { id: props.member.id, roleId: selectedRoleId.value });
  }
}
</script>
