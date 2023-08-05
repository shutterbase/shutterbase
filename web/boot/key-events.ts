import { emitter } from "~/boot/mitt";

export function keyEventHandler(event: KeyboardEvent) {
  // console.log("keypress", event.key);

  const emitterKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Escape", "Enter", "t", "h", "l"];

  if (emitterKeys.includes(event.key)) {
    emitter.emit(`key-${event.key}`, event);
  }

  if (event.shiftKey && event.keyCode >= 48 && event.keyCode <= 57) {
    emitter.emit(`key-shift-hotkey`, { event, keyNumber: event.keyCode - 48 });
  }
}
