<template>
  <main class="py-8">
    <div class="mx-auto max-w-3xl space-y-12 lg:mx-0">
      <div>
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-base font-semibold leading-7 text-primary-900 dark:text-white">Hotkeys</h2>
            <p class="mt-1 text-sm leading-6 text-primary-500 dark:text-primary-400">
              Customize the keyboard shortcuts. Changes apply after saving; reset restores the system default.
            </p>
          </div>
          <button
            type="button"
            @click="resetAll"
            class="shrink-0 rounded-md border border-primary-200 px-3 py-1.5 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:bg-primary-100 dark:border-primary-700 dark:text-primary-200 dark:hover:bg-primary-800"
          >
            Reset all to defaults
          </button>
        </div>
      </div>

      <section v-for="section in sections" :key="section.context">
        <h3 class="label-mono mb-3 text-primary-500 dark:text-primary-400">{{ section.label }}</h3>
        <ul class="divide-y divide-primary-100 rounded-lg border border-primary-200 dark:divide-primary-800/60 dark:border-primary-800">
          <li v-for="entry in section.entries" :key="entry.id" class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
            <span class="flex items-center gap-2 text-sm text-primary-700 dark:text-primary-200">
              {{ entry.label }}
              <span v-if="entry.customized" class="rounded-full bg-accent-500/15 px-2 py-0.5 text-xs font-medium text-accent-700 dark:text-accent-300">custom</span>
            </span>
            <span class="flex items-center gap-1.5">
              <span v-if="entry.keys.length === 0" class="text-xs italic text-primary-400 dark:text-primary-500">not bound</span>
              <kbd
                v-for="key in entry.keys"
                :key="key"
                class="font-data group inline-flex items-center gap-1 rounded-lg border border-primary-200 bg-primary-100 px-2 py-1 text-xs font-semibold text-primary-700 dark:border-primary-700 dark:bg-primary-800 dark:text-primary-200"
              >
                {{ formatCombo(key) }}
                <button type="button" @click="removeActionKey(entry.id, key)" class="text-primary-400 transition-colors hover:text-danger-500" title="Remove this key">
                  <XMarkIcon class="h-3 w-3" />
                </button>
              </kbd>
              <button
                type="button"
                @click="startActionCapture(entry.id)"
                :class="[
                  'rounded-md border border-dashed px-2 py-1 text-xs font-medium transition-colors',
                  capture?.type === 'action' && capture.actionId === entry.id
                    ? 'border-accent-500 text-accent-600 dark:text-accent-300'
                    : 'border-primary-300 text-primary-500 hover:border-primary-400 hover:text-primary-700 dark:border-primary-600 dark:text-primary-400 dark:hover:text-primary-200',
                ]"
              >
                {{ capture?.type === "action" && capture.actionId === entry.id ? "Press keys… (Esc cancels)" : "+ key" }}
              </button>
              <button
                v-if="entry.customized"
                type="button"
                @click="resetAction(entry.id)"
                class="rounded-md p-1 text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-200"
                title="Reset to default"
              >
                <ArrowUturnLeftIcon class="h-4 w-4" />
              </button>
            </span>
          </li>
        </ul>
      </section>

      <section>
        <h3 class="label-mono mb-1 text-primary-500 dark:text-primary-400">Tag hotkeys</h3>
        <p class="mb-3 text-sm text-primary-500 dark:text-primary-400">
          A tag hotkey toggles the named tag on the current image — assigned when missing, removed when present. Tags are matched by name, so a binding works in every
          project that has a tag of that name.
        </p>
        <ul class="divide-y divide-primary-100 rounded-lg border border-primary-200 dark:divide-primary-800/60 dark:border-primary-800">
          <li v-for="entry in tagEntries" :key="entry.combo" class="flex flex-wrap items-center gap-3 px-4 py-3">
            <kbd class="font-data rounded-lg border border-primary-200 bg-primary-100 px-2 py-1 text-xs font-semibold text-primary-700 dark:border-primary-700 dark:bg-primary-800 dark:text-primary-200">
              {{ formatCombo(entry.combo) }}
            </kbd>
            <span class="text-sm text-primary-500 dark:text-primary-400">toggles tag</span>
            <input
              type="text"
              :value="entry.tagName"
              @input="setTagName(entry.combo, ($event.target as HTMLInputElement).value)"
              placeholder="tag name"
              class="h-8 w-40 rounded-md border border-primary-200 bg-surface px-2.5 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100"
            />
            <button
              type="button"
              @click="removeTagBinding(entry.combo)"
              class="ml-auto rounded-md p-1 text-primary-400 transition-colors hover:bg-primary-100 hover:text-danger-500 dark:hover:bg-primary-800"
              title="Remove tag hotkey"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </li>
          <li v-if="tagEntries.length === 0" class="px-4 py-3 text-sm italic text-primary-400 dark:text-primary-500">No tag hotkeys configured.</li>
          <li class="px-4 py-3">
            <button
              type="button"
              @click="startTagCapture"
              :class="[
                'rounded-md border border-dashed px-2.5 py-1.5 text-xs font-medium transition-colors',
                capture?.type === 'tag'
                  ? 'border-accent-500 text-accent-600 dark:text-accent-300'
                  : 'border-primary-300 text-primary-500 hover:border-primary-400 hover:text-primary-700 dark:border-primary-600 dark:text-primary-400 dark:hover:text-primary-200',
              ]"
            >
              {{ capture?.type === "tag" ? "Press keys… (Esc cancels)" : "+ Add tag hotkey" }}
            </button>
          </li>
        </ul>
      </section>

      <div class="flex items-center gap-4 border-t border-primary-200 pt-6 dark:border-primary-800">
        <button
          type="button"
          :disabled="!dirty"
          @click="save"
          class="rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-accent-500 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Save changes
        </button>
        <span v-if="dirty" class="text-sm text-primary-500 dark:text-primary-400">Unsaved changes</span>
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { ArrowUturnLeftIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { UserHotkeys } from "src/types/api";
import { CONTEXT_LABELS, DEFAULT_TAG_BINDINGS, HOTKEY_ACTIONS, HotkeyContext, comboFromEvent, formatCombo } from "src/util/hotkeys";

const route = useRoute();
const userStore = useUserStore();

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

// draft.bindings holds only overrides (empty list = unbound);
// draft.tagBindings null = system defaults, non-null replaces them entirely.
const draft = reactive<{ bindings: Record<string, string[]>; tagBindings: Record<string, string> | null }>({
  bindings: {},
  tagBindings: null,
});
const savedSnapshot = ref("");

const userId = computed(() => `${route.params.userid}`);

async function loadItem() {
  try {
    const user = await api.users.get(userId.value);
    draft.bindings = { ...(user.hotkeys?.bindings ?? {}) };
    draft.tagBindings = user.hotkeys?.tagBindings ? { ...user.hotkeys.tagBindings } : null;
    savedSnapshot.value = snapshot();
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
watch(route, loadItem);
onMounted(loadItem);

function snapshot(): string {
  return JSON.stringify({ bindings: draft.bindings, tagBindings: draft.tagBindings });
}
const dirty = computed(() => snapshot() !== savedSnapshot.value);

const DISPLAY_CONTEXTS: HotkeyContext[] = ["global", "images", "tagging-dialog"];

const sections = computed(() =>
  DISPLAY_CONTEXTS.map((context) => ({
    context,
    label: CONTEXT_LABELS[context],
    entries: HOTKEY_ACTIONS.filter((a) => a.context === context).map((action) => ({
      id: action.id,
      label: action.label,
      keys: draft.bindings[action.id] ?? action.defaultKeys,
      customized: action.id in draft.bindings,
    })),
  })).filter((section) => section.entries.length > 0),
);

const effectiveTagDraft = computed(() => draft.tagBindings ?? DEFAULT_TAG_BINDINGS);
const tagEntries = computed(() => Object.entries(effectiveTagDraft.value).map(([combo, tagName]) => ({ combo, tagName })));

function entryKeys(actionId: string): string[] {
  return draft.bindings[actionId] ?? HOTKEY_ACTIONS.find((a) => a.id === actionId)?.defaultKeys ?? [];
}

function removeActionKey(actionId: string, key: string) {
  draft.bindings[actionId] = entryKeys(actionId).filter((k) => k !== key);
}

function resetAction(actionId: string) {
  delete draft.bindings[actionId];
}

function resetAll() {
  draft.bindings = {};
  draft.tagBindings = null;
}

// materialize the tag defaults into the draft before the first tag edit
function ensureTagDraft(): Record<string, string> {
  if (!draft.tagBindings) {
    draft.tagBindings = { ...DEFAULT_TAG_BINDINGS };
  }
  return draft.tagBindings;
}

function setTagName(combo: string, tagName: string) {
  ensureTagDraft()[combo] = tagName;
}

function removeTagBinding(combo: string) {
  delete ensureTagDraft()[combo];
}

// combo already used by an action of the same context / global, or a tag hotkey?
function conflictLabel(combo: string, context: HotkeyContext): string | null {
  for (const action of HOTKEY_ACTIONS) {
    if (action.context !== context && action.context !== "global" && context !== "global") continue;
    if (entryKeys(action.id).includes(combo)) return action.label;
  }
  if (context === "images" || context === "global") {
    const tagName = effectiveTagDraft.value[combo];
    if (tagName) return `tag hotkey '${tagName}'`;
  }
  return null;
}

// key capture: one window-level capture-phase listener, Esc cancels
type Capture = { type: "action"; actionId: string } | { type: "tag" };
const capture = ref<Capture | null>(null);

function onCaptureKeydown(event: KeyboardEvent) {
  event.preventDefault();
  event.stopPropagation();
  const combo = comboFromEvent(event);
  if (!combo) return; // bare modifier — keep listening
  const active = capture.value;
  stopCapture();
  if (!active || event.key === "Escape") return;

  const context = active.type === "action" ? (HOTKEY_ACTIONS.find((a) => a.id === active.actionId)?.context ?? "global") : "images";
  const conflict = conflictLabel(combo, context);
  if (conflict) {
    showNotificationToast({ headline: `'${formatCombo(combo)}' is already bound to ${conflict}`, type: "warning" });
    return;
  }
  if (active.type === "action") {
    const keys = entryKeys(active.actionId);
    if (!keys.includes(combo)) {
      draft.bindings[active.actionId] = [...keys, combo];
    }
  } else {
    const tags = ensureTagDraft();
    if (!(combo in tags)) tags[combo] = "";
  }
}

function startActionCapture(actionId: string) {
  startCapture({ type: "action", actionId });
}
function startTagCapture() {
  startCapture({ type: "tag" });
}
function startCapture(target: Capture) {
  stopCapture();
  capture.value = target;
  window.addEventListener("keydown", onCaptureKeydown, { capture: true });
}
function stopCapture() {
  capture.value = null;
  window.removeEventListener("keydown", onCaptureKeydown, { capture: true });
}
onUnmounted(stopCapture);

async function save() {
  const payload: UserHotkeys = { bindings: draft.bindings, tagBindings: draft.tagBindings };
  if (draft.tagBindings && Object.values(draft.tagBindings).some((name) => name.trim() === "")) {
    showNotificationToast({ headline: "Every tag hotkey needs a tag name", type: "warning" });
    return;
  }
  try {
    await api.users.update(userId.value, { hotkeys: payload });
    savedSnapshot.value = snapshot();
    showNotificationToast({ headline: "Hotkeys saved", type: "success" });
    if (userStore.user?.id === userId.value) {
      await userStore.load(); // hotkeys apply immediately for the own session
    }
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
