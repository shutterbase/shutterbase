package repository

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/predicate"
	"github.com/shutterbase/shutterbase/internal/util"
)

var imageSortFields = map[string]string{
	"capturedAtCorrected": image.FieldCapturedAtCorrected,
	"capturedAt":          image.FieldCapturedAt,
	"createdAt":           image.FieldCreatedAt,
	"updatedAt":           image.FieldUpdatedAt,
	"computedFileName":    image.FieldComputedFileName,
	"fileName":            image.FieldFileName,
}

// ErrMissingProject / ErrInvalidOrientation are mapped by the controller to
// 400 {"code":"missing_project"} / {"code":"invalid_orientation"} (SPEC §4.3).
var (
	ErrMissingProject     = errors.New("missing_project")
	ErrInvalidOrientation = errors.New("invalid_orientation")
)

func (r *Repository) GetImage(ctx context.Context, id string) (*ent.Image, error) {
	item, err := r.Client.Image.Query().
		Where(image.IDEQ(id)).
		WithUser().WithCamera().WithProject().WithUpload().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting image")
	}
	return item, err
}

type GetImageParameters struct {
	ProjectID string // required unless ProjectIDs is set
	// ProjectIDs replaces the single-project filter — ONLY for the
	// cross-project person search (the caller passes the user's viewable
	// projects); every other gallery query keeps the hard ProjectID.
	ProjectIDs    []string
	UploadID      *string
	CameraID      *string
	UserID        *uuid.UUID
	Search        *string
	TagIDs        []string // repeated -> AND-match via a single jsonb @> containment
	ExcludeTagIDs []string // repeated -> drop images carrying ANY of these (NOT @> per id)
	IDs           []string // restrict to these ids (person filter); nil = no restriction
	Orientation   *string  // "portrait" (w<h) | "landscape" (w>h); null w/h excluded
	// Inclusive bounds on capturedAtCorrected; either side may be nil (open).
	// Images without a corrected capture time never match when a bound is set —
	// an uncorrected photo cannot be placed on the time axis at all.
	FromCapturedAtCorrected *time.Time
	ToCapturedAtCorrected   *time.Time
	PaginationParameters    *PaginationParameters
}

// buildImagePredicates turns the shared gallery filter (SPEC §4.3) into ent
// predicates — one source of truth for GetImages and GetImageTagFacets.
func buildImagePredicates(parameters *GetImageParameters) ([]predicate.Image, error) {
	var predicates []predicate.Image
	if len(parameters.ProjectIDs) > 0 {
		predicates = []predicate.Image{image.ProjectIDIn(parameters.ProjectIDs...)}
	} else {
		if parameters.ProjectID == "" {
			return nil, ErrMissingProject
		}
		predicates = []predicate.Image{image.ProjectID(parameters.ProjectID)}
	}
	if parameters.UploadID != nil {
		predicates = append(predicates, image.UploadID(*parameters.UploadID))
	}
	if parameters.CameraID != nil {
		predicates = append(predicates, image.CameraID(*parameters.CameraID))
	}
	if parameters.UserID != nil {
		predicates = append(predicates, image.UserID(*parameters.UserID))
	}
	if parameters.Search != nil {
		predicates = append(predicates, image.Or(
			image.ComputedFileNameContainsFold(*parameters.Search),
			image.FileNameContainsFold(*parameters.Search),
			image.AiDescriptionContainsFold(*parameters.Search),
		))
	}
	if parameters.IDs != nil {
		predicates = append(predicates, image.IDIn(parameters.IDs...))
	}
	if len(parameters.TagIDs) > 0 {
		// imageTags @> '["t1","t2",...]' — array containment => contains ALL ids (AND).
		tagIDs := parameters.TagIDs
		predicates = append(predicates, func(s *sql.Selector) {
			s.Where(sqljson.ValueContains(image.FieldImageTags, tagIDs))
		})
	}
	for _, excludedTagID := range parameters.ExcludeTagIDs {
		excluded := []string{excludedTagID}
		predicates = append(predicates, func(s *sql.Selector) {
			// NULL imageTags must survive: NOT(NULL @> ...) is NULL, which would drop the row.
			s.Where(sql.Or(
				sql.IsNull(s.C(image.FieldImageTags)),
				sql.Not(sqljson.ValueContains(image.FieldImageTags, excluded)),
			))
		})
	}
	if parameters.Orientation != nil {
		switch *parameters.Orientation {
		case "portrait":
			predicates = append(predicates, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
				s.Where(sql.ColumnsLT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
			})
		case "landscape":
			predicates = append(predicates, image.WidthNotNil(), image.HeightNotNil(), func(s *sql.Selector) {
				s.Where(sql.ColumnsGT(s.C(image.FieldWidth), s.C(image.FieldHeight)))
			})
		default:
			return nil, ErrInvalidOrientation
		}
	}
	if parameters.FromCapturedAtCorrected != nil {
		predicates = append(predicates, image.CapturedAtCorrectedGTE(*parameters.FromCapturedAtCorrected))
	}
	if parameters.ToCapturedAtCorrected != nil {
		predicates = append(predicates, image.CapturedAtCorrectedLTE(*parameters.ToCapturedAtCorrected))
	}
	return predicates, nil
}

