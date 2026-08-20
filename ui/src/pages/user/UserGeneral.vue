<template>
  <main class="py-8">
    <div class="mx-auto max-w-3xl space-y-10 lg:mx-0 lg:max-w-none">
      <!-- identity header: who am I looking at, and can they get in right now -->
      <header
        v-if="item"
        class="flex flex-wrap items-start gap-5 rounded-lg border border-primary-200 bg-surface p-6 shadow-panel dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark"
      >
        <span
          class="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-accent-500/15 font-data text-lg font-semibold text-accent-700 dark:bg-accent-500/20 dark:text-accent-200"
        >
          {{ initials }}
        </span>
        <div class="min-w-0 flex-1">
          <h2 class="display text-2xl text-primary-900 dark:text-white">{{ item.firstName }} {{ item.lastName }}</h2>
          <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 font-data text-sm text-primary-500 dark:text-primary-400">
            <span>{{ item.username }}</span>
            <span v-if="item.email" aria-hidden="true">·</span>
            <span v-if="item.email" class="truncate">{{ item.email }}</span>
          </p>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2.5 py-0.5 text-xs font-medium', item.active ? badgeSuccess : badgeError]">
              <component :is="item.active ? CheckCircleIcon : NoSymbolIcon" class="h-4 w-4" />
              {{ item.active ? "Active" : "Inactive" }}
            </span>
            <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2.5 py-0.5 text-xs font-medium', item.role?.key === 'admin' ? badgeAccent : badgeNeutral]">
              <ShieldCheckIcon v-if="item.role?.key === 'admin'" class="h-4 w-4" />
              {{ item.role?.key === "admin" ? "Platform admin" : "User" }}
            </span>
            <span v-if="item.forcePasswordChange" :class="['inline-flex items-center gap-1.5 rounded-md border px-2.5 py-0.5 text-xs font-medium', badgeWarning]">
              <KeyIcon class="h-4 w-4" />
              Must change password
            </span>
            <span v-if="isSelf" :class="['inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-medium', badgeNeutral]">That's you</span>
          </div>
        </div>
        <div v-if="item.projectAssignments?.length" class="w-full border-t border-primary-100 pt-4 dark:border-primary-800/70 sm:w-auto sm:border-0 sm:pt-0">
          <p class="label-mono text-primary-500 dark:text-primary-400">Projects</p>
          <p class="mt-1 font-data text-sm text-primary-800 dark:text-primary-100">{{ item.projectAssignments.length }}</p>
        </div>
      </header>

      <DetailEditGroup
        :allow-edit="isAdmin || isSelf"
        @edit-save="saveItem"
        headline="Profile"
        subtitle="Name, login identity and the copyright tag stamped into this photographer's EXIF"
        :fields="informationFields"
        :item="item"
      />

      <!-- Account access. Rendered for every viewer, disabled with the reason
           when they may not act: a control that simply is not there reads as
           "broken" — which is exactly how the missing role control got reported. -->
      <section v-if="item" class="border-t border-primary-100 pt-10 dark:border-primary-800/70">
        <h2 class="display text-xl text-primary-900 dark:text-white">Account access</h2>
        <p class="mt-1 max-w-prose text-sm text-primary-500 dark:text-primary-400">
          Whether this account can sign in, and what it may do platform-wide. Roles <em>inside</em> a project are managed on that project's Members page.
        </p>

        <p
          v-if="!isAdmin"
          class="mt-4 flex max-w-prose items-start gap-1.5 rounded-md border border-primary-200 bg-surface-muted px-2.5 py-2 text-xs text-primary-600 dark:border-primary-700 dark:bg-surface-dark-muted dark:text-primary-300"
        >
          <LockClosedIcon class="mt-px h-4 w-4 shrink-0" />
          <span>Only platform administrators can change sign-in access or the platform role.</span>
        </p>

        <dl class="mt-6 space-y-8">
          <div class="sm:flex sm:items-start sm:gap-6">
            <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-40 sm:shrink-0 sm:pt-2">Sign-in</dt>
            <dd class="mt-2 sm:mt-0">
              <div class="flex flex-wrap items-center gap-3">
                <span :class="['inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium', item.active ? badgeSuccess : badgeError]">
                  <component :is="item.active ? CheckCircleIcon : NoSymbolIcon" class="h-4 w-4" />
                  {{ item.active ? "Active" : "Inactive" }}
                </span>
                <button v-if="!item.active" type="button" @click="setActive(true)" :disabled="!isAdmin || busy" :class="[buttonBase, buttonAccent]">
                  <CheckCircleIcon class="h-4 w-4" />
                  Activate account
                </button>
                <button v-else type="button" @click="showDeactivateDialog = true" :disabled="!isAdmin || busy || isSelf" :class="[buttonBase, buttonDanger]">
                  <NoSymbolIcon class="h-4 w-4" />
                  Deactivate account
                </button>
              </div>
              <p class="mt-2 max-w-prose text-xs text-primary-500 dark:text-primary-400">
                {{
                  isSelf && item.active ? "You cannot deactivate your own account." : "An inactive account cannot sign in, and its sessions and API keys stop working immediately."
                }}
              </p>
            </dd>
          </div>

          <div class="sm:flex sm:items-start sm:gap-6">
            <dt class="label-mono text-primary-500 dark:text-primary-400 sm:w-40 sm:shrink-0 sm:pt-2">Platform role</dt>
            <dd class="mt-2 sm:mt-0">
              <div class="inline-flex rounded-lg border border-primary-200 bg-surface p-0.5 dark:border-primary-700 dark:bg-surface-dark">
                <button
                  v-for="option in roleOptions"
                  :key="option.value"
                  type="button"
                  @click="setRole(option.value)"
                  :disabled="!isAdmin || busy || isSelf"
                  :class="[
                    'inline-flex h-8 items-center gap-1.5 rounded-md px-3 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60',
                    currentRole === option.value
                      ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
                      : 'cursor-pointer text-primary-500 hover:bg-primary-100 hover:text-primary-700 dark:text-primary-400 dark:hover:bg-primary-800 dark:hover:text-primary-200',
                  ]"
                >
                  <component :is="option.icon" class="h-4 w-4" />
                  {{ option.label }}
                </button>
              </div>
              <p class="mt-2 max-w-prose text-xs text-primary-500 dark:text-primary-400">
                {{
                  isSelf
                    ? "You cannot change your own role — another administrator has to. That rule is what guarantees the platform never loses its last admin."
                    : "A platform admin manages every project, every user and every account. Grant it sparingly."
                }}
              </p>
            </dd>
          </div>
        </dl>
      </section>

      <section v-if="isAdmin && item" class="border-t border-primary-100 pt-10 dark:border-primary-800/70">
        <h2 class="display text-xl text-primary-900 dark:text-white">Set a new password</h2>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">
          Sets the password directly — the user is not asked for their old one. Tell them the new password over a channel you trust.
        </p>
        <form class="mt-6 max-w-sm space-y-4" @submit.prevent="submitPassword">
          <div>
            <label for="new-password" class="label-mono block text-primary-500 dark:text-primary-400">New password</label>
            <input id="new-password" v-model="newPassword" type="password" autocomplete="new-password" :class="inputClasses" class="mt-2" />
            <PasswordRequirements ref="pwReqs" :password="newPassword" always-visible />
          </div>
          <div>
            <label for="confirm-password" class="label-mono block text-primary-500 dark:text-primary-400">Confirm new password</label>
            <input id="confirm-password" v-model="newPasswordConfirm" type="password" autocomplete="new-password" :class="inputClasses" class="mt-2" />
          </div>
          <label class="inline-flex cursor-pointer items-center gap-2">
            <input
              v-model="requireChange"
              type="checkbox"
              class="h-4 w-4 rounded border-primary-300 bg-surface text-accent-600 focus:ring-2 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark"
            />
            <span class="text-sm text-primary-700 dark:text-primary-200">Require a change on next sign-in</span>
          </label>
          <p v-if="passwordError" class="text-sm font-medium text-error-600 dark:text-error-400">{{ passwordError }}</p>
          <button type="submit" :disabled="!canSubmitPassword || busy" :class="[buttonBase, buttonAccent]">
            <KeyIcon class="h-4 w-4" />
            Set password
          </button>
        </form>
      </section>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  <ModalMessage
    :show="showDeactivateDialog"
    :type="MessageType.CONFIRM_WARNING"
    @closed="showDeactivateDialog = false"
    headline="Deactivate account"
    :message="`Deactivate '${item?.username}'? They lose access immediately — running sessions and API keys stop working.`"
    confirmText="Deactivate"
    @confirmed="confirmDeactivate"
  />
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import DetailEditGroup, { Field, FieldType, EditData } from "src/components/DetailEditGroup.vue";
import PasswordRequirements from "src/components/PasswordRequirements.vue";
import { UsersResponse } from "src/types/pocketbase";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { normalizeCopyrightTag } from "src/util/copyrightTag";
import { CheckCircleIcon, KeyIcon, LockClosedIcon, NoSymbolIcon, ShieldCheckIcon, UserIcon } from "@heroicons/vue/24/outline";

