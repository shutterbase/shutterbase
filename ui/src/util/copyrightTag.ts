// The copyright tag ends up in EXIF fields and computed filenames:
// lowercase, umlauts transliterated (ä→ae, ö→oe, ü→ue, ß→ss), and any run
// of non-letter/digit/underscore characters (whitespace, dashes, dots, …)
// collapsed to a single underscore.
// Mirrored in api/internal/repository/user.go (NormalizeCopyrightTag).
const UMLAUTS: Record<string, string> = { ä: "ae", ö: "oe", ü: "ue", ß: "ss" };

export function normalizeCopyrightTag(tag: string): string {
  return tag
    .toLowerCase()
    .replace(/[äöüß]/g, (c) => UMLAUTS[c] ?? c)
    .replace(/[^\p{L}\p{N}_]+/gu, "_");
}
