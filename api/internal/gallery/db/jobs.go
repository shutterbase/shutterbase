package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/lib/pq"
)

const (
	JobQueued  = "queued"
	JobRunning = "running"
	JobDone    = "done"
	JobFailed  = "failed"
	JobRevoked = "revoked"
)

var ErrJobNotFound = errors.New("job not found")

// Job is one bulk download request. Manifest is the image id set admitted
// at creation; the worker re-checks publication per id when it renders and
// the issuer re-checks the whole manifest before handing out the zip.
type Job struct {
	ID             string
	GalleryKey     string
	ProjectID      string
	RequesterHash  string
	Filter         json.RawMessage
	FilterHash     string
	Manifest       []string
	Status         string
	Attempts       int
	MaxAttempts    int
	LeaseToken     string
	LeaseExpiresAt *time.Time
	HeartbeatAt    *time.Time
	ImageCount     int
	Bytes          int64
	S3Key          string
	Error          string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	ExpiresAt      *time.Time
}

// NewJobID is 128 random bits, base32 without padding (26 chars): unguessable.
func NewJobID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
}

// JobStore is the download-job persistence. Every method is scoped by
// gallery key: one gallery's worker never claims, and one gallery's visitor
// never reads, another gallery's job.
type JobStore interface {
	Create(ctx context.Context, j *Job) error
	// Reusable finds an active (queued/running) or unexpired done job for the
	// same filter hash, or nil.
	Reusable(ctx context.Context, galleryKey, filterHash string, now time.Time) (*Job, error)
	Get(ctx context.Context, galleryKey, id string) (*Job, error)
	CountActive(ctx context.Context, galleryKey string) (int, error)
	CountByRequester(ctx context.Context, galleryKey, requesterHash string, since time.Time) (int, error)
	// Claim takes the oldest queued (or lease-expired running) job under a
	// fresh lease; nil when the queue is empty.
	Claim(ctx context.Context, galleryKey, token string, lease time.Duration) (*Job, error)
	Heartbeat(ctx context.Context, id, token string, lease time.Duration) error
	Finish(ctx context.Context, id, token string, imageCount int, bytes int64, s3Key string, expiresAt time.Time) error
	Fail(ctx context.Context, id, token, message string) error
	Revoke(ctx context.Context, galleryKey, id string) error
	// Expired lists done/revoked/failed jobs past their expiry (cleanup).
	Expired(ctx context.Context, galleryKey string, now time.Time) ([]*Job, error)
	Delete(ctx context.Context, galleryKey, id string) error
}

// --- Postgres ---

type PostgresJobs struct{ DB *sql.DB }

const jobColumns = `id, gallery_key, project_id, requester_hash, filter, filter_hash, manifest, status, attempts, max_attempts,
	coalesce(lease_token,''), lease_expires_at, heartbeat_at, coalesce(image_count,0), coalesce(bytes,0), coalesce(s3_key,''), coalesce(error,''),
	created_at, started_at, finished_at, expires_at`

func scanJob(row interface{ Scan(...any) error }) (*Job, error) {
	j := &Job{}
	var manifest pq.StringArray
	err := row.Scan(&j.ID, &j.GalleryKey, &j.ProjectID, &j.RequesterHash, &j.Filter, &j.FilterHash, &manifest, &j.Status, &j.Attempts, &j.MaxAttempts,
		&j.LeaseToken, &j.LeaseExpiresAt, &j.HeartbeatAt, &j.ImageCount, &j.Bytes, &j.S3Key, &j.Error,
		&j.CreatedAt, &j.StartedAt, &j.FinishedAt, &j.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	j.Manifest = []string(manifest)
	return j, nil
}

func (s *PostgresJobs) Create(ctx context.Context, j *Job) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO gallery.download_jobs
		(id, gallery_key, project_id, requester_hash, filter, filter_hash, manifest, status, max_attempts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'queued',$8)`,
		j.ID, j.GalleryKey, j.ProjectID, j.RequesterHash, []byte(j.Filter), j.FilterHash, pq.StringArray(j.Manifest), maxAttempts(j))
	return err
}

func maxAttempts(j *Job) int {
	if j.MaxAttempts <= 0 {
		return 2
	}
	return j.MaxAttempts
}

func (s *PostgresJobs) Reusable(ctx context.Context, galleryKey, filterHash string, now time.Time) (*Job, error) {
	j, err := scanJob(s.DB.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM gallery.download_jobs
		WHERE gallery_key=$1 AND filter_hash=$2 AND (status IN ('queued','running') OR (status='done' AND expires_at > $3))
		ORDER BY created_at DESC LIMIT 1`, galleryKey, filterHash, now))
	if errors.Is(err, ErrJobNotFound) {
		return nil, nil
	}
	return j, err
}