const route = useRoute();
const userStore = useUserStore();

type ITEM_TYPE = UsersResponse;

const item: Ref<ITEM_TYPE | null> = ref(null);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const showDeactivateDialog = ref(false);
const busy = ref(false);

const isAdmin = computed(() => userStore.isAdmin());
const isSelf = computed(() => !!item.value && item.value.id === userStore.user?.id);
const initials = computed(() => `${item.value?.firstName?.[0] ?? ""}${item.value?.lastName?.[0] ?? ""}`.toUpperCase());

const badgeSuccess = "border-success-300 bg-success-50 text-success-800 dark:border-success-800/70 dark:bg-success-950/40 dark:text-success-200";
const badgeError = "border-error-300 bg-error-50 text-error-700 dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300";
const badgeWarning = "border-warning-300 bg-warning-50 text-warning-800 dark:border-warning-800/70 dark:bg-warning-950/40 dark:text-warning-200";
const badgeAccent = "border-accent-400/50 bg-accent-500/10 text-accent-700 dark:border-accent-400/40 dark:text-accent-200";
const badgeNeutral = "border-primary-200 bg-surface text-primary-700 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200";

const inputClasses =
  "h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600";
const buttonBase =
  "inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950";
const buttonAccent = "bg-accent-600 font-semibold text-white shadow-sm hover:bg-accent-500 active:bg-accent-700";
const buttonDanger =
  "border border-error-300 bg-error-50 text-error-700 hover:bg-error-100 dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300 dark:hover:bg-error-950/70";