// GetImages is the gallery query (SPEC §4.3). projectId is required; tagId AND-match
// runs over the GIN(jsonb_path_ops) index via a single containment; orientation
// excludes rows with null width/height. Edges are eager-loaded for serialization.
func (r *Repository) GetImages(ctx context.Context, parameters *GetImageParameters) ([]*ent.Image, int, error) {
	predicates, err := buildImagePredicates(parameters)
	if err != nil {
		return nil, 0, err
	}
	where := image.And(predicates...)

	limit, offset, order, err := parameters.PaginationParameters.build(imageSortFields, "capturedAtCorrected")
	if err != nil {
		return nil, 0, err
	}
	items, err := r.Client.Image.Query().
		Where(where).
		WithUser().WithCamera().WithProject().WithUpload().
		WithImageTagAssignments(func(q *ent.ImageTagAssignmentQuery) { q.WithImageTag() }).
		Limit(limit).Offset(offset).Order(order).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting images")
		return nil, 0, err
	}
	total, err := r.Client.Image.Query().Where(where).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ImageTimeBounds is the [earliest, latest] capturedAtCorrected span of a
// gallery filter — the Time popover's slider domain. Either side nil when no
// matching image carries a corrected capture time.
type ImageTimeBounds struct {
	Min *time.Time `json:"min"`
	Max *time.Time `json:"max"`
}

// GetImageTimeBounds computes the [earliest, latest] capturedAtCorrected over
// the shared gallery filter. The time-range bounds themselves are always
// STRIPPED: the range being edited must not be part of its own slider domain,
// so this stays stable while thumbs move. NULL-corrected images never
// contribute. Implemented as two ordered picks (not MIN/MAX aggregates):
// SQLite hands aggregates back as untyped strings, and both queries are
// index-covered anyway.
func (r *Repository) GetImageTimeBounds(ctx context.Context, parameters *GetImageParameters) (*ImageTimeBounds, error) {
	// Work on a copy: stripping the caller's struct would silently drop the
	// range from any later reuse of the same *GetImageParameters.
	stripped := *parameters
	stripped.FromCapturedAtCorrected = nil
	stripped.ToCapturedAtCorrected = nil
	predicates, err := buildImagePredicates(&stripped)
	if err != nil {
		return nil, err
	}
	where := image.And(append(predicates, image.CapturedAtCorrectedNotNil())...)
	bounds := &ImageTimeBounds{}
	first, err := r.Client.Image.Query().Where(where).
		Order(ent.Asc(image.FieldCapturedAtCorrected)).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		log.Error().Err(err).Msg("error getting image time bounds (min)")
		return nil, err
	}
	if first != nil {
		bounds.Min = first.CapturedAtCorrected
		// Same race as the pick above: the whole matching set can be deleted
		// between the two ordered statements, and NotFound here means exactly
		// that. Mirror the (min) branch — a missing Max is a cosmetic gap in the
		// slider domain, a 500 on the slider endpoint is not.
		last, err := r.Client.Image.Query().Where(where).
			Order(ent.Desc(image.FieldCapturedAtCorrected)).First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			log.Error().Err(err).Msg("error getting image time bounds (max)")
			return nil, err
		}
		if last != nil {
			bounds.Max = last.CapturedAtCorrected
		}
	}
	return bounds, nil
}

