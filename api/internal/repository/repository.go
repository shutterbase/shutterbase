// Package repository is the per-entity data-access layer. Each entity gets its
// own file mirroring the agentic-template pattern: typed Get/Create/Update param
// structs (pointer = "provided"), Get<E>/Get<E>s/Create<E>/Update<E>/Delete<E>.
//
// Mutations are audited (CreateAuditLog via safeGo + context.WithoutCancel).
// Per-entity WebSocket broadcast is DEFERRED (REWRITE-SPEC §1) — no realtime UI
// consumes entity-mutation events yet; only the time-sync tick is live.
// ponytail: per-entity WS broadcast is dead flexibility until a realtime gallery
// exists; the tick is the only consumer.
//
// ponytail: no per-entity LRU caches. The template carries a userCache; here the
// reads are not yet a measured hot path and cache eviction in every Update/Delete
// is complexity with no consumer. Add a cache to a single entity when a profile
// flags it.
package repository

import (
	"errors"

	"entgo.io/ent/dialect/sql"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/database"
)

type Repository struct {
	Options *Options
	Client  *ent.Client
}

type Options struct {
	DatabaseConnection *database.Connection
}

func NewRepository(options *Options) (*Repository, error) {
	return &Repository{
		Options: options,
		Client:  options.DatabaseConnection.Client,
	}, nil
}

func (r *Repository) isPostgres() bool {
	return r.Options.DatabaseConnection.Options.DatabaseType == "psql"
}

// ErrInvalidSort is returned when a requested sort key is not on a resource's
// allowlist. Controllers map it to 400 {"code":"invalid_sort"}.
var ErrInvalidSort = errors.New("invalid_sort")

// PaginationParameters carries the LIST window. Sort is the API-facing key (e.g.
// "capturedAtCorrected"); build() validates it against a per-resource allowlist
// and translates it to the ent column.
type PaginationParameters struct {
	Limit  int
	Offset int
	Sort   string
	Order  string
}

// build clamps limit/offset, validates Sort against allow (API key -> column),
// defaulting to defaultKey when Sort is empty, and returns the ent order func.
// An off-allowlist Sort yields ErrInvalidSort.
func (p *PaginationParameters) build(allow map[string]string, defaultKey string) (limit, offset int, order func(*sql.Selector), err error) {
	if p == nil {
		p = &PaginationParameters{}
	}
	limit = p.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset = p.Offset
	if offset < 0 {
		offset = 0
	}
	key := p.Sort
	if key == "" {
		key = defaultKey
	}
	col, ok := allow[key]
	if !ok {
		return 0, 0, nil, ErrInvalidSort
	}
	if p.Order == "asc" {
		return limit, offset, ent.Asc(col), nil
	}
	return limit, offset, ent.Desc(col), nil
}

// modelUpdateStatus tracks field-level changes during an Update so a no-op
// update can be rolled back (no audit row) and real changes audited with diffs.
type modelUpdateStatus struct {
	modelChanged bool
	fieldChanges map[string]modelUpdateEntry
}

type modelUpdateEntry struct {
	oldValue any
	newValue any
}

// SetFieldChanged records a field's old/new values. Sensitive values must be
// passed as "<redacted>" by the caller.
func (s *modelUpdateStatus) SetFieldChanged(fieldName string, oldValue, newValue any) {
	if s.fieldChanges == nil {
		s.fieldChanges = make(map[string]modelUpdateEntry)
	}
	s.fieldChanges[fieldName] = modelUpdateEntry{oldValue: oldValue, newValue: newValue}
	s.modelChanged = true
}

func (s *modelUpdateStatus) GetChangedFieldData() map[string]any {
	result := make(map[string]any, len(s.fieldChanges))
	for key, value := range s.fieldChanges {
		result[key] = map[string]any{"old": value.oldValue, "new": value.newValue}
	}
	return result
}

// safeGo runs fn in a goroutine with panic recovery (audit fire-and-forget).
func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("recovered panic in background goroutine")
			}
		}()
		fn()
	}()
}
