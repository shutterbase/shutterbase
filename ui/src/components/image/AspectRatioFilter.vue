<template>
  <div class="aspect-ratio-filter">
    <label for="aspectRatioInput" class="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-100">Aspect Ratio</label>
    <button id="aspectRatioInput" @click="cycleState" @mouseenter="onHover" @mouseleave="onHoverEnd" class="aspect-ratio-button" :class="{ 'is-hovered': isHovered }">
      <div class="icon-container">
        <div class="icon" :class="[`icon-${currentState}`, { 'animate-bounce': isAnimating }, { 'animate-hover': isHovered }]">
          <!-- Square icon -->
          <div v-if="currentState === 'neutral'" class="square-icon"></div>
          <!-- Portrait icon -->
          <div v-else-if="currentState === 'portrait'" class="portrait-icon"></div>
          <!-- Landscape icon -->
          <div v-else-if="currentState === 'landscape'" class="landscape-icon"></div>
        </div>
      </div>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

export type AspectRatioState = "neutral" | "portrait" | "landscape";

interface Props {
  initialState?: AspectRatioState;
}

const props = withDefaults(defineProps<Props>(), {
  initialState: "neutral",
});

const emit = defineEmits<{
  stateChanged: [AspectRatioState];
}>();

const currentState = ref<AspectRatioState>(props.initialState);
const isHovered = ref(false);
const isAnimating = ref(false);

const stateOrder: AspectRatioState[] = ["neutral", "portrait", "landscape"];

const cycleState = () => {
  const currentIndex = stateOrder.indexOf(currentState.value);
  const nextIndex = (currentIndex + 1) % stateOrder.length;
  currentState.value = stateOrder[nextIndex];

  // Trigger bounce animation
  isAnimating.value = true;
  setTimeout(() => {
    isAnimating.value = false;
  }, 300);

  emit("stateChanged", currentState.value);
};

const onHover = () => {
  isHovered.value = true;
};

const onHoverEnd = () => {
  isHovered.value = false;
};

const getTooltip = computed(() => {
  switch (currentState.value) {
    case "neutral":
      return "Filter by aspect ratio - Click for portrait (9:16)";
    case "portrait":
      return "Portrait mode active - Click for landscape (16:9)";
    case "landscape":
      return "Landscape mode active - Click to clear filter";
    default:
      return "Filter by aspect ratio";
  }
});
</script>

<style scoped>
.aspect-ratio-filter {
  @apply text-center;
  @apply mt-1;
}

.aspect-ratio-button {
  @apply relative p-2 rounded-md transition-all duration-200 ease-in-out;
  @apply hover:bg-gray-100 dark:hover:bg-gray-700;
  @apply focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2;
  @apply active:scale-95;
}

.icon-container {
  @apply w-6 h-6 flex items-center justify-center;
}

.icon {
  @apply transition-all duration-200 ease-in-out;
  @apply transform-gpu;
}

.animate-bounce {
  animation: bounceFilter 0.3s ease-in-out;
}

.animate-hover {
  @apply scale-110;
}

/* Square icon (neutral state) */
.square-icon {
  @apply w-4 h-4 border-2 border-gray-600 dark:border-gray-400;
  @apply bg-transparent rounded-sm;
  transition: all 0.2s ease-in-out;
}

.is-hovered .square-icon {
  /* Slightly taller on hover to indicate next state will be portrait */
  @apply w-3 h-4;
}

/* Portrait icon (9:16 ratio) */
.portrait-icon {
  @apply w-3 h-5 border-2 border-gray-600 dark:border-gray-400;
  @apply bg-gray-600 dark:bg-gray-400 rounded-sm;
  transition: all 0.2s ease-in-out;
}

.is-hovered .portrait-icon {
  /* Wider on hover to indicate next state will be landscape */
  @apply w-3 h-4;
}

/* Landscape icon (16:9 ratio) */
.landscape-icon {
  @apply w-5 h-3 border-2 border-gray-600 dark:border-gray-400;
  @apply bg-gray-600 dark:bg-gray-400 rounded-sm;
  transition: all 0.2s ease-in-out;
}

.is-hovered .landscape-icon {
  /* Back to square on hover to indicate next state will be neutral */
  @apply w-4 h-3;
}

/* Custom bounce animation */
@keyframes bounceFilter {
  0%,
  100% {
    transform: scale(1) translateY(0);
  }
  25% {
    transform: scale(1.1) translateY(-2px);
  }
  50% {
    transform: scale(1.05) translateY(-1px);
  }
  75% {
    transform: scale(1.02) translateY(-0.5px);
  }
}

/* Active state indicators */
.icon-neutral .square-icon {
  @apply border-gray-400 dark:border-gray-500;
}

.icon-portrait .portrait-icon {
  @apply border-primary-500 bg-primary-500;
}

.icon-landscape .landscape-icon {
  @apply border-primary-500 bg-primary-500;
}
</style>
