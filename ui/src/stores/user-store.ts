import { defineStore } from "pinia";
import { useStorage } from "@vueuse/core";

export const useUserStore = defineStore("user", {
  state: () => ({
    foo: useStorage("foo", "bar"),
  }),
  getters: {},
  actions: {},
});
