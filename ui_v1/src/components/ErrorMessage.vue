<template>
  <q-dialog v-model="show" persistent>
    <q-card>
      <q-card-section>
        <div class="text-h6">{{ title }}</div>
      </q-card-section>
      <q-card-section class="row items-center">
        <q-avatar icon="signal_wifi_off" color="primary" text-color="white" />
        <span class="q-ml-sm">{{ message }}</span>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Ok" color="primary" v-close-popup />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { ErrorMessageOptions } from "src/util/error-message";
import { emitter } from "src/boot/mitt";

const show = ref(false);
const icon = ref("alert");
const title = ref("Error");
const message = ref("An error occurred");

emitter.on("error", (options: ErrorMessageOptions) => {
  show.value = true;
  icon.value = options.icon ?? "alert";
  title.value = options.title;
  message.value = options.message;
});
</script>
