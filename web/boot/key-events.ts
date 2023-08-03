import { emitter } from "~/boot/mitt";

export function keyEventHandler(event: KeyboardEvent) {
  console.log("keypress", event.key);
  if (event.key == "ArrowLeft") {
    emitter.emit("arrow-left");
  }

  if (event.key == "ArrowRight") {
    emitter.emit("arrow-right");
  }
}
