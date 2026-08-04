import { describe, it, expect, afterEach } from "vitest";
import {
  actionKeys,
  comboFromEvent,
  configureHotkeys,
  effectiveTagBindings,
  formatCombo,
  hotkeyKeydownHandler,
  onAction,
  onTagHotkey,
  pushHotkeyContext,
  resolveCombo,
  DEFAULT_TAG_BINDINGS,
} from "src/util/hotkeys";
import { UserHotkeys } from "src/types/api";

function keyEvent(init: KeyboardEventInit): KeyboardEvent {
  return new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init });
}

describe("comboFromEvent", () => {
  it("returns plain printable keys verbatim", () => {
    expect(comboFromEvent(keyEvent({ key: "t" }))).toBe("t");
    expect(comboFromEvent(keyEvent({ key: "?", shiftKey: true }))).toBe("?");
  });

  it("keeps shift implicit in uppercase letters", () => {
    expect(comboFromEvent(keyEvent({ key: "T", shiftKey: true }))).toBe("T");
  });

  it("normalizes shift+digit through the physical key code", () => {
    expect(comboFromEvent(keyEvent({ key: "!", code: "Digit1", shiftKey: true }))).toBe("shift+1");
  });

  it("adds shift explicitly for named keys", () => {
    expect(comboFromEvent(keyEvent({ key: "ArrowRight" }))).toBe("ArrowRight");
    expect(comboFromEvent(keyEvent({ key: "ArrowRight", shiftKey: true }))).toBe("shift+ArrowRight");
  });

  it("orders modifiers deterministically", () => {
    expect(comboFromEvent(keyEvent({ key: "k", ctrlKey: true, altKey: true }))).toBe("ctrl+alt+k");
  });

  it("normalizes space and ignores bare modifiers", () => {
    expect(comboFromEvent(keyEvent({ key: " " }))).toBe("Space");
    expect(comboFromEvent(keyEvent({ key: "Shift", shiftKey: true }))).toBeNull();
  });
});

describe("formatCombo", () => {
  it("renders glyphs and shift-encoded characters", () => {
    expect(formatCombo("ArrowRight")).toBe("→");
    expect(formatCombo("shift+1")).toBe("Shift + 1");
    expect(formatCombo("T")).toBe("Shift + T");
    expect(formatCombo("t")).toBe("T");
    expect(formatCombo("ctrl+alt+k")).toBe("Ctrl + Alt + K");
  });
});

describe("binding resolution", () => {
  it("falls back to defaults without config", () => {
    expect(actionKeys(null, "images.next-image")).toEqual(["ArrowRight", "l"]);
    expect(effectiveTagBindings(null)).toEqual(DEFAULT_TAG_BINDINGS);
  });

  it("applies overrides per action, including explicit unbinding", () => {
    const config: UserHotkeys = { bindings: { "images.next-image": ["n"], "images.toggle-view": [] } };
    expect(actionKeys(config, "images.next-image")).toEqual(["n"]);
    expect(actionKeys(config, "images.toggle-view")).toEqual([]);
    expect(actionKeys(config, "images.previous-image")).toEqual(["ArrowLeft", "h"]);
  });

  it("replaces tag bindings entirely when set", () => {
    const config: UserHotkeys = { tagBindings: { x: "podium" } };
    expect(effectiveTagBindings(config)).toEqual({ x: "podium" });
  });

  it("resolves combos to actions and tags, respecting overrides", () => {
    const config: UserHotkeys = { bindings: { "images.next-image": ["n"] } };
    expect(resolveCombo(config, "n").map((t) => (t.type === "action" ? t.action.id : ""))).toEqual(["images.next-image"]);
    expect(resolveCombo(config, "ArrowRight")).toEqual([]); // old default no longer bound
    const tagTargets = resolveCombo(null, "p");
    expect(tagTargets).toEqual([{ type: "tag", tagName: "review" }]);
  });
});

describe("hotkeyKeydownHandler", () => {
  const disposers: Array<() => void> = [];
  afterEach(() => {
    while (disposers.length) disposers.pop()!();
    configureHotkeys(() => null);
    document.body.innerHTML = "";
  });

  it("dispatches to registered handlers of the active context only", () => {
    const fired: string[] = [];
    disposers.push(pushHotkeyContext("images"));
    disposers.push(onAction("images.next-image", () => fired.push("next")));
    disposers.push(onAction("tagging.accept-only-result", () => fired.push("accept")));

    hotkeyKeydownHandler(keyEvent({ key: "ArrowRight" }));
    hotkeyKeydownHandler(keyEvent({ key: "Enter" })); // tagging context not active
    expect(fired).toEqual(["next"]);

    disposers.push(pushHotkeyContext("tagging-dialog"));
    hotkeyKeydownHandler(keyEvent({ key: "ArrowRight" })); // images context now shadowed
    hotkeyKeydownHandler(keyEvent({ key: "Enter" }));
    expect(fired).toEqual(["next", "accept"]);
  });

  it("suppresses non-input hotkeys while typing", () => {
    const fired: string[] = [];
    disposers.push(pushHotkeyContext("images"));
    disposers.push(onAction("images.open-tagging", () => fired.push("t")));
    disposers.push(onTagHotkey((tagName) => fired.push(`tag:${tagName}`)));

    const input = document.createElement("input");
    document.body.appendChild(input);
    document.addEventListener("keydown", hotkeyKeydownHandler);
    input.dispatchEvent(keyEvent({ key: "t" }));
    input.dispatchEvent(keyEvent({ key: "p" }));
    document.removeEventListener("keydown", hotkeyKeydownHandler);
    expect(fired).toEqual([]);

    hotkeyKeydownHandler(keyEvent({ key: "p" }));
    expect(fired).toEqual(["tag:review"]);
  });

  it("uses the user's custom bindings", () => {
    const fired: string[] = [];
    configureHotkeys(() => ({ bindings: { "images.next-image": ["n"] } }));
    disposers.push(pushHotkeyContext("images"));
    disposers.push(onAction("images.next-image", () => fired.push("next")));

    hotkeyKeydownHandler(keyEvent({ key: "ArrowRight" }));
    hotkeyKeydownHandler(keyEvent({ key: "n" }));
    expect(fired).toEqual(["next"]);
  });
});
