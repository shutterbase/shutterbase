<template>
  <div v-if="typeof passwordScore === 'number' && passwordScore !== -1" class="row">
    <q-linear-progress rounded size="20px" :value="passwordScore / 4" :color="passwordScoreColor" class="q-ma-sm">
      <div class="absolute-full flex flex-center">
        <q-badge color="white" text-color="black" :label="passwordScoreLabel" />
      </div>
    </q-linear-progress>
  </div>
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";
interface Props {
  passwordScore: number;
}

const props = defineProps({ passwordScore: Number });
const { passwordScore } = toRefs(props);

const passwordScoreColor = computed(() => {
  switch (passwordScore?.value) {
    case -1:
      return "grey";
    case 0:
      return "red";
    case 1:
      return "red";
    case 2:
      return "orange";
    case 3:
      return "yellow";
    case 4:
      return "green";
    default:
      return "blue";
  }
});

const passwordScoreLabel = computed(() => {
  switch (passwordScore?.value) {
    case -1:
      return "No password";
    case 0:
      return "Very weak password";
    case 1:
      return "Weak password";
    case 2:
      return "Weak password";
    case 3:
      return "Good password";
    case 4:
      return "Strong password";
    default:
      return "Unknown";
  }
});
</script>
