import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

// A colour utility naming a token that does not exist in tailwind.config.js
// compiles to NO CSS AT ALL — no build error, no tsc error, no unit-test
// failure. The class silently drops and the element keeps whatever the cascade
// left behind. That is how the time-range inputs ended up light-on-light in
// dark mode: `dark:bg-surface-900` against a `surface` scale that only has
// DEFAULT/muted/dark/dark-muted, so the background stayed `bg-surface-muted`
// while `dark:text-primary-100` still applied.
//
// This spec resolves every colour utility under src/ against the config, so a
// bad token fails here rather than in someone's eyes.
//
// The config is parsed as TEXT, not required: ui/package.json sets
// `"type": "module"`, so requiring this CommonJS file yields an empty object.

const ROOT = join(__dirname, "..");
const CONFIG = readFileSync(join(ROOT, "tailwind.config.js"), "utf8");

/** Body of the first `key: { … }` block, brace-matched. */
function blockAfter(src: string, key: string): string {
  const at = src.indexOf(key);
  if (at < 0) throw new Error(`tailwind.config.js has no \`${key}\``);
  let depth = 1;
  let i = at + key.length;
  while (i < src.length && depth > 0) {
    if (src[i] === "{") depth++;
    else if (src[i] === "}") depth--;
    i++;
  }
  return src.slice(at + key.length, i);
}

/**
 * Colour scales and their shades: the ones this project defines under
 * `theme.extend.colors`, plus Tailwind's own palette (which `extend` keeps),
 * so utilities like `text-violet-800` are not false positives.
 */
function parseScales(): Map<string, Set<string>> {
  const body = blockAfter(CONFIG, "colors: {");
  const scales = new Map<string, Set<string>>();

  const head = /([a-zA-Z][\w-]*):\s*\{/g;
  for (let m = head.exec(body); m; m = head.exec(body)) {
    if (m.index === 0) continue; // the `colors` key itself
    const shades = new Set<string>(["DEFAULT"]);
    for (const s of blockAfter(body.slice(m.index), `${m[1]}: {`).matchAll(/([\w-]+)\s*:\s*["']*#/g)) {
      shades.add(s[1]);
    }
    scales.set(m[1], shades);
  }
  if (scales.size === 0) throw new Error("parsed zero colour scales from tailwind.config.js");
  return scales;
}

const SCALES = parseScales();

// Utilities whose value is a colour and therefore resolves against `colors`.
const COLOR_UTIL =
  /\b(?:bg|text|border|ring|placeholder|divide|from|to|via|fill|stroke|outline|accent|caret|decoration)-([a-z][\w-]*)/g;
const KEYWORDS = new Set(["white", "black", "transparent", "current", "none", "inherit"]);

function* sourceFiles(dir: string): Generator<string> {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) yield* sourceFiles(full);
    else if (/\.(vue|ts|js)$/.test(entry)) yield full;
  }
}

/**
 * Drop comments so a class named inside an explanatory comment (this file's
 * own, or a "was `dark:bg-surface-900`, now …" note) is not mistaken for live
 * markup. Newlines are preserved exactly — reported line numbers must still
 * point at the source. `//` counts as a comment only at line start or after
 * whitespace, so a `https://` inside a string survives.
 */
function stripComments(src: string): string {
  const blank = (m: string) => m.replace(/[^\n]/g, " ");
  return src
    .replace(/\/\*[\s\S]*?\*\//g, blank)
    .replace(/<!--[\s\S]*?-->/g, blank)
    .split("\n")
    .map((line) => {
      const at = line.search(/(^|\s)\/\//);
      return at < 0 ? line : line.slice(0, at);
    })
    .join("\n");
}

/**
 * Every `<prefix>-<scale>-<shade>` whose scale this project defines but which
 * lacks that shade.
 *
 * Deliberately narrow. Only shades of PROJECT scales are checked, because the
 * generic prefix set is overloaded — `text-sm` is a font size, `ring-offset-2`
 * is a box-shadow offset, and `to-context` comes from prose ("from A to B") — so
 * validating Tailwind's own palette here would drown the signal in false
 * positives. A misspelled bare scale name (`bg-surfaces`) is also not caught;
 * only the shape that actually bit us, a real scale with a made-up shade, is.
 */
function unresolved(): string[] {
  const bad: string[] = [];
  for (const file of sourceFiles(join(ROOT, "src"))) {
    const rel = relative(ROOT, file);
    stripComments(readFileSync(file, "utf8"))
      .split("\n")
      .forEach((line, idx) => {
        for (const m of line.matchAll(COLOR_UTIL)) {
          // strip an opacity modifier: bg-primary-900/50 -> primary-900
          const parts = m[1].split("/")[0].split("-");
          if (parts.length !== 2) continue; // bare scale name, or a non-colour utility
          if (KEYWORDS.has(parts.join("-"))) continue;
          const shades = SCALES.get(parts[0]);
          if (!shades) continue; // not a project scale — Tailwind's own palette
          if (!shades.has(parts[1])) {
            bad.push(
              `${rel}:${idx + 1}: ${m[0]} — scale '${parts[0]}' has no shade '${parts[1]}' ` +
                `(has: ${[...shades].filter((s) => s !== "DEFAULT").join(", ")})`,
            );
          }
        }
      });
  }
  return bad;
}

describe("tailwind colour tokens", () => {
  it("parses the config's colour scales", () => {
    // Guards the parser itself: a Tailwind upgrade that reshapes the config
    // must fail here loudly rather than silently passing every utility.
    expect(SCALES.size).toBeGreaterThanOrEqual(6);
    expect(SCALES.get("primary")?.has("900")).toBe(true);
    expect(SCALES.get("accent")?.has("500")).toBe(true);
    // The regression this spec exists for: `surface` has no numeric shades, so
    // `dark:bg-surface-900` generated nothing and left the light background.
    expect(SCALES.get("surface")?.has("900")).toBe(false);
  });

  it("resolves every colour utility in src/ against the config", () => {
    const bad = unresolved();
    expect(bad, `unresolvable colour utilities:\n${bad.join("\n")}`).toEqual([]);
  });
});
