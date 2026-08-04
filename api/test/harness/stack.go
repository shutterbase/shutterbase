package harness

import (
	"context"
	"time"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// Stack is the full ephemeral environment: Postgres + S3 + applied schema +
// seeded fixtures. Used by the API e2e tests and by cmd/testserver (Playwright).
type Stack struct {
	PG       *Postgres
	S3       *S3
	DB       *database.Connection
	Manifest *seed.Manifest
}

// Up brings up Postgres and S3, applies the schema, and seeds fixtures derived
// from referenceNow. Call Close (or rely on testcontainers Ryuk) to tear down.
func Up(ctx context.Context, referenceNow time.Time) (*Stack, error) {
	pg, err := StartPostgres(ctx)
	if err != nil {
		return nil, err
	}
	s3c, err := StartS3(ctx)
	if err != nil {
		_ = pg.Container.Terminate(ctx)
		return nil, err
	}
	db, err := Migrate(pg.Options)
	if err != nil {
		_ = pg.Container.Terminate(ctx)
		_ = s3c.Container.Terminate(ctx)
		return nil, err
	}
	manifest, err := Seed(ctx, db.Client, referenceNow)
	if err != nil {
		db.Close()
		_ = pg.Container.Terminate(ctx)
		_ = s3c.Container.Terminate(ctx)
		return nil, err
	}
	return &Stack{PG: pg, S3: s3c, DB: db, Manifest: manifest}, nil
}

// Close tears down the DB connection and both containers.
func (s *Stack) Close(ctx context.Context) {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.PG != nil {
		_ = s.PG.Container.Terminate(ctx)
	}
	if s.S3 != nil {
		_ = s.S3.Container.Terminate(ctx)
	}
}
