<template>
  <div v-if="open" class="relative z-10" role="dialog" aria-modal="true">
    <div class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity" @click="open = false"></div>

    <div class="fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-16">
      <div
        class="mx-auto max-w-2xl transform overflow-hidden rounded-xl border border-primary-200 bg-surface shadow-2xl transition-all dark:border-primary-800 dark:bg-surface-dark"
      >
        <div class="flex items-center justify-between border-b border-primary-200 px-6 py-4 dark:border-primary-800">
          <h2 class="text-base font-semibold text-primary-900 dark:text-white">Keyboard shortcuts</h2>
          <button
            type="button"
            @click="open = false"
            class="rounded-md p-1 text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-200"
          >
            <span class="sr-only">Close</span>
            <XMarkIcon class="h-5 w-5" />
          </button>
        </div>

        <div class="max-h-[65vh] space-y-8 overflow-y-auto px-6 py-5">
          <section v-for="section in sections" :key="section.context">
            <h3 class="label-mono mb-3 text-primary-500 dark:text-primary-400">{{ section.label }}</h3>
            <ul class="divide-y divide-primary-100 dark:divide-primary-800/60">
              <li v-for="entry in section.entries" :key="entry.id" class="flex items-center justify-between gap-4 py-2">
                <span class="flex items-center gap-2 text-sm text-primary-700 dark:text-primary-200">
                  {{ entry.label }}
                  <span
                    v-if="entry.customized"
                    class="rounded-full bg-accent-500/15 px-2 py-0.5 text-xs font-medium text-accent-700 dark:text-accent-300"
                    title="Customized binding"
                    >custom</span
                  >
                </span>
                <span class="flex shrink-0 items-center gap-1.5">
                  <kbd
                    v-for="key in entry.keys"
                    :key="key"
                    class="font-data rounded-lg border border-primary-200 bg-primary-100 px-2 py-1 text-xs font-semibold text-primary-700 dark:border-primary-700 dark:bg-primary-800 dark:text-primary-200"
                    >{{ formatCombo(key) }}</kbd
                  >
                  <span v-if="entry.keys.length === 0" class="text-xs italic text-primary-400 dark:text-primary-500">not bound</span>
                </span>
              </li>
            </ul>
          </section>

          <section>
            <h3 class="label-mono mb-1 text-primary-500 dark:text-primary-400">Tag hotkeys</h3>
            <p class="mb-3 text-xs text-primary-500 dark:text-primary-400">Toggle the tag on the current image — assigned when missing, removed when present.</p>
            <ul v-if="tagEntries.length" class="divide-y divide-primary-100 dark:divide-primary-800/60">
              <li v-for="entry in tagEntries" :key="entry.combo" class="flex items-center justify-between gap-4 py-2">
                <span class="text-sm text-primary-700 dark:text-primary-200">
                  Toggle tag <b>{{ entry.tagName }}</b>
                </span>
                <kbd
                  class="font-data rounded-lg border border-primary-200 bg-primary-100 px-2 py-1 text-xs font-semibold text-primary-700 dark:border-primary-700 dark:bg-primary-800 dark:text-primary-200"
                  >{{ formatCombo(entry.combo) }}</kbd
                >
              </li>
            </ul>
            <p v-else class="text-sm italic text-primary-400 dark:text-primary-500">No tag hotkeys configured.</p>
          </section>
        </div>

        <div class="border-t border-primary-200 px-6 py-4 dark:border-primary-800">
          <a
            href="#"
            @click.prevent="goToSettings"
            class="text-sm font-semibold text-accent-600 underline hover:text-accent-500 dark:text-accent-400"
          >
            Customize hotkeys in your profile →
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { XMarkIcon } from "@heroicons/vue/24/outline";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { emitter } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import {
  CONTEXT_LABELS,
  HOTKEY_ACTIONS,
  HotkeyContext,
  actionKeys,
  effectiveTagBindings,
  formatCombo,
  pushHotkeyContext,
  useHotkeyAction,
} from "src/util/hotkeys";

const router = useRouter();
const userStore = useUserStore();

const open = ref(false);

useHotkeyAction("help.toggle", () => (open.value = !open.value));
useHotkeyAction("help.close", () => (open.value = false));

function show() {
  open.value = true;
}
onMounted(() => emitter.on("show-hotkey-help", show));
onUnmounted(() => {
  emitter.off("show-hotkey-help", show);
  popContext?.();
  popContext = null;
});

// while open, the help context suppresses page hotkeys (Escape closes)
let popContext: (() => void) | null = null;
watch(open, (isOpen) => {
  if (isOpen && !popContext) {
    popContext = pushHotkeyContext("help");
  } else if (!isOpen && popContext) {
    popContext();
    popContext = null;
  }
});

const DISPLAY_CONTEXTS: HotkeyContext[] = ["global", "images", "tagging-dialog"];

const sections = computed(() => {
  const config = userStore.user?.hotkeys;
  return DISPLAY_CONTEXTS.map((context) => ({
    context,
    label: CONTEXT_LABELS[context],
    entries: HOTKEY_ACTIONS.filter((a) => a.context === context).map((action) => ({
      id: action.id,
      label: action.label,
      keys: actionKeys(config, action.id),
      customized: !!config?.bindings?.[action.id],
    })),
  })).filter((section) => section.entries.length > 0);
});

const tagEntries = computed(() =>
  Object.entries(effectiveTagBindings(userStore.user?.hotkeys)).map(([combo, tagName]) => ({ combo, tagName })),
);

function goToSettings() {
  open.value = false;
  if (userStore.user) {
    router.push(`/users/${userStore.user.id}/hotkeys`);
  }
}
</script>
