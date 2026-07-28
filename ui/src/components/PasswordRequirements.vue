<!--
  Live password-requirements checklist. Rules mirror the backend exactly
  (api §4.12: min 8 + uppercase + lowercase + digit — no special char, min length
  fixed at 8). Parent gates its submit button on the exposed `allMet`.
-->
<template>
  <ul v-if="alwaysVisible || password" class="mt-3 space-y-1.5">
    <li v-for="req in requirements" :key="req.key" class="flex items-center gap-2 text-xs">
      <component :is="req.met ? CheckCircleIcon : XCircleIcon" class="h-4 w-4 flex-shrink-0" :class="req.met ? 'text-emerald-500' : 'text-primary-300 dark:text-primary-600'" />
      <span :class="req.met ? 'text-emerald-600 dark:text-emerald-400' : 'text-primary-500 dark:text-primary-400'">{{ req.label }}</span>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { CheckCircleIcon, XCircleIcon } from "@heroicons/vue/20/solid";

const props = withDefaults(defineProps<{ password: string; alwaysVisible?: boolean }>(), { alwaysVisible: false });

const requirements = computed(() => [
  { key: "length", label: "At least 8 characters", met: props.password.length >= 8 },
  { key: "upper", label: "One uppercase letter", met: /[A-Z]/.test(props.password) },
  { key: "lower", label: "One lowercase letter", met: /[a-z]/.test(props.password) },
  { key: "digit", label: "One number", met: /\d/.test(props.password) },
]);

const allMet = computed(() => requirements.value.every((r) => r.met));
defineExpose({ allMet });
</script>
