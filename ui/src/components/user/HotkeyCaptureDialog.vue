<template>
  <TransitionRoot as="template" :show="show">
    <Dialog as="div" class="relative z-10" @close="emit('closed')">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-gray-500 dark:bg-gray-900 bg-opacity-75 dark:bg-opacity-75 transition-opacity"></div>
      </TransitionChild>

      <div class="fixed inset-0 z-10 w-screen overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enter-to="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 translate-y-0 sm:scale-100"
            leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel class="relative transform overflow-hidden rounded-lg bg-white dark:!bg-gray-800 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-xl">
              <div class="bg-white dark:!bg-gray-800 px-6 py-6">
                <div class="sm:flex sm:items-start">
                  <div class="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-secondary-100 dark:bg-secondary-900 sm:mx-0 sm:h-10 sm:w-10">
                    <KeyIcon class="h-6 w-6 text-secondary-600 dark:text-secondary-300" aria-hidden="true" />
                  </div>
                  <div class="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left w-full">
                    <DialogTitle as="h3" class="text-base font-semibold leading-6 text-primary-900 dark:text-primary-300"> Set Hotkey </DialogTitle>
                    <div class="mt-4 space-y-4">
                      <p class="text-sm text-primary-600 dark:text-primary-400">
                        Press the desired key combination now. Supported modifiers: CTRL, SHIFT, ALT, META. Click Save to confirm.
                      </p>
                      <div
                        class="border border-dashed border-primary-400 dark:border-primary-600 rounded-md h-28 flex items-center justify-center select-none"
                        tabindex="0"
                        @keydown.prevent.stop="handleKeyDown"
                        ref="captureBox"
                      >
                        <span v-if="currentHotkey" class="text-2xl font-mono">{{ currentHotkey }}</span>
                        <span v-else class="text-primary-400 dark:text-primary-600">Waiting for input...</span>
                      </div>
                      <div v-if="initialHotkey && initialHotkey !== currentHotkey" class="text-xs text-primary-500 dark:text-primary-400">Previous: {{ initialHotkey }}</div>
                      <div v-if="errorMessage" class="text-sm text-error-600">
                        {{ errorMessage }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="bg-gray-50 dark:bg-gray-800 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
                <button
                  type="button"
                  class="bg-secondary-600 hover:bg-secondary-500 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto disabled:opacity-40 disabled:cursor-not-allowed"
                  :disabled="!currentHotkeyValid"
                  @click="save"
                >
                  Save
                </button>
                <button
                  type="button"
                  class="mt-3 inline-flex w-full justify-center rounded-md bg-white dark:!bg-gray-800 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:mt-0 sm:w-auto"
                  @click="emit('closed')"
                >
                  Cancel
                </button>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { KeyIcon } from "@heroicons/vue/24/outline";
import { onMounted, onUnmounted, ref, watch, computed } from "vue";
import { emitter } from "boot/mitt";

interface Props {
  show: boolean;
  initialHotkey?: string | null;
}

const props = withDefaults(defineProps<Props>(), {
  initialHotkey: null,
});

const emit = defineEmits<{
  closed: [];
  save: [string];
}>();

const currentHotkey = ref<string>("");
const errorMessage = ref<string>("");

const captureBox = ref<HTMLElement | null>(null);

const currentHotkeyValid = computed(() => currentHotkey.value.trim().length > 0);

watch(
  () => props.show,
  (v) => {
    if (v) {
      emitter.emit("block-hotkeys");
      currentHotkey.value = "";
      errorMessage.value = "";
      setTimeout(() => {
        captureBox.value?.focus();
      }, 50);
    } else {
      emitter.emit("unblock-hotkeys");
    }
  }
);

function normalizeKey(key: string): string {
  if (key === " ") return "SPACE";
  if (key === "Escape") return "ESC";
  if (key === "Enter") return "ENTER";
  if (key === "ArrowUp") return "ArrowUp";
  if (key === "ArrowDown") return "ArrowDown";
  if (key === "ArrowLeft") return "ArrowLeft";
  if (key === "ArrowRight") return "ArrowRight";
  if (key.length === 1) {
    // letters lower-case (style used in defaults for single letters)
    return /[a-zA-Z]/.test(key) ? key.toLowerCase() : key.toUpperCase();
  }
  return key.toUpperCase();
}

function buildHotkey(e: KeyboardEvent): string | null {
  // Ignore pure modifier presses (no printable / control key)
  const pureModifier = ["Shift", "Control", "Alt", "Meta"].includes(e.key);
  if (pureModifier) {
    return null;
  }
  const parts: string[] = [];
  if (e.ctrlKey) parts.push("CTRL");
  if (e.shiftKey) parts.push("SHIFT");
  if (e.altKey) parts.push("ALT");
  if (e.metaKey) parts.push("META");

  // Use physical key code for digits so SHIFT+1 shows as SHIFT+1 (not SHIFT+!)
  let base: string;
  if (e.code.startsWith("Digit")) {
    base = e.code.substring(5); // "Digit1" -> "1"
  } else {
    base = normalizeKey(e.key);
  }

  parts.push(base);
  return parts.join("+");
}

function handleKeyDown(e: KeyboardEvent) {
  const hk = buildHotkey(e);
  if (hk) {
    currentHotkey.value = hk;
    errorMessage.value = "";
  } else {
    // still waiting for a non-modifier
  }
}

function globalKeyListener(e: KeyboardEvent) {
  if (!props.show) return;
  handleKeyDown(e);
  e.preventDefault();
  e.stopPropagation();
}

onMounted(() => {
  window.addEventListener("keydown", globalKeyListener, { capture: true });
});

onUnmounted(() => {
  emitter.emit("unblock-hotkeys");
  window.removeEventListener("keydown", globalKeyListener, { capture: true } as any);
});

function save() {
  if (!currentHotkeyValid.value) {
    errorMessage.value = "Please press a key combination first";
    return;
  }
  emit("save", currentHotkey.value);
}
</script>
