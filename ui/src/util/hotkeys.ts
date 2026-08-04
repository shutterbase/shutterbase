import { onScopeDispose, ref } from "vue";
import { UserHotkeys } from "src/types/api";

// Central hotkey system.
//
// Three layers:
//  1. HOTKEY_ACTIONS — the static registry of everything a hotkey can do
//     (id, label, context, system default keys).
//  2. Per-user config (UserHotkeys, persisted on the backend user) — overrides
//     the default keys per action and maps key combos to image tag names.
//  3. Runtime — components register handlers for action ids; one document-level
//     keydown handler resolves the pressed combo against the effective bindings
//     and dispatches to the registered handlers of the active context.
//
// The resolution helpers are pure (no store imports) so they are unit-testable;
// App.vue injects the user's config via configureHotkeys().

export type HotkeyContext = "global" | "images" | "tagging-dialog" | "help";

export interface HotkeyActionDef {
  id: string;
  label: string;
  context: HotkeyContext;
  defaultKeys: string[];
  // fire even when an input/textarea/contenteditable has focus
  allowInInputs?: boolean;
}

export const HOTKEY_ACTIONS: HotkeyActionDef[] = [
  { id: "help.toggle", label: "Show / hide hotkey help", context: "global", defaultKeys: ["?"] },
  { id: "help.close", label: "Close hotkey help", context: "help", defaultKeys: ["Escape"], allowInInputs: true },
  { id: "images.next-image", label: "Next image", context: "images", defaultKeys: ["ArrowRight", "l"] },
  { id: "images.previous-image", label: "Previous image", context: "images", defaultKeys: ["ArrowLeft", "h"] },
  { id: "images.previous-row", label: "One row up", context: "images", defaultKeys: ["ArrowUp", "k"] },
  { id: "images.next-row", label: "One row down", context: "images", defaultKeys: ["ArrowDown", "j"] },
  { id: "images.toggle-view", label: "Toggle grid / detail view", context: "images", defaultKeys: ["g"] },
  { id: "images.open-tagging", label: "Open tagging dialog", context: "images", defaultKeys: ["t"] },
  { id: "images.repeat-last-tag", label: "Repeat last tag assignment", context: "images", defaultKeys: ["s"] },
  { id: "tagging.close", label: "Close tagging dialog", context: "tagging-dialog", defaultKeys: ["Escape"], allowInInputs: true },
  { id: "tagging.select-next", label: "Highlight next tag", context: "tagging-dialog", defaultKeys: ["ArrowDown"], allowInInputs: true },
  { id: "tagging.select-previous", label: "Highlight previous tag", context: "tagging-dialog", defaultKeys: ["ArrowUp"], allowInInputs: true },
  { id: "tagging.accept-only-result", label: "Accept highlighted tag / close and next", context: "tagging-dialog", defaultKeys: ["Enter"], allowInInputs: true },
  { id: "tagging.accept-1", label: "Accept 1st suggested tag", context: "tagging-dialog", defaultKeys: ["shift+1"], allowInInputs: true },
  { id: "tagging.accept-2", label: "Accept 2nd suggested tag", context: "tagging-dialog", defaultKeys: ["shift+2"], allowInInputs: true },
  { id: "tagging.accept-3", label: "Accept 3rd suggested tag", context: "tagging-dialog", defaultKeys: ["shift+3"], allowInInputs: true },
  { id: "tagging.accept-4", label: "Accept 4th suggested tag", context: "tagging-dialog", defaultKeys: ["shift+4"], allowInInputs: true },
  { id: "tagging.accept-5", label: "Accept 5th suggested tag", context: "tagging-dialog", defaultKeys: ["shift+5"], allowInInputs: true },
];

// Tag hotkeys toggle the named tag on the current image (assign when absent,
// remove when present). Names are used instead of ids so a binding works in
// every project that has a tag of that name.
export const DEFAULT_TAG_BINDINGS: Record<string, string> = { p: "review" };

export const CONTEXT_LABELS: Record<HotkeyContext, string> = {
  global: "Global",
  images: "Image gallery",
  "tagging-dialog": "Tagging dialog",
  help: "Hotkey help",
};

// ---------------------------------------------------------------------------
// Combo normalization
// ---------------------------------------------------------------------------
// Grammar: "[ctrl+][alt+][meta+][shift+]<key>".
// Printable characters carry shift implicitly ("T", "?") so combos stay
// keyboard-layout independent. shift+digit uses the physical digit key
// ("shift+1") so the numeric quick-accept hotkeys work on every layout.

const MODIFIER_KEYS = new Set(["Shift", "Control", "Alt", "Meta"]);

export function comboFromEvent(event: KeyboardEvent): string | null {
  if (MODIFIER_KEYS.has(event.key)) return null;
  const mods: string[] = [];
  if (event.ctrlKey) mods.push("ctrl");
  if (event.altKey) mods.push("alt");
  if (event.metaKey) mods.push("meta");

  let key = event.key === " " ? "Space" : event.key;
  const digit = /^Digit(\d)$/.exec(event.code || "");
  if (event.shiftKey && digit) {
    mods.push("shift");
    key = digit[1];
  } else if (event.shiftKey && key.length > 1) {
    mods.push("shift");
  }
  return [...mods, key].join("+");
}

