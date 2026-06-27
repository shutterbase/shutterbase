package importer

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/camera"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/projectassignment"
	"github.com/shutterbase/shutterbase/ent/timeoffset"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/s3"
)

// VerifyResult is the structured outcome of the post-import verification suite.
type VerifyResult struct {
	Counts         map[string][2]int // table -> {pbCount, pgCount}
	CountMismatch  []string
	Orphans        []string // "<entity>.<edge>=<n>" for any non-zero orphan bucket
	DuplicateAssgn int
	TagStatsDiff   []string // "<tagId>: pb=<a> pg=<b>"
	FilterDiff     []string // representative gallery filter id-set mismatches
	S3Checked      int
	S3Missing      []string
	S3Skipped      bool
}

// OK reports whether every hard check passed (S3 is soft when not configured).
func (r *VerifyResult) OK() bool {
	return len(r.CountMismatch) == 0 && len(r.Orphans) == 0 && r.DuplicateAssgn == 0 &&
		len(r.TagStatsDiff) == 0 && len(r.FilterDiff) == 0 && len(r.S3Missing) == 0
}

// Verify cross-checks the migrated Postgres against the source PB SQLite:
// per-table counts, FK-orphans, duplicate assignments, tag-statistics parity
// (old LIKE vs new jsonb), representative gallery-filter parity, and S3 HEADs.
// s3client may be nil — S3 checks are then soft-passed (skipped).
func Verify(ctx context.Context, pb *sql.DB, client *ent.Client, s3client *s3.S3Client) (*VerifyResult, error) {
	res := &VerifyResult{Counts: map[string][2]int{}}

	// 1. Per-table count parity (PB table -> Ent count). inferences is dropped.
	counts := []struct {
		table string
		pg    func() int
	}{
		{"roles", func() int { return client.Role.Query().CountX(ctx) }},
		{"users", func() int { return client.User.Query().CountX(ctx) }},
		{"projects", func() int { return client.Project.Query().CountX(ctx) }},
		{"cameras", func() int { return client.Camera.Query().CountX(ctx) }},
		{"time_offsets", func() int { return client.TimeOffset.Query().CountX(ctx) }},
		{"uploads", func() int { return client.Upload.Query().CountX(ctx) }},
		{"image_tags", func() int { return client.ImageTag.Query().CountX(ctx) }},
		{"project_assignments", func() int { return client.ProjectAssignment.Query().CountX(ctx) }},
		{"images", func() int { return client.Image.Query().CountX(ctx) }},
		{"image_tag_assignments", func() int { return client.ImageTagAssignment.Query().CountX(ctx) }},
	}
	for _, c := range counts {
		pbN, err := pbCount(pb, c.table)
		if err != nil {
			return nil, fmt.Errorf("pb count %s: %w", c.table, err)
		}
		pgN := c.pg()
		res.Counts[c.table] = [2]int{pbN, pgN}
		if pbN != pgN {
			res.CountMismatch = append(res.CountMismatch, fmt.Sprintf("%s: pb=%d pg=%d", c.table, pbN, pgN))
		}
	}

	// 2. FK-orphan scan via required-edge predicates (0 expected on both dialects).
	orphan := func(name string, n int) {
		if n > 0 {
			res.Orphans = append(res.Orphans, fmt.Sprintf("%s=%d", name, n))
		}
	}
	orphan("image.user", client.Image.Query().Where(image.Not(image.HasUser())).CountX(ctx))
	orphan("image.upload", client.Image.Query().Where(image.Not(image.HasUpload())).CountX(ctx))
	orphan("image.project", client.Image.Query().Where(image.Not(image.HasProject())).CountX(ctx))
	orphan("image.camera", client.Image.Query().Where(image.Not(image.HasCamera())).CountX(ctx))
	orphan("assignment.image", client.ImageTagAssignment.Query().Where(imagetagassignment.Not(imagetagassignment.HasImage())).CountX(ctx))
	orphan("assignment.imageTag", client.ImageTagAssignment.Query().Where(imagetagassignment.Not(imagetagassignment.HasImageTag())).CountX(ctx))
	orphan("camera.user", client.Camera.Query().Where(camera.Not(camera.HasUser())).CountX(ctx))
	orphan("upload.user", client.Upload.Query().Where(upload.Not(upload.HasUser())).CountX(ctx))
	orphan("upload.project", client.Upload.Query().Where(upload.Not(upload.HasProject())).CountX(ctx))
	orphan("upload.camera", client.Upload.Query().Where(upload.Not(upload.HasCamera())).CountX(ctx))
	orphan("timeoffset.camera", client.TimeOffset.Query().Where(timeoffset.Not(timeoffset.HasCamera())).CountX(ctx))
	orphan("projectassignment.user", client.ProjectAssignment.Query().Where(projectassignment.Not(projectassignment.HasUser())).CountX(ctx))
	orphan("projectassignment.project", client.ProjectAssignment.Query().Where(projectassignment.Not(projectassignment.HasProject())).CountX(ctx))
	orphan("projectassignment.role", client.ProjectAssignment.Query().Where(projectassignment.Not(projectassignment.HasRole())).CountX(ctx))

	// 3. Duplicate (image, imageTag) assignments.
	assigns := client.ImageTagAssignment.Query().AllX(ctx)
	seen := map[string]bool{}
	for _, a := range assigns {
		key := a.ImageID + "\x00" + a.ImageTagID
		if seen[key] {
			res.DuplicateAssgn++
		}
		seen[key] = true
	}

	// 4. Tag-statistics parity: old LIKE-on-imageTags count vs new jsonb (Go) count.
	// 5. uses the same per-image tag membership, so load images once.
	allImages := client.Image.Query().AllX(ctx)
	tagIDs := client.ImageTag.Query().IDsX(ctx)
	for _, tagID := range tagIDs {
		pbN, err := pbTagLikeCount(pb, tagID)
		if err != nil {
			return nil, fmt.Errorf("pb tag-like count %s: %w", tagID, err)
		}
		pgN := 0
		for _, im := range allImages {
			if contains(im.ImageTags, tagID) {
				pgN++
			}
		}
		if pbN != pgN {
			res.TagStatsDiff = append(res.TagStatsDiff, fmt.Sprintf("%s: pb=%d pg=%d", tagID, pbN, pgN))
		}
	}

	// 5. Representative gallery-filter parity (single-tag + two-tag AND).
	filters := representativeFilters(tagIDs, allImages)
	for _, f := range filters {
		pgIDs := map[string]bool{}
		for _, im := range allImages {
			if containsAll(im.ImageTags, f) {
				pgIDs[im.ID] = true
			}
		}
		pbIDs, err := pbFilterIDs(pb, f)
		if err != nil {
			return nil, fmt.Errorf("pb filter %v: %w", f, err)
		}
		if !sameSet(pbIDs, pgIDs) {
			res.FilterDiff = append(res.FilterDiff, fmt.Sprintf("%v: pb=%d pg=%d", f, len(pbIDs), len(pgIDs)))
		}
	}

	// 6. S3 HEAD sampling on migrated storageIds.
	if s3client == nil {
		res.S3Skipped = true
	} else {
		for _, im := range allImages {
			_, err := s3client.Client.StatObject(ctx, s3client.Options.Bucket, im.StorageId, minio.StatObjectOptions{})
			res.S3Checked++
			if err != nil {
				res.S3Missing = append(res.S3Missing, im.StorageId)
			}
		}
	}

	return res, nil
}

