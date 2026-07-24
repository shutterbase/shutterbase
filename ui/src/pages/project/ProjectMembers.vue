<template>
  <div class="mx-auto w-full max-w-7xl">
    <Table dense name="Member" subtitle="People with access to this project." :items="assignments" :columns="columns" :allow-add="canManage" :add-callback="startAddMember" />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <MemberDialog
      :show="showMemberDialog"
      :create="createMember"
      :roles="roles"
      :available-users="availableUsers"
      :member="editMemberData"
      @add="addMember"
      @edit="editRole"
      @closed="showMemberDialog = false"
    />
    <ModalMessage
      :show="showRemoveConfirm"
      :type="MessageType.CONFIRM_WARNING"
      headline="Remove member?"
      :message="removeConfirmMessage"
      confirmText="Remove member"
      cancelText="Cancel"
      @confirmed="confirmRemove"
      @closed="showRemoveConfirm = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import MemberDialog from "src/components/project/MemberDialog.vue";
import { ProjectAssignment, Role, User } from "src/types/api";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";

const route = useRoute();
const userStore = useUserStore();

// A projectAdmin manages the roster of their own project; a global admin manages
// every project's (mirrors authorization.CanManageProjectAssignment).
const canManage = computed(() => userStore.isProjectAdminOrHigher());

const projectId = computed(() => `${route.params.id}`);

const assignments = ref<ProjectAssignment[]>([]);
const roles = ref<Role[]>([]);
const users = ref<User[]>([]);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const showMemberDialog = ref(false);
const createMember = ref(false);
const editMemberData = ref<ProjectAssignment | null>(null);

const showRemoveConfirm = ref(false);
const removeTarget = ref<ProjectAssignment | null>(null);

const availableUsers = computed(() => {
  const assigned = new Set(assignments.value.map((a) => a.user.id));
  return users.value.filter((u) => !assigned.has(u.id));
});

function prettyRole(key: string): string {
  return key.replace(/([A-Z])/g, " $1").replace(/^./, (c) => c.toUpperCase());
}

const columns: TableColumn<ProjectAssignment>[] = [
  { key: "user", label: "Name", formatter: (u) => (u ? `${u.firstName} ${u.lastName}` : "—") },
  { key: ["user", "email"], label: "Email" },
  { key: "role", label: "Role", formatter: (r) => (r ? prettyRole(r.key) : "—") },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "role", label: "Change role", type: TableRowActionType.EDIT, showCallback: () => canManage.value, callback: startEditRole },
      { key: "remove", label: "Remove", type: TableRowActionType.DELETE, showCallback: () => canManage.value, callback: startRemove },
    ],
  },
];

async function loadData() {
  const id = projectId.value;
  if (!id) return;
  try {
    const assignmentsRes = await api.projectAssignments.list({ projectId: id, limit: 500 });
    assignments.value = assignmentsRes.items;
    // Roles + the user picker are only needed for management (admins).
    if (canManage.value) {
      const [rolesRes, usersRes] = await Promise.all([api.roles.list({ limit: 100 }), api.users.list({ limit: 500 })]);
      roles.value = rolesRes.items;
      users.value = usersRes.items;
    }
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function startAddMember() {
  createMember.value = true;
  editMemberData.value = null;
  showMemberDialog.value = true;
}

function startEditRole(assignment: ProjectAssignment) {
  createMember.value = false;
  editMemberData.value = assignment;
  showMemberDialog.value = true;
}

async function addMember(payload: { userId: string; roleId: string }) {
  try {
    const created = await api.projectAssignments.create({ projectId: projectId.value, userId: payload.userId, roleId: payload.roleId });
    assignments.value = [...assignments.value, created];
    showMemberDialog.value = false;
    showNotificationToast({ headline: "Member added", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function editRole(payload: { id: string; roleId: string }) {
  try {
    const updated = await api.projectAssignments.update(payload.id, payload.roleId);
    const i = assignments.value.findIndex((a) => a.id === payload.id);
    if (i !== -1) assignments.value[i] = updated;
    showMemberDialog.value = false;
    showNotificationToast({ headline: "Role updated", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function startRemove(assignment: ProjectAssignment) {
  removeTarget.value = assignment;
  showRemoveConfirm.value = true;
}

const removeConfirmMessage = computed(() => {
  const u = removeTarget.value?.user;
  return u ? `Remove ${u.firstName} ${u.lastName} from this project? They lose access until re-added.` : "";
});

async function confirmRemove() {
  const target = removeTarget.value;
  showRemoveConfirm.value = false;
  if (!target) return;
  try {
    await api.projectAssignments.remove(target.id);
    assignments.value = assignments.value.filter((a) => a.id !== target.id);
    showNotificationToast({ headline: "Member removed", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(route, loadData);
onMounted(loadData);
</script>
