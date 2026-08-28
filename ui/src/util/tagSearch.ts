// Structured tag search for the FSG triple-tag format ("car_051|tid_1292|BE Ghent U").
// Mirrors the EXIF export's pipe segmentation (internal/exif/inject.go): search
// matches per segment, never across pipes, so "51" cannot land inside
// "tid_1051". Only the FIRST segment is the primary key and the only
// searchable identifier: numbers compare zero-normalized for exact hits
// (51 ≡ 051) and by raw prefix for weaker hits; later identifier segments
// (tid_*) are internal handles and never match. Free-text segments,
// displayName and description stay substring-searched.
// Lower rank = better; -1 = no match.

export interface TagLike {
  name: string;
  description?: string;
  displayName?: string;
}

export const TAG_SEARCH_NO_MATCH = -1;

export function tagSearchRank(tag: TagLike, query: string): number {
  const tokens = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
  if (tokens.length === 0) return TAG_SEARCH_NO_MATCH;
  let worst = 0;
  for (const token of tokens) {
    const rank = tagTokenRank(tag, token);
    if (rank < 0) return TAG_SEARCH_NO_MATCH;
    worst = Math.max(worst, rank);
  }
  return worst;
}

// Sort helper for search results: best (lowest) rank first, stable on ties.
export function byTagSearchRank<T extends TagLike>(query: string): (a: T, b: T) => number {
  return (a, b) => tagSearchRank(a, query) - tagSearchRank(b, query);
}

function tagTokenRank(tag: TagLike, token: string): number {
  let best = TAG_SEARCH_NO_MATCH;
  const consider = (rank: number) => {
    if (rank >= 0 && (best < 0 || rank < best)) best = rank;
  };

  const segments = tag.name
    .split("|")
    .map((s) => s.trim())
    .filter(Boolean);
  segments.forEach((segment, index) => {
    if (IDENTIFIER_RE.test(segment)) {
      // only the primary (first) segment is searchable; tid_* handles are not
      if (index === 0) consider(identifierRank(segment, token));
    } else if (segment.toLowerCase().includes(token)) {
      consider(2);
    }
  });

  if ((tag.displayName ?? "").toLowerCase().includes(token)) consider(2);
  if ((tag.description ?? "").toLowerCase().includes(token)) consider(3);

  return best;
}

const IDENTIFIER_RE = /^([a-z]+)[_-](\d+)$/i;
function identifierRank(segment: string, token: string): number {
  const match = IDENTIFIER_RE.exec(segment);
  if (!match) return TAG_SEARCH_NO_MATCH;
  const [, label, digits] = match;

  // "car51" / "CAR-51" behave like the pure-number form against this segment
  let labelQuery = "";
  let digitQuery = "";
  const combined = /^([a-z]+)[_-]?(\d+)$/i.exec(token);
  if (/^\d+$/.test(token)) {
    digitQuery = token;
  } else if (combined && combined[1].toLowerCase() === label.toLowerCase()) {
    labelQuery = combined[1];
    digitQuery = combined[2];
  } else {
    if (!label.toLowerCase().startsWith(token)) return TAG_SEARCH_NO_MATCH;
    return 2;
  }

  const stripped = (s: string) => s.replace(/^0+(?=\d)/, "");
  if (stripped(digits) === stripped(digitQuery)) return 0;
  // pure-digit prefixes need >=2 chars so degenerate queries don't flood the
  // list, and compare against the RAW digits ("051" must not reach car_512);
  // label-carrying queries like "car5" are explicit enough to match padded
  const prefixHit = digits.startsWith(digitQuery) || (labelQuery !== "" && stripped(digits).startsWith(stripped(digitQuery)));
  if (prefixHit && (digitQuery.length >= 2 || labelQuery)) return 1;
  return TAG_SEARCH_NO_MATCH;
}