func pbCount(db *sql.DB, table string) (int, error) {
	var n int
	// table is a fixed literal from the counts list, not user input.
	err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n)
	return n, err
}

// pbTagLikeCount reproduces the legacy statistics query: a LIKE over the raw
// imageTags JSON text column.
func pbTagLikeCount(db *sql.DB, tagID string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT count(*) FROM images WHERE imageTags LIKE '%' || ? || '%'`, tagID).Scan(&n)
	return n, err
}

// pbFilterIDs returns the image ids matching an AND-list of tags using the old
// LIKE approach (intersect per tag).
func pbFilterIDs(db *sql.DB, tags []string) (map[string]bool, error) {
	if len(tags) == 0 {
		return map[string]bool{}, nil
	}
	where := make([]string, len(tags))
	args := make([]any, len(tags))
	for i, t := range tags {
		where[i] = "imageTags LIKE '%' || ? || '%'"
		args[i] = t
	}
	rows, err := db.Query("SELECT id FROM images WHERE "+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// representativeFilters picks a single-tag filter (the most-used tag) and a
// two-tag AND filter (a co-occurring pair) from the actual data, so the parity
// check exercises real id sets rather than empty ones.
func representativeFilters(tagIDs []string, images []*ent.Image) [][]string {
	var filters [][]string
	// Single-tag: the tag on the most images.
	best, bestN := "", -1
	for _, t := range tagIDs {
		n := 0
		for _, im := range images {
			if contains(im.ImageTags, t) {
				n++
			}
		}
		if n > bestN {
			best, bestN = t, n
		}
	}
	if best != "" {
		filters = append(filters, []string{best})
	}
	// Two-tag AND: first image carrying >=2 tags.
	for _, im := range images {
		if len(im.ImageTags) >= 2 {
			pair := []string{im.ImageTags[0], im.ImageTags[1]}
			filters = append(filters, pair)
			break
		}
	}
	return filters
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func containsAll(s, want []string) bool {
	for _, w := range want {
		if !contains(s, w) {
			return false
		}
	}
	return true
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// String renders a compact human-readable summary for the CLI.
func (r *VerifyResult) String() string {
	var b strings.Builder
	tables := make([]string, 0, len(r.Counts))
	for t := range r.Counts {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	for _, t := range tables {
		c := r.Counts[t]
		b.WriteString(fmt.Sprintf("  %-22s pb=%d pg=%d\n", t, c[0], c[1]))
	}
	fmt.Fprintf(&b, "count-mismatch=%d orphans=%v dup-assignments=%d tag-stats-diff=%d filter-diff=%d\n",
		len(r.CountMismatch), r.Orphans, r.DuplicateAssgn, len(r.TagStatsDiff), len(r.FilterDiff))
	if r.S3Skipped {
		b.WriteString("s3: skipped (no client configured)\n")
	} else {
		fmt.Fprintf(&b, "s3: checked=%d missing=%d\n", r.S3Checked, len(r.S3Missing))
	}
	fmt.Fprintf(&b, "OK=%v\n", r.OK())
	return b.String()
}
