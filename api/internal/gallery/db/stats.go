package db

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Stats collects public view/download counters in memory and flushes them
// in one UPSERT per image every interval. No visitor identity is involved.
type Stats struct {
	db       *sql.DB
	mu       sync.Mutex
	views    map[string]int64
	download map[string]int64
}

func NewStats(db *sql.DB) *Stats {
	return &Stats{db: db, views: map[string]int64{}, download: map[string]int64{}}
}

func (s *Stats) View(imageID string) {
	s.mu.Lock()
	s.views[imageID]++
	s.mu.Unlock()
}

func (s *Stats) Download(imageID string) {
	s.mu.Lock()
	s.download[imageID]++
	s.mu.Unlock()
}

// Counts reads one image's totals (pending increments are added on top).
func (s *Stats) Counts(ctx context.Context, imageID string) (views, downloads int64) {
	if s.db != nil {
		_ = s.db.QueryRowContext(ctx, `SELECT views, downloads FROM gallery.image_stats WHERE image_id=$1`, imageID).Scan(&views, &downloads)
	}
	s.mu.Lock()
	views += s.views[imageID]
	downloads += s.download[imageID]
	s.mu.Unlock()
	return
}

// Run flushes every interval until ctx ends, then once more.
func (s *Stats) Run(ctx context.Context, interval time.Duration) {
	if s.db == nil {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.Flush(context.Background())
			return
		case <-t.C:
			s.Flush(ctx)
		}
	}
}

func (s *Stats) Flush(ctx context.Context) {
	if s.db == nil {
		return
	}
	s.mu.Lock()
	views, downloads := s.views, s.download
	s.views, s.download = map[string]int64{}, map[string]int64{}
	s.mu.Unlock()
	ids := map[string]struct{}{}
	for id := range views {
		ids[id] = struct{}{}
	}
	for id := range downloads {
		ids[id] = struct{}{}
	}
	for id := range ids {
		_, err := s.db.ExecContext(ctx, `INSERT INTO gallery.image_stats (image_id, views, downloads, last_viewed_at)
			VALUES ($1, $2, $3, CASE WHEN $2 > 0 THEN now() END)
			ON CONFLICT (image_id) DO UPDATE SET views = image_stats.views + EXCLUDED.views,
				downloads = image_stats.downloads + EXCLUDED.downloads,
				last_viewed_at = coalesce(EXCLUDED.last_viewed_at, image_stats.last_viewed_at)`, id, views[id], downloads[id])
		if err != nil {
			log.Error().Err(err).Msg("gallery: stats flush")
			// put the increments back so a transient failure loses nothing
			s.mu.Lock()
			s.views[id] += views[id]
			s.download[id] += downloads[id]
			s.mu.Unlock()
		}
	}
}
