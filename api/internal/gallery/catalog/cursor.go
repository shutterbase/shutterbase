package catalog

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Cursor is the keyset position of the last item on a page: the sort key
// (capturedAtCorrected, nullable) and the id tiebreaker. Encoded opaque
// ("<unixnano|->.<id>", base64url) so URLs stay short and clients cannot
// meaningfully edit it.
type Cursor struct {
	At *time.Time
	ID string
}

func (c Cursor) Encode() string {
	at := "-"
	if c.At != nil {
		at = strconv.FormatInt(c.At.UnixNano(), 10)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(at + "." + c.ID))
}

var errBadCursor = errors.New("invalid cursor")

func DecodeCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, errBadCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, errBadCursor
	}
	at, id, ok := strings.Cut(string(raw), ".")
	if !ok || id == "" || len(id) > 40 {
		return Cursor{}, errBadCursor
	}
	c := Cursor{ID: id}
	if at != "-" {
		n, err := strconv.ParseInt(at, 10, 64)
		if err != nil {
			return Cursor{}, errBadCursor
		}
		// Local, not UTC: SQLite (tests) stores ent times as zoned text and
		// compares them as strings; Postgres timestamptz does not care.
		t := time.Unix(0, n)
		c.At = &t
	}
	return c, nil
}
