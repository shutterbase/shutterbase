<template>
  <div class="card bg-white shadow-md p-10">
    <ItemDescriptorLine :item="initialItem" />
    <div class="divider"></div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <label class="label">
          <span class="label-text">First name</span>
        </label>
        <input type="text" placeholder="First name" v-model="firstName" class="input input-bordered w-full max-w-xs" />
      </div>
      <div>
        <label class="label">
          <span class="label-text">Last name</span>
        </label>
        <input type="text" placeholder="Last name" v-model="lastName" class="input input-bordered w-full max-w-xs" />
      </div>
      <div>
        <label class="label">
          <span class="label-text">Copyright Tag</span>
        </label>
        <input type="text" placeholder="Copyright Tag" v-model="copyrightTag" class="input input-bordered w-full max-w-xs" />
      </div>
      <div>
        <label class="label">
          <span class="label-text">Email</span>
        </label>
        <input type="text" placeholder="Email" v-model="email" class="input input-bordered w-full max-w-xs" disabled />
      </div>
      <div class="form-control w-full max-w-xs">
        <label class="label">
          <span class="label-text">Global role</span>
        </label>
        <select class="select select-bordered" :disabled="!store.isAdmin()" v-model="globalRole">
          <option disabled selected>Select role</option>
          <option value="user" :selected="globalRole === 'user'">User</option>
          <option value="admin" :selected="globalRole === 'admin'">Administrator</option>
        </select>
      </div>
      <div class="flex flex-row">
        <div class="basis-1/4">
          <label class="label">
            <span class="label-text">Account active</span>
          </label>
          <input type="checkbox" :disabled="!store.isAdmin()" class="toggle toggle-success" v-model="accountActive" />
        </div>
        <div class="basis-1/4">
          <label class="label">
            <span class="label-text">Email validated</span>
          </label>
          <input type="checkbox" :disabled="!store.isAdmin()" class="toggle toggle-success" v-model="emailValidated" />
        </div>
      </div>
    </div>

    <div class="divider"></div>
    <div class="flex flex-row">
      <div class="mr-8"><button class="btn btn-secondary" @click="navigateToCameras">Cameras</button></div>
      <div class="mr-8"><button class="btn btn-secondary" @click="getApiKey">Get new API KEY</button></div>
      <div class="mr-8"><button class="btn btn-primary" :disabled="!modified" @click="update">Update</button></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { UpdateUserInput, User } from "~/api/user";
import { ApiKey } from "~/api/apiKey";
import { Method, getFetchOptions, getCreatedByString, getUpdatedByString, getDateTimeString } from "~/api/common";

const store = useStore();

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

console.log("props", props);
console.log("props.id", props.id);

const { data: item } = await useFetch(`/users/${props.id}`, getFetchOptions(Method.GET));
const initialItem = item as Ref<User>;

const firstName = ref("");
const lastName = ref("");
const copyrightTag = ref("");
const email = ref("");
const globalRole = ref("");
const accountActive = ref(false);
const emailValidated = ref(false);

function updateEditValues(editItem: Ref<User>) {
  if (!editItem.value) {
    return;
  }
  firstName.value = editItem.value.firstName;
  lastName.value = editItem.value.lastName;
  copyrightTag.value = editItem.value.copyrightTag;
  email.value = editItem.value.email;
  globalRole.value = editItem.value.edges.role?.key;
  accountActive.value = editItem.value.active;
  emailValidated.value = editItem.value.emailValidated;
}

updateEditValues(item as Ref<User>);

const modified = computed(() => {
  if (!initialItem.value) {
    return false;
  }
  return (
    firstName.value !== initialItem.value.firstName ||
    lastName.value !== initialItem.value.lastName ||
    copyrightTag.value !== initialItem.value.copyrightTag ||
    globalRole.value !== initialItem.value.edges.role?.key ||
    accountActive.value !== initialItem.value.active ||
    emailValidated.value !== initialItem.value.emailValidated
  );
});

async function update() {
  const updateData = {} as UpdateUserInput;
  if (firstName.value !== initialItem.value.firstName) {
    updateData.firstName = firstName.value;
  }
  if (lastName.value !== initialItem.value.lastName) {
    updateData.lastName = lastName.value;
  }
  if (copyrightTag.value !== initialItem.value.copyrightTag) {
    updateData.copyrightTag = copyrightTag.value;
  }
  if (globalRole.value !== initialItem.value.edges.role?.key) {
    updateData.role = globalRole.value;
  }
  if (accountActive.value !== initialItem.value.active) {
    updateData.active = accountActive.value;
  }
  if (emailValidated.value !== initialItem.value.emailValidated) {
    updateData.emailValidated = emailValidated.value;
  }

  const { data } = await useFetch(`/users/${props.id}`, getFetchOptions(Method.PUT, updateData));
  const updatedItem = data as Ref<User>;
  if (data) {
    initialItem.value.firstName = updatedItem.value.firstName;
    initialItem.value.lastName = updatedItem.value.lastName;
    initialItem.value.copyrightTag = updatedItem.value.copyrightTag;
    initialItem.value.active = updatedItem.value.active;
    initialItem.value.emailValidated = updatedItem.value.emailValidated;
    initialItem.value.edges.role.key = updatedItem.value.edges.role?.key;

    initialItem.value.updatedAt = updatedItem.value.updatedAt;
    initialItem.value.edges.updatedBy = store.getOwnUser() || ({} as User);

    updateEditValues(initialItem);
  }
}

function navigateToCameras() {
  navigateTo(`/dashboard/users/${props.id}/cameras`);
}

async function getApiKey() {
  const { data } = await useFetch(`/users/${props.id}/api-keys`, getFetchOptions(Method.POST));
  if (data) {
    const apiKey = data as Ref<ApiKey>;
    alert(`API KEY: ${apiKey.value.key}`);
  }
}
</script>