const KEY_GLYPHS: Record<string, string> = {
  ctrl: "Ctrl",
  alt: "Alt",
  meta: "⌘",
  shift: "Shift",
  ArrowRight: "→",
  ArrowLeft: "←",
  ArrowUp: "↑",
  ArrowDown: "↓",
};

export function formatCombo(combo: string): string {
  // split on "+" separators but keep a literal "+" key intact
  const parts = combo.split(/\+(?=.)/);
  const out: string[] = [];
  for (const part of parts) {
    if (part.length === 1 && part !== part.toLowerCase()) {
      out.push("Shift", part); // uppercase letter encodes shift
      continue;
    }
    out.push(KEY_GLYPHS[part] ?? (part.length === 1 ? part.toUpperCase() : part));
  }
  return out.join(" + ");
}

// ---------------------------------------------------------------------------
// Binding resolution (pure)
// ---------------------------------------------------------------------------

export type HotkeyTarget = { type: "action"; action: HotkeyActionDef } | { type: "tag"; tagName: string };

// Effective keys for an action: user override when present, else the default.
export function actionKeys(config: UserHotkeys | null | undefined, actionId: string): string[] {
  const override = config?.bindings?.[actionId];
  if (override) return override;
  return HOTKEY_ACTIONS.find((a) => a.id === actionId)?.defaultKeys ?? [];
}

// Effective tag bindings: a non-null user map replaces the defaults entirely.
export function effectiveTagBindings(config: UserHotkeys | null | undefined): Record<string, string> {
  return config?.tagBindings ?? DEFAULT_TAG_BINDINGS;
}

export function resolveCombo(config: UserHotkeys | null | undefined, combo: string): HotkeyTarget[] {
  const targets: HotkeyTarget[] = [];
  for (const action of HOTKEY_ACTIONS) {
    if (actionKeys(config, action.id).includes(combo)) {
      targets.push({ type: "action", action });
    }
  }
  const tagName = effectiveTagBindings(config)[combo];
  if (tagName) targets.push({ type: "tag", tagName });
  return targets;
}

// ---------------------------------------------------------------------------
// Runtime: handler registry, context stack, dispatcher
// ---------------------------------------------------------------------------

type ActionHandler = (event: KeyboardEvent) => void;
type TagHandler = (tagName: string, event: KeyboardEvent) => void;

const actionHandlers = new Map<string, Set<ActionHandler>>();
const tagHandlers = new Set<TagHandler>();
export const hotkeyContextStack = ref<string[]>([]);

export function onAction(actionId: string, handler: ActionHandler): () => void {
  let handlers = actionHandlers.get(actionId);
  if (!handlers) {
    handlers = new Set();
    actionHandlers.set(actionId, handlers);
  }
  handlers.add(handler);
  return () => handlers.delete(handler);
}

export function onTagHotkey(handler: TagHandler): () => void {
  tagHandlers.add(handler);
  return () => tagHandlers.delete(handler);
}

export function pushHotkeyContext(context: string): () => void {
  hotkeyContextStack.value.push(context);
  return () => {
    const index = hotkeyContextStack.value.lastIndexOf(context);
    if (index !== -1) hotkeyContextStack.value.splice(index, 1);
  };
}

export function activeHotkeyContext(): string {
  return hotkeyContextStack.value[hotkeyContextStack.value.length - 1] ?? "global";
}

// Component-scoped variants: registered on setup, disposed with the scope.
export function useHotkeyAction(actionId: string, handler: ActionHandler): void {
  onScopeDispose(onAction(actionId, handler));
}

export function useTagHotkey(handler: TagHandler): void {
  onScopeDispose(onTagHotkey(handler));
}

export function useHotkeyContext(context: string): void {
  onScopeDispose(pushHotkeyContext(context));
}

let getConfig: () => UserHotkeys | null | undefined = () => null;

// App.vue wires the user's persisted hotkey config in here; resolution happens
// per keydown so config changes apply immediately without re-registration.
export function configureHotkeys(configGetter: () => UserHotkeys | null | undefined): void {
  getConfig = configGetter;
}

function isEditableTarget(event: KeyboardEvent): boolean {
  const el = event.target as HTMLElement | null;
  if (!el) return false;
  if (el.isContentEditable) return true;
  return el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.tagName === "SELECT";
}

export function hotkeyKeydownHandler(event: KeyboardEvent): void {
  const combo = comboFromEvent(event);
  if (!combo) return;
  const config = getConfig();
  const context = activeHotkeyContext();
  const editable = isEditableTarget(event);
  let handled = false;
  for (const target of resolveCombo(config, combo)) {
    if (target.type === "action") {
      const { action } = target;
      if (action.context !== "global" && action.context !== context) continue;
      if (editable && !action.allowInInputs) continue;
      const handlers = actionHandlers.get(action.id);
      if (!handlers || handlers.size === 0) continue;
      handlers.forEach((handler) => handler(event));
      handled = true;
    } else {
      // tag hotkeys only make sense where a current image exists
      if (context !== "images" || editable) continue;
      if (tagHandlers.size === 0) continue;
      tagHandlers.forEach((handler) => handler(target.tagName, event));
      handled = true;
    }
  }
  if (handled) event.preventDefault();
}