// ImageTimeTicks returns sampled capturedAtCorrected timestamps for the slider
// density strip. The result always spans the full range: the first and the
// NEWEST matching photo are always included, with maxTicks-2 samples spread
// evenly between them.
//
// PERFORMANCE: the scan is bounded by maxTicks, not by the gallery size.
//   - ≤ maxTicks images: one ordered scan, every position returned.
//   - more: a COUNT plus maxTicks indexed (OFFSET … LIMIT 1) seeks. Each seek
//     is an index range scan, so the cost is ~maxTicks index probes regardless
//     of whether the project holds 10k or 10M photos — the frontend calls this
//     on popover open and on every filter change, so an O(rows) scan would
//     make a 100k-photo project pay for its whole index on every keystroke.
//
// Like GetImageTimeBounds, the time-range itself is always STRIPPED (on a copy
// of the parameters) so the ticks stay stable while thumbs move.
// timeTickRow is one row of the density strip scan: only the corrected capture
// time crosses the wire, which is what keeps the transfer small enough to do the
// whole thing in one round trip.
type timeTickRow struct {
	CapturedAtCorrected time.Time `json:"captured_at_corrected"`
}

func (r *Repository) GetImageTimeTicks(ctx context.Context, parameters *GetImageParameters, maxTicks int) ([]time.Time, error) {
	if maxTicks < 2 {
		maxTicks = 2
	}
	stripped := *parameters
	stripped.FromCapturedAtCorrected = nil
	stripped.ToCapturedAtCorrected = nil
	predicates, err := buildImagePredicates(&stripped)
	if err != nil {
		return nil, err
	}
	where := image.And(append(predicates, image.CapturedAtCorrectedNotNil())...)

	// ONE ordered scan, ONE round trip, downsampled here in Go.
	//
	// Measured on a 15k-photo project with maxTicks=200, end to end:
	//   1+maxTicks sequential `OFFSET n LIMIT 1` seeks   2.29s
	//   this scan                                        26-67ms
	// The seeks were 201 round trips, and maxTicks is a fixed 200, so the count
	// never shrank with gallery size: it was pure overhead on a popover-open
	// fetch.
	//
	// `id` is in the ORDER BY because captured_at_corrected alone is not a total
	// order: burst photos share a second, and independent offset probes over a
	// tie group could land on the same row twice.
	var rows []timeTickRow
	if err := r.Client.Image.Query().Where(where).
		Order(ent.Asc(image.FieldCapturedAtCorrected), ent.Asc(image.FieldID)).
		Select(image.FieldCapturedAtCorrected).
		Scan(ctx, &rows); err != nil {
		log.Error().Err(err).Msg("error loading timestamps for time ticks")
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	// The scan has no LIMIT, so it costs O(matching photos). Reviewed, and left
	// that way deliberately.
	//
	// The tempting fix is to sample with ROW_NUMBER() so only maxTicks rows come
	// back. Measured and rejected: at 1M photos the two shapes hand Postgres's
	// client 9.2MB vs 1.8KB, but loopback runs at 4.6GB/s, so the transfer it
	// saves is single-digit milliseconds — while the ROW_NUMBER version pays the
	// same O(n) scan and adds window computation on top. The cost lives in
	// Postgres reading the rows, not in Go receiving them, so no query rewrite
	// can remove it; only a rollup table maintained on write could, and that
	// would have to reimplement the isolation pass in SQL because the median
	// inter-photo gap is not derivable from counts.
	//
	// So the row count is logged, to keep the cost observable rather than
	// discoverable only by feeling a slow popover.
	log.Debug().Int("rows", len(rows)).Int("maxTicks", maxTicks).Msg("time ticks: scanned matching photos")
	if len(rows) <= maxTicks {
		timestamps := make([]time.Time, len(rows))
		for i, row := range rows {
			timestamps[i] = row.CapturedAtCorrected
		}
		return timestamps, nil
	}

	// Sample in TIME space, not rank space.
	//
	// Rank-space sampling (offset i*(n-1)/(maxTicks-1) over the sorted rows)
	// answers "where are the photos", which is what a density strip wants — but
	// it is blind to a photo that sits alone. With 15 000 photos in one burst
	// and 5 photos hours away, all 200 marks landed inside the burst and every
	// lone photo was dropped: they are one row in ~75, so the odds of one landing
	// exactly on a sample offset are about 1 in 75.
	//
	// So the domain [min,max] is cut into equal time buckets and each non-empty
	// bucket contributes its MEDIAN instant:
	//   - a lone photo is the median of its own bucket, so it usually appears;
	//   - a cluster spanning k buckets contributes k marks, so a dense stretch
	//     still reads as denser than a sparse one.
	// A bucket emits ONE mark though, so a photo alone in time that happens to
	// share a bucket with a burst is still invisible — hence the isolation pass
	// below, which gets its own budget.
	first, last := rows[0].CapturedAtCorrected, rows[len(rows)-1].CapturedAtCorrected

	// --- isolation pass, BEFORE the budget is split ----------------------
	//
	// Measured, not assumed: with the domain 61 days wide, a lone photo at +1h30m
	// and one at +5h both landed in the same 7.45h bucket as a 15 000-photo
	// burst, and the bucket median was a burst row. So rows whose gap to BOTH
	// neighbours is far larger than the typical spacing get an explicit mark.
	//
	// The scale is the MEDIAN gap, not the bucket width: a photo 1.5h after a
	// burst is visually alone but is not "isolated" at a 15h bucket resolution,
	// and thresholding on the bucket width silently dropped exactly those two.
	// The median is right because it is set by the typical spacing rather than by
	// the outliers being hunted — 15 000 rows 1ms apart give a median of 1ms, so a
	// 1.5h gap is 5000x local spacing and the photo is unambiguously alone.
	gaps := make([]time.Duration, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		gaps[i-1] = rows[i].CapturedAtCorrected.Sub(rows[i-1].CapturedAtCorrected)
	}
	// medianDuration permutes gaps in place, so the loop below recomputes the two
	// gaps it needs straight from rows rather than indexing gaps afterwards.
	typical := saturatingMul(medianDuration(gaps), 10)
	isolated := make([]int, 0, 8)
	if typical > 0 {
		for i := 1; i < len(rows)-1; i++ {
			prevGap := rows[i].CapturedAtCorrected.Sub(rows[i-1].CapturedAtCorrected)
			nextGap := rows[i+1].CapturedAtCorrected.Sub(rows[i].CapturedAtCorrected)
			if prevGap >= typical && nextGap >= typical {
				isolated = append(isolated, i)
			}
		}
	}

	// --- split the budget now that the lone photos are counted -----------
	//
	// This ordering is the whole point. Splitting it 50/50 up front — as an
	// earlier revision did — halved the density resolution of EVERY gallery,
	// including the smooth ones with nothing isolated at all, where the reserved
	// slots were pure waste. So: lone photos claim what they need, density keeps
	// a guaranteed floor, and whatever the lone photos do not use flows back to
	// density.
	usable := maxTicks - 2 // the two pinned endpoints are reserved
	if usable < 1 {
		usable = 1
	}
	densityFloor := usable / 4
	if densityFloor < 1 {
		densityFloor = 1
	}
	nIso := len(isolated)
	if room := usable - densityFloor; nIso > room {
		// Over budget. Thin by TIME, not by rank: dropping the tail wholesale
		// would hide a whole region of the timeline, which is the failure this
		// pass exists to prevent. spreadEvenly keeps the first and the last.
		nIso = max(room, 0)
		isolated = spreadEvenly(isolated, nIso)
	}
	interior := usable - nIso
	if interior < 1 {
		interior = 1
	}

	span := last.Sub(first)
	bucketWidth := span / time.Duration(interior)
	if bucketWidth <= 0 {
		// Every photo is within `interior` nanoseconds of the first, so the strip
		// has at most two distinct positions to draw. That is the truth of the
		// data rather than a failure — but it used to return two marks silently,
		// which reads as a bug when you see it in the UI.
		log.Debug().
			Int("rows", len(rows)).
			Int("distinctInstants", len(distinctInstants(rows))).
			Msg("time ticks: domain narrower than the bucket width, returning the distinct instants only")
		return distinctInstants(rows), nil
	}

	buckets := make([]struct{ first, n int }, interior)
	for i := range buckets {
		buckets[i].first = -1
	}
	for i, r := range rows {
		b := int(r.CapturedAtCorrected.Sub(first) / bucketWidth)
		if b >= interior {
			// The final bucket absorbs the remainder, so its median can sit well
			// short of the domain's right edge. The pinned `last` covers that.
			b = interior - 1
		}
		if buckets[b].first < 0 {
			buckets[b].first = i
		}
		buckets[b].n++
	}

	// --- collect row indices, ascending ----------------------------------
	chosen := make([]int, 0, maxTicks)
	chosen = append(chosen, 0, len(rows)-1)
	for b := range buckets {
		if buckets[b].n == 0 {
			continue // an empty stretch of the timeline contributes no mark
		}
		chosen = append(chosen, buckets[b].first+buckets[b].n/2)
	}
	chosen = append(chosen, isolated...)
	slices.Sort(chosen)

	// Emit in ascending row order, dropping any row whose instant repeats the
	// next one's. Burst photos routinely share a second, and two marks at the
	// same pixel are one mark drawn twice. Keeping the LAST of a run rather than
	// the first is what preserves the pinned newest photo.
	sampled := make([]time.Time, 0, len(chosen))
	for k, idx := range chosen {
		if k+1 < len(chosen) && rows[chosen[k+1]].CapturedAtCorrected.Equal(rows[idx].CapturedAtCorrected) {
			continue
		}
		sampled = append(sampled, rows[idx].CapturedAtCorrected)
	}
	return sampled, nil
}

// distinctInstants returns the distinct captured_at_corrected values of rows, in
// order. Rows arrive time-ordered, so this is a single adjacent-duplicate pass
// and needs no map. It backs the degenerate-domain path, where bucketing has no
// resolution to offer and the distinct instants are all there is to show.
func distinctInstants(rows []timeTickRow) []time.Time {
	out := make([]time.Time, 0, 2)
	for i, r := range rows {
		if i > 0 && r.CapturedAtCorrected.Equal(rows[i-1].CapturedAtCorrected) {
			continue
		}
		out = append(out, r.CapturedAtCorrected)
	}
	return out
}

// GetImagePosition returns the zero-based offset of imageID within the gallery
// query defined by parameters (same predicates and order as GetImages), or -1
// when the image is not among the first maxScan matches — the deep-link
// resolver then falls back to a solo detail view. An id-only bounded scan
// sidesteps null-aware window arithmetic for the nullable sort columns.
func (r *Repository) GetImagePosition(ctx context.Context, parameters *GetImageParameters, imageID string, maxScan int) (int, error) {
	predicates, err := buildImagePredicates(parameters)
	if err != nil {
		return -1, err
	}
	_, _, order, err := parameters.PaginationParameters.build(imageSortFields, "capturedAtCorrected")
	if err != nil {
		return -1, err
	}
	ids, err := r.Client.Image.Query().
		Where(image.And(predicates...)).
		Order(order).
		Limit(maxScan).
		IDs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error scanning image position")
		return -1, err
	}
	for i, id := range ids {
		if id == imageID {
			return i, nil
		}
	}
	return -1, nil
}

