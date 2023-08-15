import { emitter } from "~/boot/mitt";

let hotkeysBlocked = false;

export function keyEventHandler(event: KeyboardEvent) {
  // console.log("keypress", event.key);
  if (hotkeysBlocked) {
    console.log("hotkeys blocked due to operation in progress");
    return;
  }
  const emitterKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Escape", "Enter", "t", "r", "h", "l"];

  if (emitterKeys.includes(event.key)) {
    emitter.emit(`key-${event.key}`, event);
  }

  if (event.shiftKey && event.keyCode >= 48 && event.keyCode <= 57) {
    emitter.emit(`key-shift-hotkey`, { event, keyNumber: event.keyCode - 48 });
  }
}

emitter.on("block-hotkeys", () => (hotkeysBlocked = true));
emitter.on("unblock-hotkeys", () => (hotkeysBlocked = false));
