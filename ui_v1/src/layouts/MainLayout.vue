<template>
  <q-layout view="lHh Lpr lFf">
    <q-header bordered class="bg-white text-primary">
      <q-toolbar>
        <q-btn v-if="loggedIn" flat round dense icon="menu" class="header-button" data-test="drawerToggleButton" @click="emitter.emit('toggleDrawer')" />
        <div v-else class="header-button"><!-- spacer for alignment when logged out --></div>
        <q-toolbar-title class="text-center" style="cursor: pointer" @click="toHome">
          <!--<q-avatar>
            <img src="https://cdn.quasar.dev/logo-v2/svg/logo.svg" />
          </q-avatar>-->
          shutterbase
        </q-toolbar-title>
        <q-btn v-if="loggedIn" flat round dense icon="logout" data-test="headerLogoutButton" @click="logout" />
        <q-btn v-else flat round dense icon="person" to="/login" />
      </q-toolbar>
    </q-header>

    <drawer />
    <q-footer bordered class="bg-white text-primary">
      <div class="text-right q-pa-md">
        <router-link class="footer-link" to="/help">Help</router-link>
        <router-link class="footer-link" to="/privacy">Privacy</router-link>
        <router-link class="footer-link" to="/terms">Terms</router-link>
        <router-link class="footer-link" to="/legal">Legal</router-link>
      </div>
    </q-footer>
    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { emitter } from "../boot/mitt";
import Drawer from "../components/Drawer.vue";
import { useLoginStore } from "../stores/login-store";

const router = useRouter();
const loginStore = useLoginStore();
const { loggedIn } = storeToRefs(loginStore);

loginStore.$subscribe((mutation, state) => {
  loggedIn.value = state.loggedIn;
});

const logout = () => {
  router.push("/logout");
};

const toHome = () => {
  router.push("/");
};
</script>

<style lang="sass" scoped>
.footer-link
  padding: 0 10px
  color: grey
  text-decoration: none
  &:hover
    text-decoration: underline

.header-button
  width: 34px
  height: 34px
</style>
