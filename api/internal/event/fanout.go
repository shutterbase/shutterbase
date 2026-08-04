package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// notifyChannel is the Postgres NOTIFY channel every replica LISTENs on.
const notifyChannel = "shutterbase_events"

// Bus fans schedule events out to the websocket clients of EVERY replica. On
// Postgres it rides LISTEN/NOTIFY: Publish pg_notify's the serialized message,
// and each replica's listen loop (including the publisher's own — NOTIFY
// reaches the notifying session's listeners too) broadcasts it to its local
// connections. Without Postgres (SQLite dev/tests: single process by
// definition) Publish broadcasts directly.
type Bus struct {
	ws  *Manager
	db  *sql.DB // pg_notify sender; nil => local-only mode
	dsn string  // dedicated LISTEN connection; "" => local-only mode
}

// NewBus builds the fan-out bus. Pass db=nil / dsn="" for local-only mode.
func NewBus(ws *Manager, db *sql.DB, dsn string) *Bus {
	return &Bus{ws: ws, db: db, dsn: dsn}
}

// Publish sends a schedule event to all replicas' clients. On a NOTIFY error it
// degrades to a local broadcast so at least this replica's clients stay fresh.
func (b *Bus) Publish(ctx context.Context, msg WebsocketMessage[ScheduleEventData]) {
	PublishEvent(b, ctx, msg)
}

// PublishEvent is the payload-generic fan-out (methods can't be generic). The
// listen loop re-broadcasts payloads as raw JSON, so any payload type works.
func PublishEvent[T any](b *Bus, ctx context.Context, msg WebsocketMessage[T]) {
	if b == nil {
		return // services in unit tests run without a bus
	}
	if b.db == nil {
		Broadcast(b.ws, msg)
		return
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling event")
		return
	}
	if _, err := b.db.ExecContext(ctx, `SELECT pg_notify($1, $2)`, notifyChannel, string(payload)); err != nil {
		log.Error().Err(err).Msg("pg_notify failed; falling back to local broadcast")
		Broadcast(b.ws, msg)
	}
}

// Start launches the LISTEN loop (no-op in local-only mode). It reconnects
// with a backoff until ctx is cancelled — crash-restart is the recovery model.
func (b *Bus) Start(ctx context.Context) {
	if b.dsn == "" {
		return
	}
	go b.listenLoop(ctx)
}

func (b *Bus) listenLoop(ctx context.Context) {
	for ctx.Err() == nil {
		if err := b.listenOnce(ctx); err != nil && ctx.Err() == nil {
			log.Warn().Err(err).Msg("schedule event listener disconnected; reconnecting")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (b *Bus) listenOnce(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, b.dsn)
	if err != nil {
		return err
	}
	defer conn.Close(context.WithoutCancel(ctx))
	if _, err := conn.Exec(ctx, "LISTEN "+notifyChannel); err != nil {
		return err
	}
	log.Info().Msg("schedule event listener attached")
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		// Payload-agnostic: re-broadcast the data verbatim (json.RawMessage
		// round-trips), so schedule and AI events share one channel.
		var msg WebsocketMessage[json.RawMessage]
		if err := json.Unmarshal([]byte(n.Payload), &msg); err != nil {
			log.Warn().Err(err).Msg("dropping malformed event payload")
			continue
		}
		Broadcast(b.ws, msg)
	}
}