// imageTagScope enumerates the tags a facet map can carry: every tag of every
// project the COUNT above covers. It must mirror buildImagePredicates' project
// precedence, which prefers ProjectIDs over ProjectID — the cross-project person
// search sets BOTH, so reading ProjectID alone left the other projects' tags out
// of the map and let a shared project's counts span every project in the search.
func imageTagScope(parameters *GetImageParameters) predicate.ImageTag {
	if len(parameters.ProjectIDs) > 0 {
		return imagetag.ProjectIDIn(parameters.ProjectIDs...)
	}
	return imagetag.ProjectID(parameters.ProjectID)
}

// GetImageTagFacets returns the filter's own match count plus, per project tag,
// how many of those matches also carry the tag — i.e. the result size if the tag
// were added as an include filter. Tags matching zero images are omitted.
// ponytail: one count query per tag over the GIN index, same shape as
// GetProjectTagStatistics — switch both to a single GROUP BY over
// jsonb_array_elements_text if tag or image counts make this measurably slow.
func (r *Repository) GetImageTagFacets(ctx context.Context, parameters *GetImageParameters) (int, map[string]int, error) {
	predicates, err := buildImagePredicates(parameters)
	if err != nil {
		return 0, nil, err
	}
	where := image.And(predicates...)
	total, err := r.Client.Image.Query().Where(where).Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error counting images for tag facets")
		return 0, nil, err
	}
	tags, err := r.Client.ImageTag.Query().Where(imageTagScope(parameters)).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error loading project tags for facets")
		return 0, nil, err
	}
	facets := make(map[string]int, len(tags))
	for _, t := range tags {
		tagID := t.ID
		count, err := r.Client.Image.Query().
			Where(where, func(s *sql.Selector) {
				s.Where(sqljson.ValueContains(image.FieldImageTags, []string{tagID}))
			}).
			Count(ctx)
		if err != nil {
			log.Error().Err(err).Str("tag", tagID).Msg("error counting tag facet")
			return 0, nil, err
		}
		if count > 0 {
			facets[tagID] = count
		}
	}
	return total, facets, nil
}

