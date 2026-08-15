package repository

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

// Project statistics for the dashboard. Day buckets are computed in Go in the
// event timezone (`loc`) rather than SQL: date_trunc would split Postgres and
// the SQLite test tier, and SQL grouping would bucket on UTC days, not the
// wall-clock days photographers think in.

type StatTotals struct {
	Images        int   `json:"images"`
	Photographers int   `json:"photographers"`
	ManualTags    int   `json:"manualTags"`
	AiTags        int   `json:"aiTags"`
	StorageBytes  int64 `json:"storageBytes"`
	// Images without capturedAtCorrected (legacy rows) — counted here, absent
	// from the day buckets.
	UnbucketedImages int `json:"unbucketedImages"`
}

type StatDay struct {
	Date   string         `json:"date"` // "2006-01-02" in the event timezone
	Total  int            `json:"total"`
	ByUser map[string]int `json:"byUser"`
	ByHour [24]int        `json:"byHour"`
}

type StatAssignmentDay struct {
	Date   string `json:"date"`
	Manual int    `json:"manual"`
	AI     int    `json:"ai"`
}

type StatPhotographer struct {
	ID           string `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	CopyrightTag string `json:"copyrightTag"`
	ImageCount   int    `json:"imageCount"`
}

type StatAiStatus struct {
	Done      int `json:"done"`
	InFlight  int `json:"inFlight"`
	Error     int `json:"error"`
	NotQueued int `json:"notQueued"`
}

type StatUploadStates struct {
	Open     int `json:"open"`
	Ready    int `json:"ready"`
	Reviewed int `json:"reviewed"`
}

type ProjectStatistics struct {
	Totals            StatTotals          `json:"totals"`
	Days              []StatDay           `json:"days"` // sorted asc, sparse (UI fills gaps)
	AssignmentsPerDay []StatAssignmentDay `json:"assignmentsPerDay"`
	Photographers     []StatPhotographer  `json:"photographers"` // sorted by imageCount desc
	AiStatus          StatAiStatus        `json:"aiStatus"`
	UploadStates      StatUploadStates    `json:"uploadStates"`
	Tags              []TagStatistic      `json:"tags"`
}

// GetProjectStatistics aggregates the dashboard payload. One image scan powers
// day/hour/photographer buckets, storage and totals; one assignment scan powers
// the manual-vs-AI tagging series; status/state counts come from grouped counts.
func (r *Repository) GetProjectStatistics(ctx context.Context, projectID string, loc *time.Location) (*ProjectStatistics, error) {
	stats := &ProjectStatistics{
		Days:              []StatDay{},
		AssignmentsPerDay: []StatAssignmentDay{},
		Photographers:     []StatPhotographer{},
	}

	images, err := r.Client.Image.Query().
		Where(image.ProjectID(projectID)).
		Select(image.FieldCapturedAtCorrected, image.FieldUserID, image.FieldSize).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error scanning images for statistics")
		return nil, err
	}

	days := map[string]*StatDay{}
	userCounts := map[uuid.UUID]int{} // photographer -> image count, all images
	for _, img := range images {
		stats.Totals.Images++
		stats.Totals.StorageBytes += int64(img.Size)
		userCounts[img.UserID]++
		if img.CapturedAtCorrected == nil {
			stats.Totals.UnbucketedImages++
			continue
		}
		local := img.CapturedAtCorrected.In(loc)
		key := local.Format("2006-01-02")
		day, ok := days[key]
		if !ok {
			day = &StatDay{Date: key, ByUser: map[string]int{}}
			days[key] = day
		}
		day.Total++
		day.ByUser[img.UserID.String()]++
		day.ByHour[local.Hour()]++
	}
	for _, day := range days {
		stats.Days = append(stats.Days, *day)
	}
	sort.Slice(stats.Days, func(i, j int) bool { return stats.Days[i].Date < stats.Days[j].Date })

	assignments, err := r.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.HasImageWith(image.ProjectID(projectID))).
		Select(imagetagassignment.FieldCreatedAt, imagetagassignment.FieldType).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error scanning tag assignments for statistics")
		return nil, err
	}
	assignmentDays := map[string]*StatAssignmentDay{}
	for _, a := range assignments {
		var manual bool
		switch a.Type {
		case imagetagassignment.TypeManual:
			manual = true
			stats.Totals.ManualTags++
		case imagetagassignment.TypeInferred:
			stats.Totals.AiTags++
		default: // default | scheduled: automatic, not part of the effort series
			continue
		}
		key := a.CreatedAt.In(loc).Format("2006-01-02")
		day, ok := assignmentDays[key]
		if !ok {
			day = &StatAssignmentDay{Date: key}
			assignmentDays[key] = day
		}
		if manual {
			day.Manual++
		} else {
			day.AI++
		}
	}
	for _, day := range assignmentDays {
		stats.AssignmentsPerDay = append(stats.AssignmentsPerDay, *day)
	}
	sort.Slice(stats.AssignmentsPerDay, func(i, j int) bool {
		return stats.AssignmentsPerDay[i].Date < stats.AssignmentsPerDay[j].Date
	})

	if err := r.fillPhotographers(ctx, stats, userCounts); err != nil {
		return nil, err
	}

	var aiRows []struct {
		AiStatus string `json:"ai_status"`
		Count    int    `json:"count"`
	}
	if err := r.Client.Image.Query().
		Where(image.ProjectID(projectID), image.AiStatusNotNil()).
		GroupBy(image.FieldAiStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &aiRows); err != nil {
		log.Error().Err(err).Msg("error counting AI statuses for statistics")
		return nil, err
	}
	queued := 0
	for _, row := range aiRows {
		queued += row.Count
		switch row.AiStatus {
		case "done":
			stats.AiStatus.Done += row.Count
		case "error":
			stats.AiStatus.Error += row.Count
		default: // pending | processing
			stats.AiStatus.InFlight += row.Count
		}
	}
	stats.AiStatus.NotQueued = stats.Totals.Images - queued

	var uploadRows []struct {
		State string `json:"state"`
		Count int    `json:"count"`
	}
	if err := r.Client.Upload.Query().
		Where(upload.ProjectID(projectID)).
		GroupBy(upload.FieldState).
		Aggregate(ent.Count()).
		Scan(ctx, &uploadRows); err != nil {
		log.Error().Err(err).Msg("error counting upload states for statistics")
		return nil, err
	}
	for _, row := range uploadRows {
		switch row.State {
		case "open":
			stats.UploadStates.Open = row.Count
		case "ready":
			stats.UploadStates.Ready = row.Count
		case "reviewed":
			stats.UploadStates.Reviewed = row.Count
		}
	}

	stats.Tags, err = r.GetProjectTagStatistics(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// fillPhotographers resolves the per-user image counts to named entries, sorted
// by image count desc.
func (r *Repository) fillPhotographers(ctx context.Context, stats *ProjectStatistics, userCounts map[uuid.UUID]int) error {
	if len(userCounts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(userCounts))
	for id := range userCounts {
		ids = append(ids, id)
	}
	users, err := r.Client.User.Query().Where(user.IDIn(ids...)).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error loading photographers for statistics")
		return err
	}
	for _, u := range users {
		stats.Photographers = append(stats.Photographers, StatPhotographer{
			ID:           u.ID.String(),
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			CopyrightTag: u.CopyrightTag,
			ImageCount:   userCounts[u.ID],
		})
	}
	sort.Slice(stats.Photographers, func(i, j int) bool {
		return stats.Photographers[i].ImageCount > stats.Photographers[j].ImageCount
	})
	stats.Totals.Photographers = len(stats.Photographers)
	return nil
}