func (s *PostgresJobs) Get(ctx context.Context, galleryKey, id string) (*Job, error) {
	return scanJob(s.DB.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM gallery.download_jobs WHERE gallery_key=$1 AND id=$2`, galleryKey, id))
}

func (s *PostgresJobs) CountActive(ctx context.Context, galleryKey string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM gallery.download_jobs WHERE gallery_key=$1 AND status IN ('queued','running')`, galleryKey).Scan(&n)
	return n, err
}

func (s *PostgresJobs) CountByRequester(ctx context.Context, galleryKey, requesterHash string, since time.Time) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM gallery.download_jobs WHERE gallery_key=$1 AND requester_hash=$2 AND created_at > $3`, galleryKey, requesterHash, since).Scan(&n)
	return n, err
}

func (s *PostgresJobs) Claim(ctx context.Context, galleryKey, token string, lease time.Duration) (*Job, error) {
	j, err := scanJob(s.DB.QueryRowContext(ctx, `UPDATE gallery.download_jobs SET status='running', lease_token=$2, lease_expires_at=now()+$3::interval,
			heartbeat_at=now(), started_at=coalesce(started_at, now()), attempts=attempts+1
		WHERE id = (SELECT id FROM gallery.download_jobs
			WHERE gallery_key=$1 AND attempts < max_attempts
			  AND (status='queued' OR (status='running' AND lease_expires_at < now()))
			ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
		RETURNING `+jobColumns, galleryKey, token, lease.String()))
	if errors.Is(err, ErrJobNotFound) {
		return nil, nil
	}
	return j, err
}

func (s *PostgresJobs) Heartbeat(ctx context.Context, id, token string, lease time.Duration) error {
	return s.leased(ctx, id, token, `UPDATE gallery.download_jobs SET heartbeat_at=now(), lease_expires_at=now()+$3::interval WHERE id=$1 AND lease_token=$2 AND status='running'`, lease.String())
}

func (s *PostgresJobs) Finish(ctx context.Context, id, token string, imageCount int, bytes int64, s3Key string, expiresAt time.Time) error {
	return s.leased(ctx, id, token, `UPDATE gallery.download_jobs SET status='done', image_count=$3, bytes=$4, s3_key=$5, expires_at=$6, finished_at=now(), lease_token=NULL
		WHERE id=$1 AND lease_token=$2 AND status='running'`, imageCount, bytes, s3Key, expiresAt)
}

func (s *PostgresJobs) Fail(ctx context.Context, id, token, message string) error {
	// Retryable while attempts remain: back to queued, keep the message.
	return s.leased(ctx, id, token, `UPDATE gallery.download_jobs SET
			status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
			error=$3, lease_token=NULL, lease_expires_at=NULL, finished_at = CASE WHEN attempts >= max_attempts THEN now() ELSE NULL END
		WHERE id=$1 AND lease_token=$2 AND status='running'`, message)
}

// leased runs an UPDATE guarded by the lease token; a lost lease is an error
// so the worker stops writing to a job another worker took over.
func (s *PostgresJobs) leased(ctx context.Context, id, token, query string, args ...any) error {
	res, err := s.DB.ExecContext(ctx, query, append([]any{id, token}, args...)...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("lease lost")
	}
	return nil
}

func (s *PostgresJobs) Revoke(ctx context.Context, galleryKey, id string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE gallery.download_jobs SET status='revoked', finished_at=now() WHERE gallery_key=$1 AND id=$2`, galleryKey, id)
	return err
}