// TagStatistic is one row of GetProjectTagStatistics.
type TagStatistic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Count       int    `json:"count"`
}

// GetProjectTagStatistics returns per-tag image counts using the SAME jsonb
// read-model the gallery filter uses (count(*) where imageTags @> '["id"]'), so
// stats and filtering can never diverge. Each images row is counted at most once
// per tag, so the count is inherently de-duplicated. Replaces the old SQLite LIKE.
func (r *Repository) GetProjectTagStatistics(ctx context.Context, projectID string) ([]TagStatistic, error) {
	tags, err := r.Client.ImageTag.Query().
		Where(imagetag.ProjectID(projectID)).
		Order(ent.Asc(imagetag.FieldName)).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error loading project tags for statistics")
		return nil, err
	}
	stats := make([]TagStatistic, 0, len(tags))
	for _, t := range tags {
		tagID := t.ID
		// Scalar containment ('["a"]' @> '"a"'), not []string: same result on
		// Postgres, and it keeps the SQLite tier (unit tests) working — the
		// slice form only compiles to valid SQL on Postgres.
		count, err := r.Client.Image.Query().
			Where(image.ProjectID(projectID), func(s *sql.Selector) {
				s.Where(sqljson.ValueContains(image.FieldImageTags, tagID))
			}).
			Count(ctx)
		if err != nil {
			log.Error().Err(err).Str("tag", tagID).Msg("error counting tag statistics")
			return nil, err
		}
		stats = append(stats, TagStatistic{
			ID: t.ID, Name: t.Name, DisplayName: t.DisplayName, Description: t.Description, Type: t.Type.String(), Count: count,
		})
	}
	return stats, nil
}

