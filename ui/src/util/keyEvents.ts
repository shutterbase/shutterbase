import { emitter } from "boot/mitt";
import { debug } from "src/util/logger";

let hotkeysBlocked = false;

export type HotkeyEvent = {
  event: KeyboardEvent;
  key: string;
  modifierKeys: ModifierKey[];
};

export type HotkeyFilter = {
  key: string;
  modifierKeys: ModifierKey[];
};

type ModifierKey = "shift" | "ctrl" | "alt" | "meta";

export function keyEventHandler(event: KeyboardEvent) {
  // console.log("keypress", event.key);
  if (hotkeysBlocked) {
    console.log("hotkeys blocked due to operation in progress");
    return;
  }
  const emitterKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Escape", "Enter", "g", "t", "s", "h", "l", "j", "k"];

  if (emitterKeys.includes(event.key)) {
    debug(`key-${event.key}`);
    emitter.emit(`hotkey`, { event, key: event.key, modifierKeys: [] });
  }

  if (event.shiftKey && event.keyCode >= 48 && event.keyCode <= 57) {
    debug(`key-shift-hotkey`);
    emitter.emit(`hotkey`, { event, modifierKeys: ["shift"], key: `${event.keyCode - 48}` });
  }
}

emitter.on("block-hotkeys", () => (hotkeysBlocked = true));
emitter.on("unblock-hotkeys", () => (hotkeysBlocked = false));

export function onHotkey(filter: HotkeyFilter, callback: (event: HotkeyEvent) => void) {
  emitter.on("hotkey", (event: HotkeyEvent) => {
    if (filter.key === event.key && filter.modifierKeys.every((key) => event.modifierKeys.includes(key))) {
      callback(event);
    }
  });
}