func (s *PostgresJobs) Expired(ctx context.Context, galleryKey string, now time.Time) ([]*Job, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+jobColumns+` FROM gallery.download_jobs
		WHERE gallery_key=$1 AND ((status='done' AND expires_at < $2) OR (status IN ('failed','revoked') AND finished_at < $2 - interval '1 day'))`, galleryKey, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *PostgresJobs) Delete(ctx context.Context, galleryKey, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM gallery.download_jobs WHERE gallery_key=$1 AND id=$2`, galleryKey, id)
	return err
}

// --- in-memory (unit tests, SQLite dev) ---

type MemoryJobs struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewMemoryJobs() *MemoryJobs { return &MemoryJobs{jobs: map[string]*Job{}} }

func (m *MemoryJobs) Create(_ context.Context, j *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	cp.Status = JobQueued
	cp.MaxAttempts = maxAttempts(j)
	cp.CreatedAt = time.Now()
	m.jobs[j.ID] = &cp
	return nil
}

func (m *MemoryJobs) Reusable(_ context.Context, galleryKey, filterHash string, now time.Time) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *Job
	for _, j := range m.jobs {
		if j.GalleryKey != galleryKey || j.FilterHash != filterHash {
			continue
		}
		active := j.Status == JobQueued || j.Status == JobRunning || (j.Status == JobDone && j.ExpiresAt != nil && j.ExpiresAt.After(now))
		if active && (best == nil || j.CreatedAt.After(best.CreatedAt)) {
			best = j
		}
	}
	if best == nil {
		return nil, nil
	}
	cp := *best
	return &cp, nil
}

func (m *MemoryJobs) Get(_ context.Context, galleryKey, id string) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok || j.GalleryKey != galleryKey {
		return nil, ErrJobNotFound
	}
	cp := *j
	return &cp, nil
}

func (m *MemoryJobs) CountActive(_ context.Context, galleryKey string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.jobs {
		if j.GalleryKey == galleryKey && (j.Status == JobQueued || j.Status == JobRunning) {
			n++
		}
	}
	return n, nil
}

func (m *MemoryJobs) CountByRequester(_ context.Context, galleryKey, requesterHash string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.jobs {
		if j.GalleryKey == galleryKey && j.RequesterHash == requesterHash && j.CreatedAt.After(since) {
			n++
		}
	}
	return n, nil
}

func (m *MemoryJobs) Claim(_ context.Context, galleryKey, token string, lease time.Duration) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	var pick *Job
	for _, j := range m.jobs {
		if j.GalleryKey != galleryKey || j.Attempts >= j.MaxAttempts {
			continue
		}
		expired := j.Status == JobRunning && j.LeaseExpiresAt != nil && j.LeaseExpiresAt.Before(now)
		if (j.Status == JobQueued || expired) && (pick == nil || j.CreatedAt.Before(pick.CreatedAt)) {
			pick = j
		}
	}
	if pick == nil {
		return nil, nil
	}
	exp := now.Add(lease)
	pick.Status, pick.LeaseToken, pick.LeaseExpiresAt, pick.HeartbeatAt = JobRunning, token, &exp, &now
	if pick.StartedAt == nil {
		pick.StartedAt = &now
	}
	pick.Attempts++
	cp := *pick
	return &cp, nil
}

func (m *MemoryJobs) leased(id, token string, fn func(j *Job)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok || j.LeaseToken != token || j.Status != JobRunning {
		return errors.New("lease lost")
	}
	fn(j)
	return nil
}

func (m *MemoryJobs) Heartbeat(_ context.Context, id, token string, lease time.Duration) error {
	return m.leased(id, token, func(j *Job) { now := time.Now(); exp := now.Add(lease); j.HeartbeatAt, j.LeaseExpiresAt = &now, &exp })
}

func (m *MemoryJobs) Finish(_ context.Context, id, token string, imageCount int, bytes int64, s3Key string, expiresAt time.Time) error {
	return m.leased(id, token, func(j *Job) {
		now := time.Now()
		j.Status, j.ImageCount, j.Bytes, j.S3Key, j.ExpiresAt, j.FinishedAt, j.LeaseToken = JobDone, imageCount, bytes, s3Key, &expiresAt, &now, ""
	})
}

func (m *MemoryJobs) Fail(_ context.Context, id, token, message string) error {
	return m.leased(id, token, func(j *Job) {
		j.Error, j.LeaseToken, j.LeaseExpiresAt = message, "", nil
		if j.Attempts >= j.MaxAttempts {
			now := time.Now()
			j.Status, j.FinishedAt = JobFailed, &now
		} else {
			j.Status = JobQueued
		}
	})
}

func (m *MemoryJobs) Revoke(_ context.Context, galleryKey, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok && j.GalleryKey == galleryKey {
		now := time.Now()
		j.Status, j.FinishedAt = JobRevoked, &now
	}
	return nil
}

func (m *MemoryJobs) Expired(_ context.Context, galleryKey string, now time.Time) ([]*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Job
	for _, j := range m.jobs {
		if j.GalleryKey != galleryKey {
			continue
		}
		if (j.Status == JobDone && j.ExpiresAt != nil && j.ExpiresAt.Before(now)) ||
			((j.Status == JobFailed || j.Status == JobRevoked) && j.FinishedAt != nil && j.FinishedAt.Before(now.Add(-24*time.Hour))) {
			cp := *j
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *MemoryJobs) Delete(_ context.Context, galleryKey, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok && j.GalleryKey == galleryKey {
		delete(m.jobs, id)
	}
	return nil
}
