-- gallery schema v1: public counters + bulk download jobs.
CREATE TABLE IF NOT EXISTS gallery.image_stats (
  image_id       text PRIMARY KEY,
  views          bigint NOT NULL DEFAULT 0,
  downloads      bigint NOT NULL DEFAULT 0,
  last_viewed_at timestamptz
);

CREATE TABLE IF NOT EXISTS gallery.download_jobs (
  id               text PRIMARY KEY,                -- 128-bit random, base32
  gallery_key      text NOT NULL,
  project_id       text NOT NULL,
  requester_hash   text NOT NULL,                   -- sha256(ip), for per-requester quotas only
  filter           jsonb NOT NULL,
  filter_hash      text NOT NULL,
  manifest         text[] NOT NULL,                 -- image ids at admission, re-validated at issue time
  status           text NOT NULL CHECK (status IN ('queued','running','done','failed','revoked')),
  attempts         int NOT NULL DEFAULT 0,
  max_attempts     int NOT NULL DEFAULT 2,
  lease_token      text,
  lease_expires_at timestamptz,
  heartbeat_at     timestamptz,
  image_count      int,
  bytes            bigint,
  s3_key           text,
  error            text,
  created_at       timestamptz NOT NULL DEFAULT now(),
  started_at       timestamptz,
  finished_at      timestamptz,
  expires_at       timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS download_jobs_active
  ON gallery.download_jobs (gallery_key, filter_hash) WHERE status IN ('queued','running','done');
CREATE INDEX IF NOT EXISTS download_jobs_claim
  ON gallery.download_jobs (gallery_key, status, created_at);
CREATE INDEX IF NOT EXISTS download_jobs_requester
  ON gallery.download_jobs (gallery_key, requester_hash, created_at);