const informationFields: Field<ITEM_TYPE>[] = [
  { key: "firstName", label: "First name", type: FieldType.TEXT },
  { key: "lastName", label: "Last name", type: FieldType.TEXT },
  { key: "username", label: "Username", type: FieldType.TEXT },
  { key: "email", label: "Email", type: FieldType.TEXT },
  { key: "copyrightTag", label: "Copyright Tag", type: FieldType.TEXT, transform: normalizeCopyrightTag },
];

async function loadItem() {
  const itemId: string = `${route.params.userid}`;
  if (!itemId || itemId === "") return;
  try {
    item.value = await api.users.get(itemId);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function saveItem(editData: EditData<ITEM_TYPE>) {
  if (!item.value) return;
  const rollbackData = { ...item.value };
  item.value = { ...item.value, ...editData };
  try {
    item.value = await api.users.update(rollbackData.id, editData as Record<string, any>);
    showNotificationToast({ headline: `User saved`, type: "success" });
  } catch (error: any) {
    item.value = rollbackData;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function setActive(active: boolean) {
  if (!item.value) return;
  busy.value = true;
  try {
    item.value = await api.users.update(item.value.id, { active });
    showNotificationToast({ headline: `Account ${active ? "activated" : "deactivated"}`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    busy.value = false;
  }
}

async function confirmDeactivate() {
  showDeactivateDialog.value = false;
  await setActive(false);
}

// --- platform role ---

const roleOptions = [
  { value: "user" as const, label: "User", icon: UserIcon },
  { value: "admin" as const, label: "Platform admin", icon: ShieldCheckIcon },
];

const currentRole = computed(() => (item.value?.role?.key === "admin" ? "admin" : "user"));

async function setRole(role: "user" | "admin") {
  if (!item.value || role === currentRole.value) return;
  busy.value = true;
  try {
    item.value = await api.users.update(item.value.id, { role });
    showNotificationToast({ headline: role === "admin" ? "Promoted to platform admin" : "Platform admin removed", type: "success" });
  } catch (error: any) {
    // 409 is the self-change guard; surface the server's reason rather than a modal.
    showNotificationToast({ headline: error?.response?.data?.message ?? "Could not change the role", type: "error" });
  } finally {
    busy.value = false;
  }
}

// --- password reset ---

const newPassword = ref("");
const newPasswordConfirm = ref("");
const requireChange = ref(true);
const passwordError = ref("");
const pwReqs = ref<{ allMet: boolean } | null>(null);

const canSubmitPassword = computed(() => !!pwReqs.value?.allMet && newPassword.value === newPasswordConfirm.value);

async function submitPassword() {
  if (!item.value) return;
  passwordError.value = "";
  if (newPassword.value !== newPasswordConfirm.value) {
    passwordError.value = "The two passwords do not match.";
    return;
  }
  busy.value = true;
  try {
    item.value = await api.users.update(item.value.id, { password: newPassword.value, forcePasswordChange: requireChange.value });
    newPassword.value = "";
    newPasswordConfirm.value = "";
    showNotificationToast({ headline: `Password updated`, type: "success" });
  } catch (error: any) {
    passwordError.value = error?.response?.data?.message ?? "Could not set the password.";
  } finally {
    busy.value = false;
  }
}

watch(route, loadItem);
onMounted(loadItem);
</script>