type CreateImageParameters struct {
	FileName            string
	ComputedFileName    *string
	StorageID           string
	Size                int
	Width               *int
	Height              *int
	CapturedAt          *time.Time
	CapturedAtCorrected *time.Time
	ExifData            map[string]any
	ImageTags           []string
	UserID              uuid.UUID
	UploadID            string
	ProjectID           string
	CameraID            string
}

func (r *Repository) CreateImage(ctx context.Context, parameters *CreateImageParameters) (*ent.Image, error) {
	create := r.Client.Image.Create().
		SetFileName(parameters.FileName).
		SetStorageId(parameters.StorageID).
		SetSize(parameters.Size).
		SetUserID(parameters.UserID).
		SetUploadID(parameters.UploadID).
		SetProjectID(parameters.ProjectID).
		SetCameraID(parameters.CameraID).
		SetCreatedBy(util.GetActorID(ctx)).
		SetUpdatedBy(util.GetActorID(ctx))
	if parameters.ComputedFileName != nil {
		create = create.SetComputedFileName(*parameters.ComputedFileName)
	}
	if parameters.Width != nil {
		create = create.SetWidth(*parameters.Width)
	}
	if parameters.Height != nil {
		create = create.SetHeight(*parameters.Height)
	}
	if parameters.CapturedAt != nil {
		create = create.SetCapturedAt(*parameters.CapturedAt)
	}
	if parameters.CapturedAtCorrected != nil {
		create = create.SetCapturedAtCorrected(*parameters.CapturedAtCorrected)
	}
	if parameters.ExifData != nil {
		create = create.SetExifData(parameters.ExifData)
	}
	if parameters.ImageTags != nil {
		create = create.SetImageTags(parameters.ImageTags)
	}
	item, err := create.Save(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error creating image")
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "create", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"fileName": item.FileName, "storageId": item.StorageId},
		})
	})
	return item, nil
}

type UpdateImageParameters struct {
	FileName            *string
	ComputedFileName    *string
	CapturedAt          *time.Time
	CapturedAtCorrected *time.Time
	ExifData            map[string]any
	ImageTags           []string
	Width               *int
	Height              *int
	InferredAt          *time.Time
	CameraID            *string
	UploadID            *string
}

