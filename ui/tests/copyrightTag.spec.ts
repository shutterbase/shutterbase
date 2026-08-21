import { describe, expect, it } from "vitest";
import { normalizeCopyrightTag } from "src/util/copyrightTag";

describe("normalizeCopyrightTag", () => {
  it("lowercases and replaces separators with underscores", () => {
    expect(normalizeCopyrightTag("mm")).toBe("mm");
    expect(normalizeCopyrightTag("MM")).toBe("mm");
    expect(normalizeCopyrightTag("Max Mustermann")).toBe("max_mustermann");
    expect(normalizeCopyrightTag("max-mustermann")).toBe("max_mustermann");
    expect(normalizeCopyrightTag("max.muster - x")).toBe("max_muster_x");
    expect(normalizeCopyrightTag("Möller")).toBe("moeller");
    expect(normalizeCopyrightTag("Äöü ß")).toBe("aeoeue_ss");
    expect(normalizeCopyrightTag("")).toBe("");
  });
});
