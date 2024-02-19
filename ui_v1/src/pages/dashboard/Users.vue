<template>
  <q-page class="" ref="pageContainer" id="pageContainer">
    <div class="" style="height: 100%">
      <q-table
        title="Users"
        :rows="items"
        ref="tableRef"
        row-key="id"
        virtual-scroll
        :columns="columns"
        v-model:pagination="pagination"
        :loading="loading"
        :filter="search"
        binary-state-sort
        @request="onRequest"
        @row-click="onRowClick"
        class="sticky-virtscroll-table"
        :style="`max-height: ${tableHeight}px;`"
      >
        <template v-slot:top-right>
          <q-input borderless dense debounce="300" v-model="search" placeholder="Search">
            <template v-slot:append>
              <q-icon name="search" />
            </template>
          </q-input>
        </template>
      </q-table>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getUsers, User } from "src/api/user";
import { useRouter } from "vue-router";
import { toDateTime } from "src/utils/date";
import { emitter } from "src/boot/mitt";

const router = useRouter();

const tableRef = ref();
const items = ref([] as User[]);
const loading = ref(false);
const search = ref("");
const pagination = ref({
  sortBy: "",
  descending: false,
  page: 1,
  rowsPerPage: 25,
  rowsNumber: 10,
});

const columns = [
  // { name: "id", label: "ID", field: "id", align: "left", sortable: true },
  { name: "firstName", label: "First Name", field: "firstName", align: "left", sortable: true },
  { name: "lastName", label: "Last Name", field: "lastName", align: "left", sortable: true },
  { name: "email", label: "Email", field: "email", align: "left", sortable: true },
  { name: "emailValidated", label: "Email Validated", field: "emailValidated", align: "left", sortable: true },
  { name: "active", label: "Active", field: "active", align: "left", sortable: true },
  { name: "createdAt", label: "Created At", field: "createdAt", align: "left", sortable: true, format: (val: string) => toDateTime(val) },
  { name: "updatedAt", label: "Updated At", field: "updatedAt", align: "left", sortable: true, format: (val: string) => toDateTime(val) },
];

const onRequest = async (props: any) => {
  const { page, rowsPerPage, sortBy, descending } = props.pagination;
  const filter = props.filter;

  loading.value = true;
  const limit = rowsPerPage === 0 ? 500 : rowsPerPage;
  const offset = (page - 1) * rowsPerPage;

  const result = await getUsers({ limit, offset, search: search.value, sort: sortBy, order: descending ? "desc" : "asc" });
  if (!result.response.ok) {
    loading.value = false;
    emitter.emit("error", { title: "Error loading users", message: result.response.message });
    return;
  }

  pagination.value.rowsNumber = result.total || 0;
  items.value.splice(0, items.value.length, ...(result.items || []));

  pagination.value.page = page;
  pagination.value.rowsPerPage = rowsPerPage;
  pagination.value.sortBy = sortBy;
  pagination.value.descending = descending;

  loading.value = false;
};

const onRowClick = (event: any, row: any) => {
  const userId = row.id;
  if (typeof userId !== "string") return;
  router.push(`/dashboard/users/${userId}`);
};

const tableHeight = ref(0);
const pageContainer = ref(null);

onMounted(() => {
  tableRef.value.requestServerInteraction();
  const resizeObserver = new ResizeObserver(function () {
    if (pageContainer.value !== null) {
      console.log();
      tableHeight.value = Math.min(window.innerHeight - 51 - 54, 2000);
    }
  });
  //@ts-ignore
  resizeObserver.observe(document.getElementById("pageContainer"));
});
</script>

<style lang="sass" scoped>
.users-screen-card
  min-width: 350px
  @media (max-width: $breakpoint-xs)
    width: 95%
  @media (max-width: $breakpoint-sm)
    max-width: 1000px
  @media (max-width: $breakpoint-md)
    max-width: 1200px
  @media (max-width: $breakpoint-lg)
    max-width: 1200px

.sticky-virtscroll-table
  .q-table__top,
  .q-table__bottom,
  thead tr:first-child th /* bg color is important for th; just specify one */
    background-color: #fff

  thead tr th
    position: sticky
    z-index: 1
  /* this will be the loading indicator */
  thead tr:last-child th
    /* height of all previous header rows */
    top: 48px
  thead tr:first-child th
    top: 0
</style>