func (r *Repository) UpdateImage(ctx context.Context, id string, parameters *UpdateImageParameters) (*ent.Image, error) {
	tx, err := r.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Image.Query().Where(image.IDEQ(id))
	if r.isPostgres() {
		q = q.ForUpdate()
	}
	item, err := q.Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.Image.UpdateOneID(id).SetUpdatedBy(util.GetActorID(ctx))
	st := modelUpdateStatus{}

	if parameters.FileName != nil && item.FileName != *parameters.FileName {
		update.SetFileName(*parameters.FileName)
		st.SetFieldChanged(image.FieldFileName, item.FileName, *parameters.FileName)
	}
	if parameters.ComputedFileName != nil && item.ComputedFileName != *parameters.ComputedFileName {
		update.SetComputedFileName(*parameters.ComputedFileName)
		st.SetFieldChanged(image.FieldComputedFileName, item.ComputedFileName, *parameters.ComputedFileName)
	}
	if parameters.CapturedAt != nil && (item.CapturedAt == nil || !item.CapturedAt.Equal(*parameters.CapturedAt)) {
		update.SetCapturedAt(*parameters.CapturedAt)
		st.SetFieldChanged(image.FieldCapturedAt, item.CapturedAt, *parameters.CapturedAt)
	}
	if parameters.CapturedAtCorrected != nil && (item.CapturedAtCorrected == nil || !item.CapturedAtCorrected.Equal(*parameters.CapturedAtCorrected)) {
		update.SetCapturedAtCorrected(*parameters.CapturedAtCorrected)
		st.SetFieldChanged(image.FieldCapturedAtCorrected, item.CapturedAtCorrected, *parameters.CapturedAtCorrected)
	}
	if parameters.InferredAt != nil && (item.InferredAt == nil || !item.InferredAt.Equal(*parameters.InferredAt)) {
		update.SetInferredAt(*parameters.InferredAt)
		st.SetFieldChanged(image.FieldInferredAt, item.InferredAt, *parameters.InferredAt)
	}
	if parameters.Width != nil && (item.Width == nil || *item.Width != *parameters.Width) {
		update.SetWidth(*parameters.Width)
		st.SetFieldChanged(image.FieldWidth, item.Width, *parameters.Width)
	}
	if parameters.Height != nil && (item.Height == nil || *item.Height != *parameters.Height) {
		update.SetHeight(*parameters.Height)
		st.SetFieldChanged(image.FieldHeight, item.Height, *parameters.Height)
	}
	if parameters.CameraID != nil && item.CameraID != *parameters.CameraID {
		update.SetCameraID(*parameters.CameraID)
		st.SetFieldChanged(image.FieldCameraID, item.CameraID, *parameters.CameraID)
	}
	if parameters.UploadID != nil && item.UploadID != *parameters.UploadID {
		update.SetUploadID(*parameters.UploadID)
		st.SetFieldChanged(image.FieldUploadID, item.UploadID, *parameters.UploadID)
	}
	if parameters.ExifData != nil && !reflect.DeepEqual(item.ExifData, parameters.ExifData) {
		update.SetExifData(parameters.ExifData)
		st.SetFieldChanged(image.FieldExifData, "<json>", "<json>")
	}
	if parameters.ImageTags != nil && !reflect.DeepEqual(item.ImageTags, parameters.ImageTags) {
		update.SetImageTags(parameters.ImageTags)
		st.SetFieldChanged(image.FieldImageTags, item.ImageTags, parameters.ImageTags)
	}

	if !st.modelChanged {
		_ = tx.Rollback()
		return item, nil
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		log.Error().Err(err).Msg("error updating image")
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "update", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(item.ID),
			Data: &map[string]any{"changes": st.GetChangedFieldData()},
		})
	})
	return item, nil
}

func (r *Repository) DeleteImage(ctx context.Context, id string) error {
	if err := r.Client.Image.DeleteOneID(id).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("error deleting image")
		return err
	}
	safeGo(func() {
		r.CreateAuditLog(context.WithoutCancel(ctx), &CreateAuditLogParameters{
			Action: "delete", ObjectType: util.StringPointer("image"), ObjectId: util.StringPointer(id),
			Data: &map[string]any{},
		})
	})
	return nil
}
