<template>
  <div class="overflow-x-auto">
    <client-only>
      <h3>Total Users: {{ data?.total }}</h3>
      <table class="table table-xs">
        <thead>
          <tr>
            <th>Name</th>
            <th>Role</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in data?.items || []" :key="item.id" class="hover click hover:cursor-pointer">
            <td>{{ item.edges.user.firstName }} {{ item.edges.user.lastName }}</td>
            <td>{{ item.edges.role.description }}</td>
            <td>
              <button class="btn btn-xs btn-primary mr-2" @click="editUserAssignment(item)">Modify</button>
              <button class="btn btn-xs btn-error mr-2" @click="removeUserAssignment(item)">Remove</button>
            </td>
          </tr>
        </tbody>
      </table>
    </client-only>
    <div class="mr-8"><button class="btn btn-secondary" onclick="addUserDialog.showModal()">Add User</button></div>
    <dialog id="addUserDialog" ref="addUserDialog" class="modal">
      <form method="dialog" class="modal-box">
        <h3 class="font-bold text-lg">Add User to project</h3>
        <div class="h-72 border-2 rounded">
          <user-select @selected="userSelected"></user-select>
        </div>
        <select v-model="projectAssignmentRole" class="select select-bordered w-full max-w-xs m-2">
          <option v-for="option in projectAssignmentRoleOptions">{{ option }}</option>
        </select>
        <div class="modal-action">
          <button class="btn" onclick="addUserDialog.close()">Cancel</button>
          <button class="btn" @click="addProjectAssignment">Add</button>
        </div>
      </form>
    </dialog>
    <dialog id="editUserDialog" ref="editUserDialog" class="modal">
      <form method="dialog" class="modal-box">
        <h3 class="font-bold text-lg">Edit project assignment for {{ editProjectAssignment?.edges.user.firstName }} {{ editProjectAssignment?.edges.user.lastName }}</h3>
        <select v-model="editProjectAssignmentRole" class="select select-bordered w-full max-w-xs m-2">
          <option v-for="option in projectAssignmentRoleOptions">{{ option }}</option>
        </select>
        <div class="modal-action">
          <button class="btn" onclick="editUserDialog.close()">Cancel</button>
          <button class="btn" @click="saveProjectAssignment">Save</button>
        </div>
      </form>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { ProjectAssignment } from "~/api/projectAssignment";
import { ref } from "vue";
import { Method, ListResult, API_BASE_URL, getFetchOptions } from "~/api/common";
import { User } from "~/api/user";
const limit = ref(1000);

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

const { data, refresh } = await useFetch<ListResult<ProjectAssignment>>(`/projects/${props.projectId}/assignments`, {
  method: Method.GET,
  baseURL: API_BASE_URL,
  credentials: "include",
  params: {
    limit,
  },
});

async function removeUserAssignment(item: ProjectAssignment) {
  const { data } = await useFetch(`/projects/${props.projectId}/assignments/${item.id}`, getFetchOptions(Method.DELETE, {}));
  await refresh();
}

function userSelected(user: User) {
  projectAssignmentUserId.value = user.id;
}

const addUserDialog = ref<HTMLDialogElement | null>(null);
const projectAssignmentRole = ref("project_viewer");
const projectAssignmentUserId = ref("");
const projectAssignmentRoleOptions = ref(["project_admin", "project_editor", "project_viewer"]);

const editUserDialog = ref<HTMLDialogElement | null>(null);
const editProjectAssignment = ref<ProjectAssignment | null>(null);
const editProjectAssignmentRole = ref("project_viewer");

async function addProjectAssignment() {
  const { data } = await useFetch(
    `/projects/${props.projectId}/assignments`,
    getFetchOptions(Method.POST, { userId: projectAssignmentUserId.value, role: projectAssignmentRole.value })
  );
  await refresh();
  projectAssignmentRole.value = "project_viewer";
  projectAssignmentUserId.value = "";
  addUserDialog.value?.close();
}

async function editUserAssignment(item: ProjectAssignment) {
  editProjectAssignment.value = item;
  editProjectAssignmentRole.value = item.edges.role.key;
  editUserDialog.value?.showModal();
}
async function saveProjectAssignment() {
  const { data } = await useFetch(
    `/projects/${props.projectId}/assignments/${editProjectAssignment.value?.id}`,
    getFetchOptions(Method.PUT, { role: editProjectAssignmentRole.value })
  );
  await refresh();
  editProjectAssignmentRole.value = "";
  editProjectAssignment.value = null;
  editUserDialog.value?.close();
}
</script>
