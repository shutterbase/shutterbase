<template>
  <!-- The over-assignment celebration (transcript 08:07): a purple rupture
       wave rippling from the item across the whole screen, Ultracode-style.
       Pure CSS keyframes, no dependency. -->
  <Teleport to="body">
    <div v-if="active" class="pointer-events-none fixed inset-0 z-[60] overflow-hidden" aria-hidden="true">
      <span class="wave" :style="originStyle"></span>
      <span class="wave wave-late" :style="originStyle"></span>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

const active = ref(false);
const origin = ref({ x: 0.5, y: 0.5 });
let timer: ReturnType<typeof setTimeout> | undefined;

const originStyle = computed(() => ({
  left: `${origin.value.x * 100}%`,
  top: `${origin.value.y * 100}%`,
}));

// trigger plays the wave once, radiating from viewport-relative coordinates
// (0..1). Retriggering restarts the animation.
function trigger(x = 0.5, y = 0.5) {
  origin.value = { x, y };
  active.value = false;
  requestAnimationFrame(() => {
    active.value = true;
    clearTimeout(timer);
    timer = setTimeout(() => (active.value = false), 1600);
  });
}

defineExpose({ trigger });
</script>

<style scoped>
.wave {
  position: absolute;
  width: 24px;
  height: 24px;
  transform: translate(-50%, -50%) scale(0);
  border-radius: 9999px;
  border: 3px solid rgb(167 139 250 / 0.9); /* violet-400 */
  box-shadow:
    0 0 40px 10px rgb(139 92 246 / 0.55),
    inset 0 0 30px 6px rgb(139 92 246 / 0.35);
  animation: purple-rupture 1.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
.wave-late {
  animation-delay: 0.15s;
}
@keyframes purple-rupture {
  0% {
    transform: translate(-50%, -50%) scale(0);
    opacity: 1;
  }
  70% {
    opacity: 0.55;
  }
  100% {
    /* 24px * 150 covers > 3000px viewports from any origin */
    transform: translate(-50%, -50%) scale(150);
    opacity: 0;
  }
}
</style>
